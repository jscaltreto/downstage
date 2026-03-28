package render

import (
	"io"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
)

type recordingRenderer struct {
	events []string
}

func (r *recordingRenderer) BeginDocument(_ *ast.Document, _ io.Writer) error { return nil }
func (r *recordingRenderer) EndDocument(_ *ast.Document) error                { return nil }
func (r *recordingRenderer) RenderTitlePage(_ *ast.TitlePage) error           { return nil }
func (r *recordingRenderer) BeginSong(_ *ast.Song) error                      { return nil }
func (r *recordingRenderer) EndSong(_ *ast.Song) error                        { return nil }
func (r *recordingRenderer) BeginDialogue(d *ast.Dialogue) error {
	r.events = append(r.events, "dialogue:"+d.Character)
	return nil
}
func (r *recordingRenderer) EndDialogue(_ *ast.Dialogue) error               { return nil }
func (r *recordingRenderer) BeginDialogueLine(_ *ast.DialogueLine) error     { return nil }
func (r *recordingRenderer) EndDialogueLine(_ *ast.DialogueLine) error       { return nil }
func (r *recordingRenderer) BeginStageDirection(_ *ast.StageDirection) error { return nil }
func (r *recordingRenderer) EndStageDirection(_ *ast.StageDirection) error   { return nil }
func (r *recordingRenderer) BeginVerseBlock(_ *ast.VerseBlock) error         { return nil }
func (r *recordingRenderer) EndVerseBlock(_ *ast.VerseBlock) error           { return nil }
func (r *recordingRenderer) BeginVerseLine(_ *ast.VerseLine) error           { return nil }
func (r *recordingRenderer) EndVerseLine(_ *ast.VerseLine) error             { return nil }
func (r *recordingRenderer) BeginSection(_ *ast.Section) error               { return nil }
func (r *recordingRenderer) EndSection(_ *ast.Section) error                 { return nil }
func (r *recordingRenderer) BeginSectionLine(sl *ast.SectionLine) error {
	r.events = append(r.events, "line:"+PlainText(sl.Content))
	return nil
}
func (r *recordingRenderer) EndSectionLine(_ *ast.SectionLine) error     { return nil }
func (r *recordingRenderer) RenderPageBreak(_ *ast.PageBreak) error      { return nil }
func (r *recordingRenderer) RenderComment(_ *ast.Comment) error          { return nil }
func (r *recordingRenderer) RenderText(_ *ast.TextNode) error            { return nil }
func (r *recordingRenderer) BeginBold(_ *ast.BoldNode) error             { return nil }
func (r *recordingRenderer) EndBold(_ *ast.BoldNode) error               { return nil }
func (r *recordingRenderer) BeginItalic(_ *ast.ItalicNode) error         { return nil }
func (r *recordingRenderer) EndItalic(_ *ast.ItalicNode) error           { return nil }
func (r *recordingRenderer) BeginBoldItalic(_ *ast.BoldItalicNode) error { return nil }
func (r *recordingRenderer) EndBoldItalic(_ *ast.BoldItalicNode) error   { return nil }
func (r *recordingRenderer) BeginUnderline(_ *ast.UnderlineNode) error   { return nil }
func (r *recordingRenderer) EndUnderline(_ *ast.UnderlineNode) error     { return nil }
func (r *recordingRenderer) BeginStrikethrough(_ *ast.StrikethroughNode) error {
	return nil
}
func (r *recordingRenderer) EndStrikethrough(_ *ast.StrikethroughNode) error { return nil }
func (r *recordingRenderer) BeginInlineDirection(_ *ast.InlineDirectionNode) error {
	return nil
}
func (r *recordingRenderer) EndInlineDirection(_ *ast.InlineDirectionNode) error { return nil }

func TestWalkPreservesGenericSectionOrder(t *testing.T) {
	section := &ast.Section{}
	section.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "before"}}})
	section.AppendChild(&ast.Dialogue{Character: "ALICE"})
	section.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "after"}}})

	doc := &ast.Document{Body: []ast.Node{section}}
	renderer := &recordingRenderer{}
	if err := Walk(renderer, doc, io.Discard); err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	expected := []string{"line:before", "dialogue:ALICE", "line:after"}
	if len(renderer.events) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, renderer.events)
	}
	for i := range expected {
		if renderer.events[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, renderer.events)
		}
	}
}
