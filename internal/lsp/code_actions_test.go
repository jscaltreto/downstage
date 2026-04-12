package lsp

import (
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeCodeActions_AddUnknownCharacterToDramatisPersonae(t *testing.T) {
	content := `# Play

## Dramatis Personae
HAMLET

## ACT I

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
	if edits[0].Range.Start.Line != 4 {
		t.Fatalf("expected insert on line 2, got %d", edits[0].Range.Start.Line)
	}
}

func TestComputeCodeActions_UpdateScriptToV2(t *testing.T) {
	content := `Title: Hamlet
Author: William Shakespeare

# Dramatis Personae

HAMLET — Prince of Denmark

# Hamlet

HAMLET
To be.`

	doc, errs := parser.Parse([]byte(content))
	diagnostics := buildDiagnostics(doc, errs)

	var v1Diagnostics []protocol.Diagnostic
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == diagnosticCodeV1Document {
			v1Diagnostics = append(v1Diagnostics, diagnostic)
		}
	}
	if len(v1Diagnostics) != 1 {
		t.Fatalf("expected 1 v1 diagnostic, got %d", len(v1Diagnostics))
	}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), v1Diagnostics, diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}
	if actions[0].Title != "Update script to V2" {
		t.Fatalf("unexpected action title: %q", actions[0].Title)
	}

	edits := actions[0].Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if !strings.Contains(edits[0].NewText, "## Dramatis Personae") {
		t.Fatalf("expected V2 rewrite, got %q", edits[0].NewText)
	}
}

func TestComputeCodeActions_EmptyDramatisPersonaeAddsSpacedEntry(t *testing.T) {
	content := `# Play

## Dramatis Personae

## ACT I

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
	content := `# Play

## ACT I

### SCENE: The Kitchen`

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

func TestComputeCodeActions_BareActHeadingNoTitle(t *testing.T) {
	content := `# Play

## ACT`

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
	if action.Title != "Number heading as ## ACT I" {
		t.Fatalf("unexpected title: %q", action.Title)
	}

	edits := action.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "## ACT I" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}
}

func TestComputeCodeActions_RenumberMisnumberedActHeading(t *testing.T) {
	content := `# Play

## ACT I

## ACT I: Duplicate

## ACT V: Finale`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)

	var singleAction, bulkAction *protocol.CodeAction
	for i := range actions {
		switch actions[i].Title {
		case "Renumber heading as ## ACT II: Duplicate":
			singleAction = &actions[i]
		case "Normalize all acts in document":
			bulkAction = &actions[i]
		}
	}
	if singleAction == nil {
		t.Fatal("expected single act renumber action")
	}
	if bulkAction == nil {
		t.Fatal("expected bulk act normalization action")
	}

	edits := singleAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if edits[0].NewText != "## ACT II: Duplicate" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}

	bulkEdits := bulkAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(bulkEdits) != 2 {
		t.Fatalf("expected 2 bulk act edits, got %d", len(bulkEdits))
	}
}

func TestComputeCodeActions_RenumberMisnumberedSceneHeading(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

### SCENE 4: Garden

### SCENE 7: Finale`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)

	var singleAction, bulkAction *protocol.CodeAction
	for i := range actions {
		switch actions[i].Title {
		case "Renumber heading as ### SCENE 2: Garden":
			singleAction = &actions[i]
		case "Normalize all scenes in document":
			bulkAction = &actions[i]
		}
	}
	if singleAction == nil {
		t.Fatal("expected single scene renumber action")
	}
	if bulkAction == nil {
		t.Fatal("expected bulk scene normalization action")
	}

	edits := singleAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if edits[0].NewText != "### SCENE 2: Garden" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}

	bulkEdits := bulkAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(bulkEdits) != 2 {
		t.Fatalf("expected 2 bulk scene edits, got %d", len(bulkEdits))
	}
}

func TestComputeCodeActions_RenumberMisnumberedSceneOutsideActs(t *testing.T) {
	content := `# Play

## SCENE 1

## SCENE 3: Out of Order`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diagnostics := buildDiagnostics(doc, errs)
	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)

	var singleAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Renumber heading as ## SCENE 2: Out of Order" {
			singleAction = &actions[i]
			break
		}
	}
	if singleAction == nil {
		t.Fatal("expected scene renumber action outside acts")
	}

	edits := singleAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "## SCENE 2: Out of Order" {
		t.Fatalf("unexpected replacement text: %q", edits[0].NewText)
	}
}
