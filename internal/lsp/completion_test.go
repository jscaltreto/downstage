package lsp

import (
	"sort"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeCompletion_CharacterCueContext(t *testing.T) {
	content := `# Play

## Dramatis Personae
HAMLET
OPHELIA

## ACT I

### SCENE 1

HA`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 10, Character: 2})
	if len(result.Items) != 1 {
		labels := make([]string, 0, len(result.Items))
		for _, item := range result.Items {
			labels = append(labels, item.Label)
		}
		t.Fatalf("expected 1 completion item, got %d: %v", len(result.Items), labels)
	}
	if result.Items[0].Label != "HAMLET" {
		t.Fatalf("expected HAMLET completion, got %q", result.Items[0].Label)
	}
}

func TestComputeCompletion_DialogueLineReturnsNothing(t *testing.T) {
	content := `# Play

## Dramatis Personae
HAMLET

## ACT I

### SCENE 1

HAMLET
The sto`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 10, Character: 7})
	if len(result.Items) != 0 {
		t.Fatalf("expected no completion items in dialogue text, got %d", len(result.Items))
	}
}

func TestComputeCompletion_ForcedCharacterCue(t *testing.T) {
	content := `# Play

## Dramatis Personae
HAMLET

## ACT I

### SCENE 1

@HA`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 9, Character: 3})
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 completion item, got %d", len(result.Items))
	}
	if result.Items[0].Label != "@HAMLET" {
		t.Fatalf("expected @HAMLET completion, got %q", result.Items[0].Label)
	}
}

func TestComputeCompletion_DramatisPersonaeReturnsNothing(t *testing.T) {
	content := `# Play

## Dramatis Personae
HA`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 3, Character: 2})
	if len(result.Items) != 0 {
		t.Fatalf("expected no completion items in dramatis personae, got %d", len(result.Items))
	}
}

func TestSceneCompletionCandidates_RankSpeakersByRecencyWithLastSpeakerLast(t *testing.T) {
	content := `# Play

## Dramatis Personae
ADAM
EVE

## ACT I

### SCENE 1

SERPENT
Temptation.

EVE
What do you mean?

S`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	index := newDocumentIndex(doc)
	labels, ok := sceneCompletionCandidates(doc, index, content, 16)
	if !ok {
		t.Fatal("expected scene completion candidates")
	}

	expected := []string{"SERPENT", "EVE", "ADAM"}
	if len(labels) != len(expected) {
		t.Fatalf("expected %d completion candidates, got %d", len(expected), len(labels))
	}
	for i := range expected {
		if labels[i] != expected[i] {
			t.Fatalf("expected completion order %v, got %v", expected, labels)
		}
	}
}

func TestComputeCompletion_UsesSortTextToPreserveSceneOrder(t *testing.T) {
	content := `# Play

## Dramatis Personae
ADAM
EVE

## ACT I

### SCENE 1

SERPENT
Temptation.

EVE
What do you mean?

`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 16, Character: 0})
	expectedLabels := []string{"SERPENT", "EVE", "ADAM"}
	expectedSortText := []string{"0000:SERPENT", "0001:EVE", "0002:ADAM"}

	if len(result.Items) < len(expectedLabels) {
		t.Fatalf("expected at least %d completion items, got %d", len(expectedLabels), len(result.Items))
	}

	for i := range expectedLabels {
		if result.Items[i].Label != expectedLabels[i] {
			t.Fatalf("expected labels %v, got first items %v, %v, %v", expectedLabels, result.Items[0].Label, result.Items[1].Label, result.Items[2].Label)
		}
		if result.Items[i].SortText != expectedSortText[i] {
			t.Fatalf("expected sortText %v, got %q at index %d", expectedSortText, result.Items[i].SortText, i)
		}
	}
}

func TestComputeCompletion_H1HeadingSuggestsDramatisPersonae(t *testing.T) {
	content := `# Dr`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 0, Character: 4})
	if len(result.Items) != 0 {
		t.Fatalf("expected no H1 heading completions, got %d", len(result.Items))
	}
}

func TestComputeCompletion_H2HeadingSuggestsDramatisPersonaeWhenMissing(t *testing.T) {
	content := `# Play

## Dr`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 2, Character: 5})
	if len(result.Items) == 0 {
		t.Fatal("expected heading completions")
	}
	if result.Items[0].Label != "## Dramatis Personae" {
		t.Fatalf("expected dramatis personae heading, got %q", result.Items[0].Label)
	}
}

func TestComputeCompletion_H2HeadingSuggestsNextAct(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

## AC`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 6, Character: 5})
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 heading completion, got %d", len(result.Items))
	}
	if result.Items[0].Label != "## ACT II" {
		t.Fatalf("expected next act heading, got %q", result.Items[0].Label)
	}
}

func TestComputeCompletion_H3HeadingSuggestsNextSceneWithinAct(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

### SC`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 6, Character: 6})
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 heading completion, got %d", len(result.Items))
	}
	if result.Items[0].Label != "### SCENE 2" {
		t.Fatalf("expected next scene heading, got %q", result.Items[0].Label)
	}
}

func TestDocumentIndex_CachesActsScenesAndCueLines(t *testing.T) {
	content := `# Play

## Dramatis Personae
HAMLET

## ACT I

### SCENE 1

HAMLET
To be.

## ACT II

### SCENE 1

OPHELIA
To listen.`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	index := newDocumentIndex(doc)
	if len(index.acts) != 2 {
		t.Fatalf("expected 2 acts, got %d", len(index.acts))
	}
	if len(index.scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(index.scenes))
	}
	if !index.isCharacterCueLine(9) {
		t.Fatal("expected HAMLET line to be indexed as a cue line")
	}
	if !index.isCharacterCueLine(16) {
		t.Fatal("expected OPHELIA line to be indexed as a cue line")
	}
	if got := index.sceneForLine(16); got == nil || got.Range.Start.Line != 14 {
		t.Fatalf("expected scene lookup to return the second scene, got %#v", got)
	}
}

func TestComputeCompletionWithIndex_SuggestsNextActFromCache(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

## AC`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletionWithIndex(doc, newDocumentIndex(doc), content, protocol.Position{Line: 6, Character: 5})
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 heading completion, got %d", len(result.Items))
	}
	if result.Items[0].Label != "## ACT II" {
		t.Fatalf("expected next act heading from cached acts, got %q", result.Items[0].Label)
	}
}

func TestComputeCompletion_H2HeadingSuggestsNextActWithinCurrentPlay(t *testing.T) {
	content := `# Compilation

# Sub Play 1

## ACT I

# Sub Play 2

## AC`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletionWithIndex(doc, newDocumentIndex(doc), content, protocol.Position{Line: 8, Character: 5})
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 heading completion, got %d", len(result.Items))
	}
	if result.Items[0].Label != "## ACT I" {
		t.Fatalf("expected next act heading to reset within play, got %q", result.Items[0].Label)
	}
}

func TestDocumentIndex_SortsSceneSpeakerCuesByLine(t *testing.T) {
	index := &documentIndex{
		sceneSpeakers: map[*ast.Section][]sceneSpeakerCue{},
	}
	scene := &ast.Section{}
	index.sceneSpeakers[scene] = []sceneSpeakerCue{
		{line: 9, name: "SECOND"},
		{line: 3, name: "FIRST"},
	}

	sort.Slice(index.sceneSpeakers[scene], func(i, j int) bool {
		return index.sceneSpeakers[scene][i].line < index.sceneSpeakers[scene][j].line
	})

	speakers := index.sceneSpeakersBeforeLine(scene, 10)
	expected := []string{"FIRST", "SECOND"}
	if len(speakers) != len(expected) {
		t.Fatalf("expected %d speakers, got %d", len(expected), len(speakers))
	}
	for i := range expected {
		if speakers[i] != expected[i] {
			t.Fatalf("expected speakers %v, got %v", expected, speakers)
		}
	}
}
