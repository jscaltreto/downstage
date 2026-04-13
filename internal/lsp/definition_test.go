package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

func TestComputeDefinition_NoDocument(t *testing.T) {
	result := computeDefinition(nil, nil, "file:///test.ds", protocol.Position{})
	if result != nil {
		t.Error("expected nil for nil doc")
	}
}

func TestComputeDefinition_Found(t *testing.T) {
	charRange := token.Range{
		Start: token.Position{Line: 2, Column: 0},
		End:   token.Position{Line: 2, Column: 20},
	}

	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{
						Name:  "HAMLET",
						Range: charRange,
					},
				},
			},
			&ast.Dialogue{
				Character: "HAMLET",
				Range: token.Range{
					Start: token.Position{Line: 10, Column: 0},
					End:   token.Position{Line: 12, Column: 0},
				},
			},
		},
	}

	uri := protocol.DocumentURI("file:///test.ds")
	loc := computeDefinition(doc, nil, uri, protocol.Position{Line: 10, Character: 3})
	if loc == nil {
		t.Fatal("expected location")
	}
	if loc.URI != uri {
		t.Errorf("expected URI %s, got %s", uri, loc.URI)
	}
	if loc.Range.Start.Line != 2 || loc.Range.Start.Character != 0 {
		t.Errorf("expected start at 2:0, got %d:%d", loc.Range.Start.Line, loc.Range.Start.Character)
	}
}

func TestComputeDefinition_NotOnCharacter(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{},
	}

	loc := computeDefinition(doc, nil, "file:///test.ds", protocol.Position{Line: 0, Character: 0})
	if loc != nil {
		t.Error("expected nil when not on character")
	}
}

func TestComputeDefinition_DualDialogueRangeExcludesCaret(t *testing.T) {
	doc, errs := parser.Parse([]byte(`# Play

## Dramatis Personae
HAMLET

## ACT I

### SCENE 1

HAMLET ^
Hello.`))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	uri := protocol.DocumentURI("file:///test.ds")
	assertNil := computeDefinition(doc, nil, uri, protocol.Position{Line: 9, Character: 7})
	if assertNil != nil {
		t.Fatal("expected no definition when cursor is on the trailing caret")
	}

	loc := computeDefinition(doc, nil, uri, protocol.Position{Line: 9, Character: 5})
	if loc == nil {
		t.Fatal("expected definition on the character name")
	}
}
