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
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
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

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
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

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
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
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
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
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
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
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
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

func TestComputeCodeActions_NumberAllActsInDocument(t *testing.T) {
	content := `# Play

## ACT: Prologue

### SCENE 1

## ACT: Finale`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)

	var bulkAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Number all acts in document" {
			bulkAction = &actions[i]
			break
		}
	}
	if bulkAction == nil {
		t.Fatal("expected bulk act numbering action")
	}

	edits := bulkAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 2 {
		t.Fatalf("expected 2 text edits, got %d", len(edits))
	}
	if edits[0].NewText != "## ACT I: Prologue" {
		t.Fatalf("unexpected first act replacement: %q", edits[0].NewText)
	}
	if edits[1].NewText != "## ACT II: Finale" {
		t.Fatalf("unexpected second act replacement: %q", edits[1].NewText)
	}
}

func TestComputeCodeActions_NumberAllScenesInDocument(t *testing.T) {
	content := `# Play

## ACT I

## The Kitchen

### SCENE

## ACT II

## The Garden`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)

	var bulkAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Number all scenes in document" {
			bulkAction = &actions[i]
			break
		}
	}
	if bulkAction == nil {
		t.Fatal("expected bulk scene numbering action")
	}

	edits := bulkAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 3 {
		t.Fatalf("expected 3 text edits, got %d", len(edits))
	}
	if edits[0].NewText != "## SCENE 1: The Kitchen" {
		t.Fatalf("unexpected first scene replacement: %q", edits[0].NewText)
	}
	if edits[1].NewText != "### SCENE 2" {
		t.Fatalf("unexpected second scene replacement: %q", edits[1].NewText)
	}
	if edits[2].NewText != "## SCENE 1: The Garden" {
		t.Fatalf("unexpected third scene replacement: %q", edits[2].NewText)
	}
}

func TestComputeCodeActions_ContextDiagnosticStillShowsBulkDocumentActions(t *testing.T) {
	content := `# Play

## ACT I

## The Kitchen

### SCENE

## ACT II

## The Garden`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	allDiagnostics := buildDiagnostics(doc, errs)
	contextDiagnostics := []protocol.Diagnostic{allDiagnostics[0]}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), contextDiagnostics, allDiagnostics)

	var hasSingle bool
	var hasBulk bool
	for _, action := range actions {
		if action.Title == "Number heading as ## SCENE 1: The Kitchen" {
			hasSingle = true
		}
		if action.Title == "Number all scenes in document" {
			hasBulk = true
		}
	}

	if !hasSingle {
		t.Fatal("expected single-heading quick fix")
	}
	if !hasBulk {
		t.Fatal("expected document-level scene quick fix")
	}
}
