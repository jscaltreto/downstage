package lsp

import (
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func codeActionsFor(t *testing.T, content string) []protocol.CodeAction {
	t.Helper()
	doc, errs := parser.Parse([]byte(content))
	require.Empty(t, errs)
	diags := buildDiagnostics(doc, errs)
	return computeCodeActions(doc, content, protocol.DocumentURI("file:///test.ds"), diags, diags)
}

func TestComputeCodeActions_DeleteDuplicateDPEntry(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE
ALICE

## ACT I

ALICE
Hi.`

	actions := codeActionsFor(t, content)

	var del *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Delete duplicate Dramatis Personae entry" {
			del = &actions[i]
			break
		}
	}
	require.NotNil(t, del, "expected a delete-duplicate action")

	edits := del.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	require.Len(t, edits, 1)
	assert.Equal(t, "", edits[0].NewText)
	// Deletion should consume through the start of the next line so the
	// duplicate row vanishes entirely.
	assert.Equal(t, uint32(4), edits[0].Range.Start.Line)
	assert.Equal(t, uint32(5), edits[0].Range.End.Line)
	assert.Equal(t, uint32(0), edits[0].Range.Start.Character)
	assert.Equal(t, uint32(0), edits[0].Range.End.Character)
}

func TestComputeCodeActions_InsertMissingDramatisPersonae(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

ALICE
Hi.

BOB
Hello.`

	actions := codeActionsFor(t, content)

	var add *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add Dramatis Personae section" {
			add = &actions[i]
			break
		}
	}
	require.NotNil(t, add, "expected an add-DP-section action")

	edits := add.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	require.Len(t, edits, 1)

	assert.True(t, strings.HasPrefix(edits[0].NewText, "## Dramatis Personae\n"), "edit should start with DP heading, got %q", edits[0].NewText)
	assert.Contains(t, edits[0].NewText, "ALICE")
	assert.Contains(t, edits[0].NewText, "BOB")
	// ALICE should come before BOB (first-appearance order).
	assert.Less(t, strings.Index(edits[0].NewText, "ALICE"), strings.Index(edits[0].NewText, "BOB"))
}

func TestComputeCodeActions_InsertMissingDramatisPersonae_NoHeading(t *testing.T) {
	content := `ALICE
Hi.

BOB
Hello.`

	actions := codeActionsFor(t, content)

	var add *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add Dramatis Personae section" {
			add = &actions[i]
			break
		}
	}
	require.NotNil(t, add, "expected an add-DP-section action even without a play heading")

	edits := add.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	require.Len(t, edits, 1)
	// Insertion falls back to the top of the document.
	assert.Equal(t, uint32(0), edits[0].Range.Start.Line)
	assert.True(t, strings.HasPrefix(edits[0].NewText, "## Dramatis Personae\n"))
}
