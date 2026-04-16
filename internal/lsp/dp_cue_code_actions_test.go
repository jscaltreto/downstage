package lsp

import (
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
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

func TestComputeCodeActions_InsertMissingDramatisPersonae_AfterPlayMetadata(t *testing.T) {
	// The first play has a metadata block directly under the heading.
	// Inserting the DP between the heading and the metadata would turn
	// valid metadata into content; insertion must land after the block.
	content := `# My Play
Author: Jane Doe
Opened: 2024-01-01

## ACT I

### SCENE 1

ALICE
Hi.`

	actions := codeActionsFor(t, content)

	var add *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add Dramatis Personae section" {
			add = &actions[i]
			break
		}
	}
	require.NotNil(t, add)

	edits := add.Edit.Changes[protocol.DocumentURI("file:///test.ds")]
	require.Len(t, edits, 1)
	// Metadata ends at line 2 (0-indexed), so insertion should land at
	// line 3 (the blank) or later after blank-skipping — specifically the
	// `## ACT I` line at index 4.
	assert.GreaterOrEqual(t, edits[0].Range.Start.Line, uint32(3))
	assert.True(t, strings.HasPrefix(edits[0].NewText, "## Dramatis Personae\n"))

	// Apply the edit and re-parse to confirm the metadata survives as
	// metadata (not as content under the DP heading).
	lines := strings.Split(content, "\n")
	insertAt := int(edits[0].Range.Start.Line)
	merged := strings.Join(lines[:insertAt], "\n") + "\n" + edits[0].NewText + strings.Join(lines[insertAt:], "\n")
	postDoc, postErrs := parser.Parse([]byte(merged))
	require.Empty(t, postErrs, "post-edit parse should have no errors")
	require.NotNil(t, postDoc)
	require.NotEmpty(t, postDoc.Body)
	play, ok := postDoc.Body[0].(*ast.Section)
	require.True(t, ok, "first body node should still be a Section")
	require.NotNil(t, play.Metadata, "play metadata must still be attached after the edit")
	assert.NotEmpty(t, play.Metadata.Entries)
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

func TestComputeCodeActions_InsertMissingDramatisPersonae_TargetsFirstDialoguePlay(t *testing.T) {
	content := `# Compilation
Author: Editor

## Notes
This is frontmatter prose.

# Play One

ALICE
Hi.`

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
	// The edit should land in the play that actually contains the dialogue,
	// not in the compilation header or notes section.
	assert.GreaterOrEqual(t, edits[0].Range.Start.Line, uint32(7))
	assert.True(t, strings.HasPrefix(edits[0].NewText, "## Dramatis Personae\n"))
}
