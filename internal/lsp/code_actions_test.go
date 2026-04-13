package lsp

import (
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeCodeActions_UnknownCharacterOffersForceAndAddToDP(t *testing.T) {
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
	if len(actions) != 2 {
		t.Fatalf("expected 2 code actions (force cue + add to DP), got %d", len(actions))
	}

	var forceAction, addAction *protocol.CodeAction
	for i := range actions {
		switch {
		case actions[i].Title == "Exclude cue from check":
			forceAction = &actions[i]
		case actions[i].Title == "Add GHOST to Dramatis Personae":
			addAction = &actions[i]
		}
	}
	if forceAction == nil {
		t.Fatal("expected a Force-cue code action")
	}
	if addAction == nil {
		t.Fatal("expected an Add-to-DP code action")
	}

	forceEdits := forceAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(forceEdits) != 1 {
		t.Fatalf("expected 1 force text edit, got %d", len(forceEdits))
	}
	if forceEdits[0].NewText != "@" {
		t.Fatalf("expected Force edit to insert `@`, got %q", forceEdits[0].NewText)
	}
	if forceEdits[0].Range.Start.Line != 7 || forceEdits[0].Range.End.Line != 7 {
		t.Fatalf("expected force edit on cue line 7, got %+v", forceEdits[0].Range)
	}
	if forceEdits[0].Range.Start.Character != forceEdits[0].Range.End.Character {
		t.Fatalf("expected zero-length insertion, got %+v", forceEdits[0].Range)
	}

	addEdits := addAction.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	if len(addEdits) != 1 {
		t.Fatalf("expected 1 DP text edit, got %d", len(addEdits))
	}
	if addEdits[0].NewText != "GHOST\n" {
		t.Fatalf("expected insertion for character entry, got %q", addEdits[0].NewText)
	}
	if addEdits[0].Range.Start.Line != 4 {
		t.Fatalf("expected insert on line 4, got %d", addEdits[0].Range.Start.Line)
	}
}

func TestComputeCodeActions_ReplaceUnicodeDashInDP(t *testing.T) {
	content := "# Play\n\n## Dramatis Personae\n\nHAMLET — Prince of Denmark\n"

	doc, errs := parser.Parse([]byte(content))
	diagnostics := buildDiagnostics(doc, errs)

	var ctx []protocol.Diagnostic
	for _, d := range diagnostics {
		if code, _ := d.Code.(string); code == parser.ErrCodeDPUnicodeDash {
			ctx = append(ctx, d)
		}
	}
	if len(ctx) != 1 {
		t.Fatalf("expected 1 unicode-dash diagnostic, got %d", len(ctx))
	}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///t.ds"), ctx, diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}
	if !strings.Contains(actions[0].Title, "Replace Unicode dash") {
		t.Fatalf("unexpected title: %q", actions[0].Title)
	}

	edit := actions[0].Edit.Changes[protocol.DocumentURI("file:///t.ds")][0]
	if edit.NewText != "HAMLET - Prince of Denmark" {
		t.Fatalf("expected ASCII rewrite, got %q", edit.NewText)
	}
}

func TestComputeCodeActions_InlineStandaloneAlias(t *testing.T) {
	content := "# Play\n\n## Dramatis Personae\n\nHAMLET - Prince\n[HAMLET/HAM]\n"

	doc, errs := parser.Parse([]byte(content))
	diagnostics := buildDiagnostics(doc, errs)

	var ctx []protocol.Diagnostic
	for _, d := range diagnostics {
		if code, _ := d.Code.(string); code == parser.ErrCodeDPStandaloneAlias {
			ctx = append(ctx, d)
		}
	}
	if len(ctx) != 1 {
		t.Fatalf("expected 1 standalone-alias diagnostic, got %d", len(ctx))
	}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///t.ds"), ctx, diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}
	if !strings.Contains(actions[0].Title, "Rewrite alias") {
		t.Fatalf("unexpected title: %q", actions[0].Title)
	}

	edit := actions[0].Edit.Changes[protocol.DocumentURI("file:///t.ds")][0]
	if edit.NewText != "HAMLET/HAM" {
		t.Fatalf("expected bracketless rewrite, got %q", edit.NewText)
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
	if len(actions) != 2 {
		t.Fatalf("expected 2 code actions (force + add to DP), got %d", len(actions))
	}

	var addEdit *protocol.TextEdit
	for i := range actions {
		if actions[i].Title == "Add GHOST to Dramatis Personae" {
			edits := actions[i].Edit.Changes[protocol.DocumentURI("file:///test.ds")]
			if len(edits) == 1 {
				addEdit = &edits[0]
			}
		}
	}
	if addEdit == nil {
		t.Fatal("expected an Add-to-DP action with a single edit")
	}
	if addEdit.NewText != "\nGHOST\n" {
		t.Fatalf("expected spaced insertion for empty dramatis personae, got %q", addEdit.NewText)
	}
}

func TestForceCharacterCueEdit_SkipsLeadingWhitespace(t *testing.T) {
	// The lexer anchors character-token ranges at column 0 of the raw line
	// regardless of leading whitespace, so the quick fix must target the
	// first non-whitespace column. Otherwise an indented cue like
	// "\tHAMLET" would become "@\tHAMLET" and the parser would record the
	// character name as "\tHAMLET" instead of "HAMLET".
	cases := []struct {
		name         string
		line         string
		wantInsertAt uint32
	}{
		{name: "no indent", line: "GHOST", wantInsertAt: 0},
		{name: "single space", line: " GHOST", wantInsertAt: 1},
		{name: "tab indent", line: "\tGHOST", wantInsertAt: 1},
		{name: "mixed tabs and spaces", line: " \t GHOST", wantInsertAt: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.line + "\n"
			// The diagnostic's range has no effect on the column choice —
			// the helper derives it from the line contents — but pass in
			// column 0 to mirror what the lexer actually produces.
			r := protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: uint32(len(tc.line))},
			}
			edit, ok := forceCharacterCueEdit(content, r)
			if !ok {
				t.Fatal("expected an edit")
			}
			if edit.NewText != "@" {
				t.Fatalf("expected NewText=@, got %q", edit.NewText)
			}
			if edit.Range.Start.Character != tc.wantInsertAt || edit.Range.End.Character != tc.wantInsertAt {
				t.Fatalf("expected insertion at column %d, got %+v", tc.wantInsertAt, edit.Range)
			}
		})
	}
}

func TestForceCharacterCueEdit_AlreadyForced(t *testing.T) {
	// Defensive: a stale diagnostic on an already-forced cue should not
	// produce a double-`@`.
	_, ok := forceCharacterCueEdit("@ghost\n", protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 0, Character: 6},
	})
	if ok {
		t.Fatal("expected no edit for already-forced cue")
	}
}

func TestComputeCodeActions_NoDramatisPersonaeOffersOnlyForce(t *testing.T) {
	// Without a Dramatis Personae the "unknown character" diagnostic does
	// not fire in practice. If a stale diagnostic is still present, the
	// force-cue fix is still valid because there is no DP section to edit.
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
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 0},
				End:   protocol.Position{Line: 2, Character: 5},
			},
			Data: map[string]interface{}{
				"character": "GHOST",
			},
		},
	}

	actions := computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diagnostics, diagnostics)
	if len(actions) != 1 {
		t.Fatalf("expected 1 code action (force cue only), got %d", len(actions))
	}
	if actions[0].Title != "Exclude cue from check" {
		t.Fatalf("unexpected title: %q", actions[0].Title)
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
