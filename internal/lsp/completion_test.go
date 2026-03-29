package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeCompletion_CharacterCueContext(t *testing.T) {
	content := `# Dramatis Personae
HAMLET
OPHELIA

# Play

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
	content := `# Dramatis Personae
HAMLET

# Play

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
	content := `# Dramatis Personae
HAMLET

# Play

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
	content := `# Dramatis Personae
HA`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 1, Character: 2})
	if len(result.Items) != 0 {
		t.Fatalf("expected no completion items in dramatis personae, got %d", len(result.Items))
	}
}

func TestSceneCompletionCandidates_RankSpeakersByRecencyWithLastSpeakerLast(t *testing.T) {
	content := `# Dramatis Personae
ADAM
EVE

# Play

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

	labels, ok := sceneCompletionCandidates(doc, content, 16)
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
	content := `# Dramatis Personae
ADAM
EVE

# Play

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
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 heading completion, got %d", len(result.Items))
	}
	if result.Items[0].Label != "# Dramatis Personae" {
		t.Fatalf("expected dramatis personae heading, got %q", result.Items[0].Label)
	}
}

func TestComputeCompletion_H1HeadingSkipsDramatisPersonaeWhenAlreadyPresent(t *testing.T) {
	content := `# Dramatis Personae
HAMLET

# Dr`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 3, Character: 4})
	if len(result.Items) != 0 {
		t.Fatalf("expected no duplicate dramatis personae completion, got %d", len(result.Items))
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
