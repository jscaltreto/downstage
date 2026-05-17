// Package diff implements a structural diff between two parsed Downstage
// documents. It produces a sequence of hunks (Equal, Insert, Delete, Modify)
// aligned to the renderer's block-level walk order, which lets downstream
// callers (internal/revisions) compute revision regions that the PDF renderer
// can emit.
//
// The flattened block stream produced by FlattenedBlocks is the unit of
// comparison: a Section is split into a SectionHeader block followed by the
// blocks for its ordered children. Songs and verse blocks are atoms — their
// inner content is folded into the parent fingerprint so that any edit inside
// shows up as a single change.
package diff

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/token"
)

// BlockKind enumerates the diffable block kinds. Each kind corresponds to a
// node the PDF renderer emits as a top-level visual block.
type BlockKind int

const (
	BlockSectionHeader BlockKind = iota
	BlockSectionLine
	BlockDialogue
	BlockDualDialogue
	BlockStageDirection
	BlockCallout
	BlockSong
	BlockVerseBlock
	BlockPageBreak
)

// Block is one element in the flattened diffable stream.
type Block struct {
	Kind BlockKind
	// Fingerprint is the content-derived hash used by the diff algorithm.
	// Two blocks with the same fingerprint are considered semantically equal
	// for revision-page purposes.
	Fingerprint string
	// Path is the block's location in the source document, expressed as a
	// chain of indices: doc.Body[Path[0]] → Section.OrderedItems[Path[1]] →
	// (and so on). The first element of Path identifies the body slot.
	Path []int
	// SectionPath records the titles of the enclosing sections in
	// outermost-first order, so that callers can build a "ACT II — SCENE 3"
	// style context heading without re-walking the AST.
	SectionPath []string
	// Source is the original source range of the block, useful for
	// diagnostics and asterisk-targeting at the line level.
	Source token.Range
	// Node points back to the original AST node for the block. For
	// BlockSectionLine it is nil and Line is populated instead.
	Node ast.Node
	// Line is the source line for BlockSectionLine blocks.
	Line *ast.SectionLine
}

// FlattenedBlocks returns the document's diffable block stream in the order
// the PDF renderer walks the AST. It mirrors the structure of
// internal/render.walkNode at the block level.
//
// The character canonicalizer is consulted when fingerprinting dialogue cues
// so that an alias rename does not show up as a content change. Pass nil to
// fingerprint character names literally.
func FlattenedBlocks(doc *ast.Document, canonName func(string) string) []Block {
	if doc == nil {
		return nil
	}
	w := &flattener{canonName: canonName}
	for i, node := range doc.Body {
		w.walk(node, []int{i}, nil)
	}
	return w.blocks
}

type flattener struct {
	blocks    []Block
	canonName func(string) string
}

func (f *flattener) walk(node ast.Node, path []int, sectionPath []string) {
	switch n := node.(type) {
	case *ast.Section:
		f.emitSection(n, path, sectionPath)
	case *ast.DualDialogue:
		f.emit(Block{
			Kind:        BlockDualDialogue,
			Fingerprint: fingerprintDualDialogue(n, f.canonName),
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.Dialogue:
		f.emit(Block{
			Kind:        BlockDialogue,
			Fingerprint: fingerprintDialogue(n, f.canonName),
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.StageDirection:
		f.emit(Block{
			Kind:        BlockStageDirection,
			Fingerprint: fingerprintStageDirection(n),
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.Callout:
		f.emit(Block{
			Kind:        BlockCallout,
			Fingerprint: fingerprintCallout(n),
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.Song:
		f.emit(Block{
			Kind:        BlockSong,
			Fingerprint: fingerprintSong(n, f.canonName),
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.VerseBlock:
		f.emit(Block{
			Kind:        BlockVerseBlock,
			Fingerprint: fingerprintVerseBlock(n),
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.PageBreak:
		f.emit(Block{
			Kind:        BlockPageBreak,
			Fingerprint: "pagebreak",
			Path:        copyPath(path),
			SectionPath: copyStrings(sectionPath),
			Source:      n.Range,
			Node:        n,
		})
	case *ast.Comment:
		// Comments do not render and do not participate in diffing.
	}
}

func (f *flattener) emitSection(s *ast.Section, path []int, sectionPath []string) {
	f.emit(Block{
		Kind:        BlockSectionHeader,
		Fingerprint: fingerprintSectionHeader(s),
		Path:        copyPath(path),
		SectionPath: copyStrings(sectionPath),
		Source:      s.HeadingRange(),
		Node:        s,
	})

	childSectionPath := append(copyStrings(sectionPath), sectionDisplayLabel(s))
	for i, item := range s.OrderedItems() {
		childPath := append(copyPath(path), i)
		switch {
		case item.Node != nil:
			f.walk(item.Node, childPath, childSectionPath)
		case item.Line != nil:
			f.emit(Block{
				Kind:        BlockSectionLine,
				Fingerprint: fingerprintSectionLine(item.Line),
				Path:        childPath,
				SectionPath: copyStrings(childSectionPath),
				Source:      item.Line.Range,
				Line:        item.Line,
			})
		}
	}
}

func (f *flattener) emit(b Block) {
	f.blocks = append(f.blocks, b)
}

// sectionDisplayLabel constructs a human-readable label for a section,
// preferring the title and falling back to "<KIND> <NUMBER>" when titles are
// empty (which is common for "## ACT II" headings where the parser stores
// the literal as (Number="II", Title="")).
func sectionDisplayLabel(s *ast.Section) string {
	title := strings.TrimSpace(s.Title)
	number := strings.TrimSpace(s.Number)
	switch s.Kind {
	case ast.SectionAct:
		if number != "" && title != "" {
			return "ACT " + number + ": " + title
		}
		if number != "" {
			return "ACT " + number
		}
	case ast.SectionScene:
		if number != "" && title != "" {
			return "SCENE " + number + ": " + title
		}
		if number != "" {
			return "SCENE " + number
		}
	}
	return title
}

func copyPath(p []int) []int {
	out := make([]int, len(p))
	copy(out, p)
	return out
}

func copyStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
