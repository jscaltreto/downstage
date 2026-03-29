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

HA`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 6, Character: 2})
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

HAMLET
The sto`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 6, Character: 7})
	if len(result.Items) != 0 {
		t.Fatalf("expected no completion items in dialogue text, got %d", len(result.Items))
	}
}

func TestComputeCompletion_ForcedCharacterCue(t *testing.T) {
	content := `# Dramatis Personae
HAMLET

# Play

@HA`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	result := computeCompletion(doc, errs, content, protocol.Position{Line: 5, Character: 3})
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
