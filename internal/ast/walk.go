package ast

// Visitor is implemented by types that want to traverse the AST.
type Visitor interface {
	Visit(node Node) Visitor
}

// Walk traverses the AST depth-first, calling v.Visit for each node.
// If v.Visit returns a non-nil Visitor, Walk recurses into child nodes
// using the returned Visitor. If v.Visit returns nil, Walk does not
// descend into children.
func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}

	v = v.Visit(node)
	if v == nil {
		return
	}

	switch n := node.(type) {
	case *Document:
		if n.TitlePage != nil {
			Walk(v, n.TitlePage)
		}
		for _, child := range n.Body {
			Walk(v, child)
		}

	case *TitlePage:
		// KeyValue is not a Node, nothing to walk

	case *Section:
		for _, item := range n.OrderedItems() {
			if item.Node != nil {
				Walk(v, item.Node)
				continue
			}
			if item.Line != nil {
				walkInlines(v, item.Line.Content)
			}
		}

	case *DualDialogue:
		Walk(v, n.Left)
		Walk(v, n.Right)

	case *Dialogue:
		for _, line := range n.Lines {
			walkInlines(v, line.Content)
		}

	case *StageDirection:
		walkInlines(v, n.Content)

	case *Callout:
		walkInlines(v, n.Content)

	case *Song:
		for _, child := range n.Content {
			Walk(v, child)
		}

	case *VerseBlock:
		for _, line := range n.Lines {
			walkInlines(v, line.Content)
		}

	case *Comment:
		// leaf

	case *PageBreak:
		// leaf

	case *TextNode:
		// leaf

	case *BoldNode:
		walkInlines(v, n.Content)

	case *ItalicNode:
		walkInlines(v, n.Content)

	case *BoldItalicNode:
		walkInlines(v, n.Content)

	case *UnderlineNode:
		walkInlines(v, n.Content)

	case *StrikethroughNode:
		walkInlines(v, n.Content)

	case *InlineDirectionNode:
		walkInlines(v, n.Content)
	}
}

func walkInlines(v Visitor, inlines []Inline) {
	for _, inline := range inlines {
		Walk(v, inline)
	}
}
