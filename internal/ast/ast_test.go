package ast

import "testing"

func TestSectionTrimTrailingBlankLinesRemovesStaleOrderRefs(t *testing.T) {
	section := &Section{}
	section.AppendLine(SectionLine{Content: []Inline{&TextNode{Value: "before"}}})
	section.AppendLine(SectionLine{})
	section.AppendChild(&Dialogue{Character: "ALICE"})

	section.TrimTrailingBlankLines()

	items := section.OrderedItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items after trimming, got %d", len(items))
	}
	if items[0].Line == nil || len(items[0].Line.Content) != 1 {
		t.Fatalf("expected first item to be the prose line, got %#v", items[0])
	}
	if items[1].Node == nil {
		t.Fatalf("expected second item to remain the child node, got %#v", items[1])
	}
}
