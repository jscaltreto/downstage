package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeCodeActions_AddUnknownCharacterToDramatisPersonae(t *testing.T) {
	content := `# Dramatis Personae
HAMLET

# Play

GHOST
Boo.`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}

	action := actions[0]
	if action.Title != "Add GHOST to Dramatis Personae" {
		t.Fatalf("unexpected title: %q", action.Title)
	}
	if action.Kind != protocol.QuickFix {
		t.Fatalf("expected quick fix kind, got %q", action.Kind)
	}
	if action.Edit == nil {
		t.Fatal("expected workspace edit")
	}

	edits := action.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "GHOST\n" {
		t.Fatalf("expected insertion for character entry, got %q", edits[0].NewText)
	}
	if edits[0].Range.Start.Line != 2 {
		t.Fatalf("expected insert on line 2, got %d", edits[0].Range.Start.Line)
	}
}

func TestComputeCodeActions_EmptyDramatisPersonaeAddsSpacedEntry(t *testing.T) {
	content := `# Dramatis Personae

# Play

GHOST
Boo.`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := []protocol.Diagnostic{
		{
			Code: diagnosticCodeUnknownCharacter,
			Data: map[string]interface{}{
				"character": "GHOST",
			},
		},
	}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}

	edits := actions[0].Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if edits[0].NewText != "\nGHOST\n" {
		t.Fatalf("expected spaced insertion for empty dramatis personae, got %q", edits[0].NewText)
	}
}

func TestComputeCodeActions_NoDramatisPersonaeReturnsNothing(t *testing.T) {
	content := `# Play

GHOST
Boo.`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := []protocol.Diagnostic{
		{
			Code: diagnosticCodeUnknownCharacter,
			Data: map[string]interface{}{
				"character": "GHOST",
			},
		},
	}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics)
	if len(actions) != 0 {
		t.Fatalf("expected 0 code actions, got %d", len(actions))
	}
}

func TestComputeCodeActions_NumberUnnumberedActHeading(t *testing.T) {
	content := `# Play

## ACT: Prologue`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}

	action := actions[0]
	if action.Title != "Number heading as ## ACT I: Prologue" {
		t.Fatalf("unexpected title: %q", action.Title)
	}

	edits := action.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "## ACT I: Prologue" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}
}

func TestComputeCodeActions_NumberUnnumberedSceneHeading(t *testing.T) {
	content := `# Play

## ACT I

## The Kitchen`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}

	action := actions[0]
	if action.Title != "Number heading as ## SCENE 1: The Kitchen" {
		t.Fatalf("unexpected title: %q", action.Title)
	}

	edits := action.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "## SCENE 1: The Kitchen" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}
}

func TestComputeCodeActions_NumberSceneHeadingWithSubtitle(t *testing.T) {
	content := `### SCENE: The Kitchen`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}

	action := actions[0]
	if action.Title != "Number heading as ### SCENE 1: The Kitchen" {
		t.Fatalf("unexpected title: %q", action.Title)
	}

	edits := action.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "### SCENE 1: The Kitchen" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}
}
