package render

import (
	"io"

	"github.com/jscaltreto/downstage/internal/ast"
)

// NodeRenderer is the rendering interface. Format implementations (PDF, HTML,
// DOCX) implement this interface. The Walk function calls these methods during
// AST traversal — implementations handle output only, never tree walking.
//
// Begin methods are called before children are traversed.
// End methods are called after children are traversed.
// Render methods handle leaf or self-contained nodes.
type NodeRenderer interface {
	// Lifecycle
	BeginDocument(doc *ast.Document, w io.Writer) error
	EndDocument(doc *ast.Document) error

	// Front matter (self-contained — children are data types, not Nodes)
	RenderTitlePage(tp *ast.TitlePage) error

	// Structural
	BeginSong(song *ast.Song) error
	EndSong(song *ast.Song) error

	// Content blocks
	BeginDialogue(d *ast.Dialogue) error
	EndDialogue(d *ast.Dialogue) error
	BeginDialogueLine(line *ast.DialogueLine) error
	EndDialogueLine(line *ast.DialogueLine) error
	BeginStageDirection(sd *ast.StageDirection) error
	EndStageDirection(sd *ast.StageDirection) error
	BeginVerseBlock(vb *ast.VerseBlock) error
	EndVerseBlock(vb *ast.VerseBlock) error
	BeginVerseLine(vl *ast.VerseLine) error
	EndVerseLine(vl *ast.VerseLine) error

	// Sections (arbitrary content blocks)
	BeginSection(s *ast.Section) error
	EndSection(s *ast.Section) error
	BeginSectionLine(sl *ast.SectionLine) error
	EndSectionLine(sl *ast.SectionLine) error

	// Leaves
	RenderPageBreak(pb *ast.PageBreak) error
	RenderComment(c *ast.Comment) error

	// Inline
	RenderText(t *ast.TextNode) error
	BeginBold(b *ast.BoldNode) error
	EndBold(b *ast.BoldNode) error
	BeginItalic(i *ast.ItalicNode) error
	EndItalic(i *ast.ItalicNode) error
	BeginBoldItalic(bi *ast.BoldItalicNode) error
	EndBoldItalic(bi *ast.BoldItalicNode) error
	BeginUnderline(u *ast.UnderlineNode) error
	EndUnderline(u *ast.UnderlineNode) error
	BeginStrikethrough(s *ast.StrikethroughNode) error
	EndStrikethrough(s *ast.StrikethroughNode) error
	BeginInlineDirection(id *ast.InlineDirectionNode) error
	EndInlineDirection(id *ast.InlineDirectionNode) error
}

// Walk traverses the AST and calls the appropriate NodeRenderer methods.
func Walk(nr NodeRenderer, doc *ast.Document, w io.Writer) error {
	if err := nr.BeginDocument(doc, w); err != nil {
		return err
	}

	if doc.TitlePage != nil {
		if err := nr.RenderTitlePage(doc.TitlePage); err != nil {
			return err
		}
	}
	for _, node := range doc.Body {
		if err := walkNode(nr, node); err != nil {
			return err
		}
	}

	return nr.EndDocument(doc)
}

func walkNode(nr NodeRenderer, node ast.Node) error {
	switch n := node.(type) {
	case *ast.Dialogue:
		if err := nr.BeginDialogue(n); err != nil {
			return err
		}
		for i := range n.Lines {
			if err := nr.BeginDialogueLine(&n.Lines[i]); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Lines[i].Content); err != nil {
				return err
			}
			if err := nr.EndDialogueLine(&n.Lines[i]); err != nil {
				return err
			}
		}
		return nr.EndDialogue(n)

	case *ast.StageDirection:
		if err := nr.BeginStageDirection(n); err != nil {
			return err
		}
		if err := walkInlines(nr, n.Content); err != nil {
			return err
		}
		return nr.EndStageDirection(n)

	case *ast.Song:
		if err := nr.BeginSong(n); err != nil {
			return err
		}
		for _, child := range n.Content {
			if err := walkNode(nr, child); err != nil {
				return err
			}
		}
		return nr.EndSong(n)

	case *ast.VerseBlock:
		if err := nr.BeginVerseBlock(n); err != nil {
			return err
		}
		for i := range n.Lines {
			if err := nr.BeginVerseLine(&n.Lines[i]); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Lines[i].Content); err != nil {
				return err
			}
			if err := nr.EndVerseLine(&n.Lines[i]); err != nil {
				return err
			}
		}
		return nr.EndVerseBlock(n)

	case *ast.Section:
		if err := nr.BeginSection(n); err != nil {
			return err
		}
		for _, item := range n.OrderedItems() {
			if item.Node != nil {
				if err := walkNode(nr, item.Node); err != nil {
					return err
				}
				continue
			}
			if item.Line == nil {
				continue
			}
			if len(item.Line.Content) == 0 {
				// Blank line — still call Begin/End so renderer can add spacing
				if err := nr.BeginSectionLine(item.Line); err != nil {
					return err
				}
				if err := nr.EndSectionLine(item.Line); err != nil {
					return err
				}
				continue
			}
			if err := nr.BeginSectionLine(item.Line); err != nil {
				return err
			}
			if err := walkInlines(nr, item.Line.Content); err != nil {
				return err
			}
			if err := nr.EndSectionLine(item.Line); err != nil {
				return err
			}
		}
		return nr.EndSection(n)

	case *ast.PageBreak:
		return nr.RenderPageBreak(n)

	case *ast.Comment:
		return nr.RenderComment(n)
	}

	return nil
}

func walkInlines(nr NodeRenderer, inlines []ast.Inline) error {
	for _, inline := range inlines {
		switch n := inline.(type) {
		case *ast.TextNode:
			if err := nr.RenderText(n); err != nil {
				return err
			}
		case *ast.BoldNode:
			if err := nr.BeginBold(n); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Content); err != nil {
				return err
			}
			if err := nr.EndBold(n); err != nil {
				return err
			}
		case *ast.ItalicNode:
			if err := nr.BeginItalic(n); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Content); err != nil {
				return err
			}
			if err := nr.EndItalic(n); err != nil {
				return err
			}
		case *ast.BoldItalicNode:
			if err := nr.BeginBoldItalic(n); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Content); err != nil {
				return err
			}
			if err := nr.EndBoldItalic(n); err != nil {
				return err
			}
		case *ast.UnderlineNode:
			if err := nr.BeginUnderline(n); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Content); err != nil {
				return err
			}
			if err := nr.EndUnderline(n); err != nil {
				return err
			}
		case *ast.StrikethroughNode:
			if err := nr.BeginStrikethrough(n); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Content); err != nil {
				return err
			}
			if err := nr.EndStrikethrough(n); err != nil {
				return err
			}
		case *ast.InlineDirectionNode:
			if err := nr.BeginInlineDirection(n); err != nil {
				return err
			}
			if err := walkInlines(nr, n.Content); err != nil {
				return err
			}
			if err := nr.EndInlineDirection(n); err != nil {
				return err
			}
		}
	}
	return nil
}
