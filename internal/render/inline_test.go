package render

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
)

func TestPlainText_StripsFormattingAndPreservesInlineDirections(t *testing.T) {
	inlines := []ast.Inline{
		&ast.TextNode{Value: "Hello "},
		&ast.BoldNode{Content: []ast.Inline{&ast.TextNode{Value: "bold"}}},
		&ast.TextNode{Value: " "},
		&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "italic"}}},
		&ast.TextNode{Value: " "},
		&ast.BoldItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "both"}}},
		&ast.TextNode{Value: " "},
		&ast.UnderlineNode{Content: []ast.Inline{&ast.TextNode{Value: "under"}}},
		&ast.TextNode{Value: " "},
		&ast.StrikethroughNode{Content: []ast.Inline{&ast.TextNode{Value: "strike"}}},
		&ast.TextNode{Value: " "},
		&ast.InlineDirectionNode{Content: []ast.Inline{&ast.TextNode{Value: "aside"}}},
	}

	got := PlainText(inlines)

	if got != "Hello bold italic both under strike (aside)" {
		t.Fatalf("unexpected plain text: %q", got)
	}
}

func TestPlainText_EmptyInput(t *testing.T) {
	if got := PlainText(nil); got != "" {
		t.Fatalf("expected empty plain text, got %q", got)
	}
}
