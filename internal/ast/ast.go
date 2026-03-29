package ast

import (
	"unicode/utf16"

	"github.com/jscaltreto/downstage/internal/token"
)

// Node is the interface all AST nodes implement.
type Node interface {
	NodeRange() token.Range
	nodeType() string
}

// Inline represents inline content nodes.
type Inline interface {
	Node
	inlineNode()
}

// --- Document (root) ---

var _ Node = (*Document)(nil)

// Document is the root AST node.
type Document struct {
	TitlePage *TitlePage
	Body      []Node
	Range     token.Range
}

func (d *Document) NodeRange() token.Range { return d.Range }
func (d *Document) nodeType() string       { return "Document" }

// --- TitlePage ---

var _ Node = (*TitlePage)(nil)

// TitlePage holds metadata key-value pairs.
type TitlePage struct {
	Entries []KeyValue
	Range   token.Range
}

func (t *TitlePage) NodeRange() token.Range { return t.Range }
func (t *TitlePage) nodeType() string       { return "TitlePage" }

// KeyValue is a single title page entry.
type KeyValue struct {
	Key   string
	Value string
	Range token.Range
}

// --- Section ---

// SectionKind classifies the semantic role of a section.
type SectionKind int

const (
	SectionGeneric          SectionKind = iota // arbitrary prose (notes, credits, etc.)
	SectionAct                                 // ## ACT ...
	SectionScene                               // ### SCENE ... or ## non-ACT heading inside an act
	SectionDramatisPersonae                    // # Dramatis Personae
)

var _ Node = (*Section)(nil)

// Section represents a headed content block. All # / ## / ### headings
// produce Section nodes; the Kind field carries semantic meaning.
type Section struct {
	Kind       SectionKind
	Level      int              // 1 (#), 2 (##), 3 (###)
	Title      string           // heading text (e.g. "ACT I", "Playwright's Notes")
	Number     string           // act/scene number (e.g. "I", "1") — empty for generic
	Children   []Node           // nested sections + content (dialogue, directions, songs, etc.)
	Characters []Character      // populated only for SectionDramatisPersonae
	Groups     []CharacterGroup // populated only for SectionDramatisPersonae
	Lines      []SectionLine    // populated only for SectionGeneric (prose content)
	Range      token.Range
	order      []sectionItemRef
}

func (s *Section) NodeRange() token.Range { return s.Range }
func (s *Section) nodeType() string       { return "Section" }

func (s *Section) AppendChild(child Node) {
	s.Children = append(s.Children, child)
	s.order = append(s.order, sectionItemRef{kind: SectionItemNode, index: len(s.Children) - 1})
}

// WrapLastTwoChildren replaces the last two adjacent child items in the section
// ordering with a single node. It returns false when the last two children are
// not consecutive rendered items, such as when prose lines appear between them.
func (s *Section) WrapLastTwoChildren(replacement Node) bool {
	n := len(s.Children)
	if n < 2 {
		return false
	}
	if len(s.order) < 2 {
		return false
	}

	prev := s.order[len(s.order)-2]
	last := s.order[len(s.order)-1]
	if prev.kind != SectionItemNode || last.kind != SectionItemNode {
		return false
	}
	if prev.index != n-2 || last.index != n-1 {
		return false
	}
	s.Children[n-2] = replacement
	s.Children = s.Children[:n-1]
	s.order = s.order[:len(s.order)-1]
	return true
}

func (s *Section) AppendLine(line SectionLine) {
	s.Lines = append(s.Lines, line)
	s.order = append(s.order, sectionItemRef{kind: SectionItemLine, index: len(s.Lines) - 1})
}

func (s *Section) OrderedItems() []SectionItem {
	if len(s.order) == 0 {
		items := make([]SectionItem, 0, len(s.Children)+len(s.Lines))
		for i := range s.Children {
			items = append(items, SectionItem{Kind: SectionItemNode, Node: s.Children[i]})
		}
		for i := range s.Lines {
			items = append(items, SectionItem{Kind: SectionItemLine, Line: &s.Lines[i]})
		}
		return items
	}

	items := make([]SectionItem, 0, len(s.order))
	for _, ref := range s.order {
		switch ref.kind {
		case SectionItemNode:
			items = append(items, SectionItem{Kind: ref.kind, Node: s.Children[ref.index]})
		case SectionItemLine:
			items = append(items, SectionItem{Kind: ref.kind, Line: &s.Lines[ref.index]})
		}
	}
	return items
}

func (s *Section) TrimTrailingBlankLines() {
	for len(s.Lines) > 0 && len(s.Lines[len(s.Lines)-1].Content) == 0 {
		if len(s.order) > 0 {
			last := s.order[len(s.order)-1]
			if last.kind == SectionItemLine && last.index == len(s.Lines)-1 {
				s.order = s.order[:len(s.order)-1]
			}
		}
		s.Lines = s.Lines[:len(s.Lines)-1]
	}
}

// SectionLine is a single line of prose content in a generic section.
type SectionLine struct {
	Content []Inline
	Range   token.Range
}

type SectionItemKind int

const (
	SectionItemNode SectionItemKind = iota
	SectionItemLine
)

type SectionItem struct {
	Kind SectionItemKind
	Node Node
	Line *SectionLine
}

type sectionItemRef struct {
	kind  SectionItemKind
	index int
}

// Character describes a character entry in the dramatis personae.
type Character struct {
	Name        string
	Aliases     []string
	Description string
	Range       token.Range
}

// CharacterGroup is a named group of characters.
type CharacterGroup struct {
	Name       string
	Characters []Character
	Range      token.Range
}

// FindDramatisPersonae searches the body for a SectionDramatisPersonae node.
func FindDramatisPersonae(body []Node) *Section {
	for _, node := range body {
		if s, ok := node.(*Section); ok && s.Kind == SectionDramatisPersonae {
			return s
		}
	}
	return nil
}

// AllCharacters returns all characters from a DramatisPersonae section,
// including those in groups.
func (s *Section) AllCharacters() []Character {
	if s.Kind != SectionDramatisPersonae {
		return nil
	}
	var chars []Character
	chars = append(chars, s.Characters...)
	for _, g := range s.Groups {
		chars = append(chars, g.Characters...)
	}
	return chars
}

// --- Dialogue ---

var _ Node = (*Dialogue)(nil)

// Dialogue represents character dialogue.
type Dialogue struct {
	Character          string
	Parenthetical      string
	Lines              []DialogueLine
	Range              token.Range
	nameRange          token.Range
	parentheticalRange token.Range
}

func (d *Dialogue) NodeRange() token.Range { return d.Range }
func (d *Dialogue) nodeType() string       { return "Dialogue" }
func (d *Dialogue) NameRange() token.Range {
	if d.nameRange.Start == (token.Position{}) && d.nameRange.End == (token.Position{}) {
		r := d.Range
		r.End = r.Start
		r.End.Column += len(utf16.Encode([]rune(d.Character)))
		r.End.Offset += len(d.Character)
		return r
	}
	return d.nameRange
}
func (d *Dialogue) SetNameRange(r token.Range) {
	d.nameRange = r
}
func (d *Dialogue) ParentheticalRange() token.Range {
	return d.parentheticalRange
}
func (d *Dialogue) SetParentheticalRange(r token.Range) {
	d.parentheticalRange = r
}

// --- Dual Dialogue ---

var _ Node = (*DualDialogue)(nil)

// DualDialogue represents two dialogue blocks spoken simultaneously, rendered side-by-side.
type DualDialogue struct {
	Left  *Dialogue
	Right *Dialogue
	Range token.Range
}

func (d *DualDialogue) NodeRange() token.Range { return d.Range }
func (d *DualDialogue) nodeType() string       { return "DualDialogue" }

// DialogueLine is a single line of dialogue.
type DialogueLine struct {
	Content []Inline
	IsVerse bool
	Range   token.Range
}

// --- StageDirection ---

var _ Node = (*StageDirection)(nil)

// StageDirection is a standalone stage direction.
type StageDirection struct {
	Content []Inline
	Range   token.Range
}

func (sd *StageDirection) NodeRange() token.Range { return sd.Range }
func (sd *StageDirection) nodeType() string       { return "StageDirection" }

// --- Song ---

var _ Node = (*Song)(nil)

// Song represents a song section.
type Song struct {
	Number  string
	Title   string
	Content []Node
	Range   token.Range
}

func (s *Song) NodeRange() token.Range { return s.Range }
func (s *Song) nodeType() string       { return "Song" }

// --- VerseBlock ---

var _ Node = (*VerseBlock)(nil)

// VerseBlock is a block of verse lines.
type VerseBlock struct {
	Lines []VerseLine
	Range token.Range
}

func (vb *VerseBlock) NodeRange() token.Range { return vb.Range }
func (vb *VerseBlock) nodeType() string       { return "VerseBlock" }

// VerseLine is a single verse line.
type VerseLine struct {
	Content []Inline
	Range   token.Range
}

// --- Comment ---

var _ Node = (*Comment)(nil)

// Comment represents a line or block comment.
type Comment struct {
	Text  string
	Block bool
	Range token.Range
}

func (c *Comment) NodeRange() token.Range { return c.Range }
func (c *Comment) nodeType() string       { return "Comment" }

// --- PageBreak ---

var _ Node = (*PageBreak)(nil)

// PageBreak represents a page break marker.
type PageBreak struct {
	Range token.Range
}

func (pb *PageBreak) NodeRange() token.Range { return pb.Range }
func (pb *PageBreak) nodeType() string       { return "PageBreak" }

// --- Inline nodes ---

var _ Inline = (*TextNode)(nil)

// TextNode is plain text content.
type TextNode struct {
	Value string
	Range token.Range
}

func (t *TextNode) NodeRange() token.Range { return t.Range }
func (t *TextNode) nodeType() string       { return "TextNode" }
func (t *TextNode) inlineNode()            {}

var _ Inline = (*BoldNode)(nil)

// BoldNode is bold formatted text.
type BoldNode struct {
	Content []Inline
	Range   token.Range
}

func (b *BoldNode) NodeRange() token.Range { return b.Range }
func (b *BoldNode) nodeType() string       { return "BoldNode" }
func (b *BoldNode) inlineNode()            {}

var _ Inline = (*ItalicNode)(nil)

// ItalicNode is italic formatted text.
type ItalicNode struct {
	Content []Inline
	Range   token.Range
}

func (i *ItalicNode) NodeRange() token.Range { return i.Range }
func (i *ItalicNode) nodeType() string       { return "ItalicNode" }
func (i *ItalicNode) inlineNode()            {}

var _ Inline = (*BoldItalicNode)(nil)

// BoldItalicNode is bold+italic formatted text.
type BoldItalicNode struct {
	Content []Inline
	Range   token.Range
}

func (bi *BoldItalicNode) NodeRange() token.Range { return bi.Range }
func (bi *BoldItalicNode) nodeType() string       { return "BoldItalicNode" }
func (bi *BoldItalicNode) inlineNode()            {}

var _ Inline = (*UnderlineNode)(nil)

// UnderlineNode is underlined text.
type UnderlineNode struct {
	Content []Inline
	Range   token.Range
}

func (u *UnderlineNode) NodeRange() token.Range { return u.Range }
func (u *UnderlineNode) nodeType() string       { return "UnderlineNode" }
func (u *UnderlineNode) inlineNode()            {}

var _ Inline = (*StrikethroughNode)(nil)

// StrikethroughNode is strikethrough text.
type StrikethroughNode struct {
	Content []Inline
	Range   token.Range
}

func (s *StrikethroughNode) NodeRange() token.Range { return s.Range }
func (s *StrikethroughNode) nodeType() string       { return "StrikethroughNode" }
func (s *StrikethroughNode) inlineNode()            {}

var _ Inline = (*InlineDirectionNode)(nil)

// InlineDirectionNode is an inline stage direction within dialogue.
type InlineDirectionNode struct {
	Content []Inline
	Range   token.Range
}

func (id *InlineDirectionNode) NodeRange() token.Range { return id.Range }
func (id *InlineDirectionNode) nodeType() string       { return "InlineDirectionNode" }
func (id *InlineDirectionNode) inlineNode()            {}
