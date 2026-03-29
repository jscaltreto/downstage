package ast

import "testing"

type recordingVisitor struct {
	visited []string
	stopAt  map[string]bool
}

func (v *recordingVisitor) Visit(node Node) Visitor {
	v.visited = append(v.visited, node.nodeType())
	if v.stopAt[node.nodeType()] {
		return nil
	}
	return v
}

func TestWalk_VisitsNodesDepthFirst(t *testing.T) {
	doc := &Document{
		TitlePage: &TitlePage{},
		Body: []Node{
			&Section{
				Children: []Node{
					&Dialogue{
						Lines: []DialogueLine{
							{
								Content: []Inline{
									&TextNode{Value: "plain"},
									&BoldNode{Content: []Inline{&TextNode{Value: "bold"}}},
								},
							},
						},
					},
				},
				Lines: []SectionLine{
					{Content: []Inline{&InlineDirectionNode{Content: []Inline{&TextNode{Value: "aside"}}}}},
				},
				order: []sectionItemRef{
					{kind: SectionItemLine, index: 0},
					{kind: SectionItemNode, index: 0},
				},
			},
		},
	}

	visitor := &recordingVisitor{}
	Walk(visitor, doc)

	expected := []string{
		"Document",
		"TitlePage",
		"Section",
		"InlineDirectionNode",
		"TextNode",
		"Dialogue",
		"TextNode",
		"BoldNode",
		"TextNode",
	}

	if len(visitor.visited) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, visitor.visited)
	}
	for i := range expected {
		if visitor.visited[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, visitor.visited)
		}
	}
}

func TestWalk_StopVisitorPreventsChildTraversal(t *testing.T) {
	doc := &Document{
		Body: []Node{
			&Section{
				Children: []Node{
					&Dialogue{
						Lines: []DialogueLine{
							{Content: []Inline{&TextNode{Value: "hidden"}}},
						},
					},
				},
			},
		},
	}

	visitor := &recordingVisitor{stopAt: map[string]bool{"Section": true}}
	Walk(visitor, doc)

	expected := []string{"Document", "Section"}
	if len(visitor.visited) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, visitor.visited)
	}
	for i := range expected {
		if visitor.visited[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, visitor.visited)
		}
	}
}

func TestWalk_NilNodeDoesNothing(t *testing.T) {
	visitor := &recordingVisitor{}
	Walk(visitor, nil)

	if len(visitor.visited) != 0 {
		t.Fatalf("expected no visits, got %v", visitor.visited)
	}
}
