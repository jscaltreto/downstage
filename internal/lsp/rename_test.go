package lsp

import (
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

const renameURI protocol.DocumentURI = "file:///rename.ds"

func TestPrepareRename_OnDramatisPersonaePrimaryName(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB - The good guy
JANE

## ACT I

### SCENE 1

BOB
Hello.
`
	doc, _ := parser.Parse([]byte(src))

	r := computePrepareRename(doc, protocol.Position{Line: 4, Character: 1})
	if r == nil {
		t.Fatal("expected rename range on DP primary name")
	}
	if r.Start.Line != 4 || r.Start.Character != 0 || r.End.Character != 3 {
		t.Errorf("unexpected range %+v", r)
	}
}

func TestPrepareRename_OnAliasDeclaration(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB/B - The good guy

## ACT I

### SCENE 1

B
Hello.
`
	doc, _ := parser.Parse([]byte(src))

	r := computePrepareRename(doc, protocol.Position{Line: 4, Character: 4})
	if r == nil {
		t.Fatal("expected rename range on alias")
	}
	if r.Start.Character != 4 || r.End.Character != 5 {
		t.Errorf("unexpected alias range %+v", r)
	}
}

func TestPrepareRename_OnDialogueCue(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hello.
`
	doc, _ := parser.Parse([]byte(src))

	r := computePrepareRename(doc, protocol.Position{Line: 10, Character: 1})
	if r == nil {
		t.Fatal("expected rename range on cue")
	}
	if r.Start.Line != 10 || r.Start.Character != 0 || r.End.Character != 3 {
		t.Errorf("unexpected cue range %+v", r)
	}
}

func TestPrepareRename_RefusesOnDialogueBody(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hello.
`
	doc, _ := parser.Parse([]byte(src))

	r := computePrepareRename(doc, protocol.Position{Line: 11, Character: 1})
	if r != nil {
		t.Errorf("expected no rename on dialogue body, got %+v", r)
	}
}

func TestPrepareRename_RefusesOnConjunctionCue(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB
JANE

## ACT I

### SCENE 1

BOB AND JANE
Hello.
`
	doc, _ := parser.Parse([]byte(src))

	r := computePrepareRename(doc, protocol.Position{Line: 11, Character: 1})
	if r != nil {
		t.Errorf("expected no rename on conjunction cue, got %+v", r)
	}
}

func TestPrepareRename_RefusesOnPlainProse(t *testing.T) {
	src := `# Notes

Just some prose mentioning BOB.
`
	doc, _ := parser.Parse([]byte(src))

	r := computePrepareRename(doc, protocol.Position{Line: 2, Character: 28})
	if r != nil {
		t.Errorf("expected no rename on prose, got %+v", r)
	}
}

func TestComputeRename_UpdatesDeclarationAndCues(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB - The good guy

## ACT I

### SCENE 1

BOB
Hello.

BOB
Again.
`
	doc, _ := parser.Parse([]byte(src))

	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if edit == nil {
		t.Fatal("expected workspace edit")
	}
	edits := edit.Changes[renameURI]
	if got, want := len(edits), 3; got != want {
		t.Fatalf("expected %d edits (1 decl + 2 cues), got %d: %+v", want, got, edits)
	}
	for _, e := range edits {
		if e.NewText != "ROBERT" {
			t.Errorf("expected NewText ROBERT, got %q", e.NewText)
		}
	}
}

func TestComputeRename_PreservesAliasWhenRenamingPrimary(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB/B - The good guy

## ACT I

### SCENE 1

BOB
Hi.

B
Yo.
`
	doc, _ := parser.Parse([]byte(src))

	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	edits := edit.Changes[renameURI]
	// Expect: BOB declaration + the BOB cue. The B alias and B cue stay put.
	if got, want := len(edits), 2; got != want {
		t.Fatalf("expected %d edits, got %d: %+v", want, got, edits)
	}
	for _, e := range edits {
		if !strings.EqualFold(e.NewText, "ROBERT") {
			t.Errorf("expected NewText ROBERT, got %q", e.NewText)
		}
	}
}

func TestComputeRename_AliasOnly(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB/B - The good guy

## ACT I

### SCENE 1

BOB
Hi.

B
Yo.
`
	doc, _ := parser.Parse([]byte(src))

	// Cursor on the alias B in the DP entry (column 4).
	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 4}, "BOBBY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	edits := edit.Changes[renameURI]
	// Expect: alias declaration + the B cue; BOB cue and BOB primary untouched.
	if got, want := len(edits), 2; got != want {
		t.Fatalf("expected %d edits, got %d: %+v", want, got, edits)
	}
}

func TestComputeRename_CuePosition(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hi.
`
	doc, _ := parser.Parse([]byte(src))

	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 10, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(edit.Changes[renameURI]), 2; got != want {
		t.Fatalf("expected %d edits from cue trigger, got %d", want, got)
	}
}

func TestComputeRename_RejectsConflictWithExistingName(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB
JANE

## ACT I

### SCENE 1

BOB
Hi.
`
	doc, _ := parser.Parse([]byte(src))

	_, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "JANE")
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestComputeRename_RejectsConflictWithAlias(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB/B
JANE

## ACT I

### SCENE 1

BOB
Hi.
`
	doc, _ := parser.Parse([]byte(src))

	_, err := computeRename(doc, renameURI, protocol.Position{Line: 5, Character: 1}, "B")
	if err == nil {
		t.Fatal("expected conflict error against existing alias")
	}
}

func TestComputeRename_RejectsInvalidName(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hi.
`
	doc, _ := parser.Parse([]byte(src))

	cases := map[string]string{
		"empty":           "",
		"slash":           "BOB/X",
		"description sep": "BOB - X",
		"punctuation":     "BOB!",
		"digits only":     "1234",
	}
	for name, newName := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, newName)
			if err == nil {
				t.Fatalf("expected error for %q", newName)
			}
		})
	}
}

func TestComputeRename_PreservesCueCasing(t *testing.T) {
	// The forced cue prefix lets writers cue characters in non-canonical
	// case. Rename should preserve each cue's existing case style.
	src := `# Play

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hi.

@Bob
Hi too.
`
	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "Robert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	edits := edit.Changes[renameURI]
	gotTexts := make(map[string]int)
	for _, e := range edits {
		gotTexts[e.NewText]++
	}
	if gotTexts["ROBERT"] < 1 {
		t.Errorf("expected ALL-CAPS cue to stay ALL CAPS, got %+v", gotTexts)
	}
	if gotTexts["Robert"] < 1 {
		t.Errorf("expected mixed-case cue to keep typed case, got %+v", gotTexts)
	}
}

func TestComputeRename_DualDialogue(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB
JANE

## ACT I

### SCENE 1

BOB ^
Hi.

JANE
Hello.
`
	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(edit.Changes[renameURI]), 2; got != want {
		t.Fatalf("expected %d edits across DP + dual cue, got %d: %+v", want, got, edit.Changes[renameURI])
	}
}

func TestComputeRename_SongDialogue(t *testing.T) {
	src := `# Play

## Dramatis Personae

BOB

## ACT I

### SCENE 1

SONG: Opening

BOB
Sing it.

SONG END
`
	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(edit.Changes[renameURI]), 2; got != want {
		t.Fatalf("expected %d edits across DP + song cue, got %d: %+v", want, got, edit.Changes[renameURI])
	}
}

func TestComputeRename_CompilationScopesToOwningPlay(t *testing.T) {
	// Each play in a compilation has its own dramatis personae. Renaming
	// BOB in Play A must not rewrite the (different) BOB in Play B.
	src := `# Play A

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hello from A.

# Play B

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hello from B.
`
	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	// Cursor on Play A's DP entry.
	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	edits := edit.Changes[renameURI]
	// Expect: Play A DP entry + Play A cue. Play B's DP entry and cue stay put.
	if got, want := len(edits), 2; got != want {
		t.Fatalf("expected %d edits scoped to Play A, got %d: %+v", want, got, edits)
	}
	for _, e := range edits {
		if e.Range.Start.Line >= 13 {
			t.Errorf("edit leaked into Play B at line %d: %+v", e.Range.Start.Line, e)
		}
	}
}

func TestComputeRename_CompilationCueScopesToOwningPlay(t *testing.T) {
	// Triggering rename from a cue should also stay within that cue's
	// owning play.
	src := `# Play A

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hello from A.

# Play B

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hello from B.
`
	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	// Cursor on Play B's cue (line 23).
	edit, err := computeRename(doc, renameURI, protocol.Position{Line: 23, Character: 1}, "ROBERT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	edits := edit.Changes[renameURI]
	if got, want := len(edits), 2; got != want {
		t.Fatalf("expected %d edits scoped to Play B, got %d: %+v", want, got, edits)
	}
	for _, e := range edits {
		if e.Range.Start.Line < 13 {
			t.Errorf("edit leaked into Play A at line %d: %+v", e.Range.Start.Line, e)
		}
	}
}

func TestComputeRename_CompilationConflictScopedToPlay(t *testing.T) {
	// A name that exists in another play should NOT be flagged as a
	// conflict; each play's DP scope is independent.
	src := `# Play A

## Dramatis Personae

BOB

## ACT I

### SCENE 1

BOB
Hi.

# Play B

## Dramatis Personae

JANE

## ACT I

### SCENE 1

JANE
Hi.
`
	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	// Renaming Play A's BOB → JANE is fine because Play A has no JANE.
	if _, err := computeRename(doc, renameURI, protocol.Position{Line: 4, Character: 1}, "JANE"); err != nil {
		t.Fatalf("expected rename to succeed across plays, got %v", err)
	}
}

func TestPrepareRename_NilDoc(t *testing.T) {
	if r := computePrepareRename(nil, protocol.Position{}); r != nil {
		t.Errorf("expected nil for nil doc, got %+v", r)
	}
}

func TestComputeRename_NilDoc(t *testing.T) {
	if _, err := computeRename(nil, renameURI, protocol.Position{}, "X"); err == nil {
		t.Error("expected error for nil doc")
	}
}
