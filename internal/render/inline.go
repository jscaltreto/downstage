package render

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

// PlainText extracts plain text content from inline nodes, stripping formatting.
func PlainText(inlines []ast.Inline) string {
	var b strings.Builder
	plainTextInto(&b, inlines)
	return b.String()
}

func plainTextInto(b *strings.Builder, inlines []ast.Inline) {
	for _, inline := range inlines {
		switch n := inline.(type) {
		case *ast.TextNode:
			b.WriteString(n.Value)
		case *ast.BoldNode:
			plainTextInto(b, n.Content)
		case *ast.ItalicNode:
			plainTextInto(b, n.Content)
		case *ast.BoldItalicNode:
			plainTextInto(b, n.Content)
		case *ast.UnderlineNode:
			plainTextInto(b, n.Content)
		case *ast.StrikethroughNode:
			plainTextInto(b, n.Content)
		case *ast.InlineDirectionNode:
			b.WriteString("(")
			plainTextInto(b, n.Content)
			b.WriteString(")")
		}
	}
}
