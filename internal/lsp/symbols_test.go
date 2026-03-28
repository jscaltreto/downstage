package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

func TestComputeDocumentSymbols_Nil(t *testing.T) {
	symbols := computeDocumentSymbols(nil, nil)
	if symbols != nil {
		t.Errorf("expected nil for nil doc, got %v", symbols)
	}
}

func TestComputeDocumentSymbols_Act(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionAct,
				Title: "Act I",
				Children: []ast.Node{
					&ast.Section{
						Kind:  ast.SectionScene,
						Title: "Scene 1",
						Children: []ast.Node{
							&ast.Dialogue{
								Character: "HAMLET",
								Range: token.Range{
									Start: token.Position{Line: 5, Column: 0},
									End:   token.Position{Line: 7, Column: 0},
								},
							},
						},
						Range: token.Range{
							Start: token.Position{Line: 3, Column: 0},
							End:   token.Position{Line: 10, Column: 0},
						},
					},
				},
				Range: token.Range{
					Start: token.Position{Line: 1, Column: 0},
					End:   token.Position{Line: 20, Column: 0},
				},
			},
		},
	}

	symbols := computeDocumentSymbols(doc, nil)
	if len(symbols) != 1 {
		t.Fatalf("expected 1 top-level symbol, got %d", len(symbols))
	}

	actSym := symbols[0]
	if actSym.Name != "Act I" {
		t.Errorf("expected act name %q, got %q", "Act I", actSym.Name)
	}
	if actSym.Kind != protocol.SymbolKindNamespace {
		t.Errorf("expected kind Namespace, got %v", actSym.Kind)
	}
	if len(actSym.Children) != 1 {
		t.Fatalf("expected 1 child (scene), got %d", len(actSym.Children))
	}

	sceneSym := actSym.Children[0]
	if sceneSym.Name != "Scene 1" {
		t.Errorf("expected scene name %q, got %q", "Scene 1", sceneSym.Name)
	}
	if sceneSym.Kind != protocol.SymbolKindClass {
		t.Errorf("expected kind Class, got %v", sceneSym.Kind)
	}
	if len(sceneSym.Children) != 1 {
		t.Fatalf("expected 1 character child, got %d", len(sceneSym.Children))
	}
	if sceneSym.Children[0].Name != "HAMLET" {
		t.Errorf("expected character name %q, got %q", "HAMLET", sceneSym.Children[0].Name)
	}
}

func TestComputeDocumentSymbols_FlatDialogue(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "ROMEO",
				Range: token.Range{
					Start: token.Position{Line: 0, Column: 0},
					End:   token.Position{Line: 2, Column: 0},
				},
			},
		},
	}

	symbols := computeDocumentSymbols(doc, nil)
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "ROMEO" {
		t.Errorf("expected %q, got %q", "ROMEO", symbols[0].Name)
	}
	if symbols[0].Kind != protocol.SymbolKindFunction {
		t.Errorf("expected Function kind, got %v", symbols[0].Kind)
	}
}
