package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func parseDoc(t *testing.T, content string) ([]protocol.Diagnostic, []*parser.ParseError) {
	t.Helper()
	doc, errs := parser.Parse([]byte(content))
	return buildDiagnostics(doc, errs), errs
}

func TestMissingDramatisPersonae_FiresOnceWhenDialogueHasNoDP(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

ALICE
Hello.

BOB
Hi.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)

	missing := filterDiagnostics(diags, diagnosticCodeMissingDramatisPersonae)
	require.Len(t, missing, 1, "should fire exactly once even with multiple dialogues")
	assert.Equal(t, protocol.DiagnosticSeverityInformation, missing[0].Severity)
}

func TestMissingDramatisPersonae_SuppressedWhenDPPresent(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

ALICE
Hello.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	missing := filterDiagnostics(diags, diagnosticCodeMissingDramatisPersonae)
	assert.Empty(t, missing)
}

func TestMissingDramatisPersonae_SuppressedWhenNoDialogue(t *testing.T) {
	content := `# Play

Just some prose with no dialogue.
`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	missing := filterDiagnostics(diags, diagnosticCodeMissingDramatisPersonae)
	assert.Empty(t, missing)
}

func TestDPDuplicateCharacterName_FlagsSecondOccurrence(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE
ALICE

## ACT I

ALICE
Hi.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	dup := filterDiagnostics(diags, diagnosticCodeDPDuplicateCharacterName)
	require.Len(t, dup, 1, "only the second ALICE entry should flag")
	assert.Equal(t, protocol.DiagnosticSeverityWarning, dup[0].Severity)
	// The flagged line is the second ALICE (0-indexed line 4).
	assert.Equal(t, uint32(4), dup[0].Range.Start.Line)
}

func TestDPDuplicateAlias_FlagsAliasVsAlias(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE/AL
BOB/AL

## ACT I

AL
Hi.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	dup := filterDiagnostics(diags, diagnosticCodeDPDuplicateAlias)
	require.Len(t, dup, 1, "only the second AL alias should flag")
	assert.Equal(t, protocol.DiagnosticSeverityWarning, dup[0].Severity)
}

func TestDPDuplicateAlias_FlagsNameCollidingWithAlias(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE/AL
AL

## ACT I

AL
Hi.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	dup := filterDiagnostics(diags, diagnosticCodeDPDuplicateAlias)
	require.Len(t, dup, 1, "the standalone AL entry should flag as alias collision")
}

func TestDPDuplicateAlias_NoFalsePositiveOnUnrelatedEntries(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE/AL
BOB/BEE

## ACT I

AL
Hi.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	dup := filterDiagnostics(diags, diagnosticCodeDPDuplicateAlias, diagnosticCodeDPDuplicateCharacterName)
	assert.Empty(t, dup)
}

func TestDPCharacterNoDialogue_InfoOnUnusedDPEntry(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE
BOB

## ACT I

ALICE
Hi.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	none := filterDiagnostics(diags, diagnosticCodeDPCharacterNoDialogue)
	require.Len(t, none, 1)
	assert.Equal(t, protocol.DiagnosticSeverityInformation, none[0].Severity)
	assert.Contains(t, none[0].Message, "BOB")
}

func TestDPCharacterNoDialogue_ForcedCueCountsAsUsage(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

@ALICE
Hi.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	none := filterDiagnostics(diags, diagnosticCodeDPCharacterNoDialogue)
	assert.Empty(t, none, "forced cue should satisfy the no-dialogue check")
}

func TestDPCharacterNoDialogue_AliasSatisfiesCheck(t *testing.T) {
	content := `# Play

## Dramatis Personae
PRINCE HAMLET/HAMLET

## ACT I

HAMLET
Hi.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	none := filterDiagnostics(diags, diagnosticCodeDPCharacterNoDialogue)
	assert.Empty(t, none)
}

func TestDPCharacterNoDialogue_ScopedPerPlay(t *testing.T) {
	content := `# First Play

## Dramatis Personae
ALICE

## ACT I

ALICE
Hi.

# Second Play

## Dramatis Personae
ALICE

## ACT I

BOB
Hi.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	none := filterDiagnostics(diags, diagnosticCodeDPCharacterNoDialogue)
	// Second play's ALICE never speaks.
	require.Len(t, none, 1)
	assert.Contains(t, none[0].Message, "ALICE")
}

func TestCueOrphaned_FlagsCueWithNoLines(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE
BOB

## ACT I

### SCENE 1

ALICE

BOB
Hi.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	orphans := filterDiagnostics(diags, diagnosticCodeCueOrphaned)
	require.Len(t, orphans, 1)
	assert.Equal(t, protocol.DiagnosticSeverityWarning, orphans[0].Severity)
	assert.Contains(t, orphans[0].Message, "ALICE")
}

func TestCueOrphaned_SuppressedWhenDialogueFollows(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

ALICE
Hi.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	orphans := filterDiagnostics(diags, diagnosticCodeCueOrphaned)
	assert.Empty(t, orphans)
}

func TestCueConsecutiveSameCharacter_FlagsRepeatedCue(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

### SCENE 1

ALICE
First line.

ALICE
Second line.`

	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	consecutive := filterDiagnostics(diags, diagnosticCodeCueConsecutiveSameCharacter)
	require.Len(t, consecutive, 1)
	assert.Equal(t, protocol.DiagnosticSeverityInformation, consecutive[0].Severity)
}

func TestCueConsecutiveSameCharacter_StageDirectionBreaksChain(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

### SCENE 1

ALICE
First line.

> ALICE paces.

ALICE
Second line.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	consecutive := filterDiagnostics(diags, diagnosticCodeCueConsecutiveSameCharacter)
	assert.Empty(t, consecutive, "a standalone stage direction between cues should reset the chain")
}

func TestCueConsecutiveSameCharacter_SceneBoundaryResets(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

### SCENE 1

ALICE
First line.

### SCENE 2

ALICE
Second line.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	consecutive := filterDiagnostics(diags, diagnosticCodeCueConsecutiveSameCharacter)
	assert.Empty(t, consecutive, "a new scene resets the consecutive-cue check")
}

func TestCueConsecutiveSameCharacter_DifferentCharactersNotFlagged(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE
BOB

## ACT I

### SCENE 1

ALICE
Hi.

BOB
Hello.

ALICE
Hi again.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	consecutive := filterDiagnostics(diags, diagnosticCodeCueConsecutiveSameCharacter)
	assert.Empty(t, consecutive)
}

func TestCueConsecutiveSameCharacter_PageBreakResets(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

### SCENE 1

ALICE
First line.

===

ALICE
Second line.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	consecutive := filterDiagnostics(diags, diagnosticCodeCueConsecutiveSameCharacter)
	assert.Empty(t, consecutive, "explicit page break should reset the chain")
}

func TestCueConsecutiveSameCharacter_CalloutResets(t *testing.T) {
	content := `# Play

## Dramatis Personae
ALICE

## ACT I

### SCENE 1

ALICE
First line.

>> A quick note.

ALICE
Second line.`
	diags, errs := parseDoc(t, content)
	require.Empty(t, errs)
	consecutive := filterDiagnostics(diags, diagnosticCodeCueConsecutiveSameCharacter)
	assert.Empty(t, consecutive, ">> note should reset the chain")
}
