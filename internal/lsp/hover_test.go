package lsp

import (
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/token"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestComputeHover_NoDocument(t *testing.T) {
	result := computeHover(nil, nil, protocol.Position{Line: 0, Character: 0})
	if result != nil {
		t.Error("expected nil hover for nil doc")
	}
}

func TestComputeHover_OnCharacterName(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{
						Name:        "HAMLET",
						Description: "Prince of Denmark",
						Aliases:     []string{"The Prince"},
						Range: token.Range{
							Start: token.Position{Line: 2, Column: 0},
							End:   token.Position{Line: 2, Column: 30},
						},
					},
				},
				Groups: []ast.CharacterGroup{
					{
						Name: "Royalty",
						Characters: []ast.Character{
							{
								Name:        "CLAUDIUS",
								Description: "King of Denmark",
								Range: token.Range{
									Start: token.Position{Line: 4, Column: 0},
									End:   token.Position{Line: 4, Column: 20},
								},
							},
						},
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

	result := computeHover(doc, nil, protocol.Position{Line: 10, Character: 3})
	if result == nil {
		t.Fatal("expected hover result")
	}

	if result.Contents.Kind != protocol.Markdown {
		t.Errorf("expected markdown, got %s", result.Contents.Kind)
	}
	if !strings.Contains(result.Contents.Value, "**HAMLET**") {
		t.Errorf("expected character name in hover, got: %s", result.Contents.Value)
	}
	if !strings.Contains(result.Contents.Value, "Prince of Denmark") {
		t.Errorf("expected description in hover, got: %s", result.Contents.Value)
	}
	if !strings.Contains(result.Contents.Value, "The Prince") {
		t.Errorf("expected alias in hover, got: %s", result.Contents.Value)
	}
}

func TestComputeHover_OnGroupedCharacter(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Groups: []ast.CharacterGroup{
					{
						Name: "Royalty",
						Characters: []ast.Character{
							{
								Name:        "CLAUDIUS",
								Description: "King of Denmark",
								Range: token.Range{
									Start: token.Position{Line: 4, Column: 0},
									End:   token.Position{Line: 4, Column: 20},
								},
							},
						},
					},
				},
			},
			&ast.Dialogue{
				Character: "CLAUDIUS",
				Range: token.Range{
					Start: token.Position{Line: 10, Column: 0},
					End:   token.Position{Line: 12, Column: 0},
				},
			},
		},
	}

	result := computeHover(doc, nil, protocol.Position{Line: 10, Character: 3})
	if result == nil {
		t.Fatal("expected hover result")
	}
	if !strings.Contains(result.Contents.Value, "Royalty") {
		t.Errorf("expected group in hover, got: %s", result.Contents.Value)
	}
}

func TestComputeHover_NotOnCharacter(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{Name: "HAMLET"},
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

	// Position is on a line with no character
	result := computeHover(doc, nil, protocol.Position{Line: 0, Character: 0})
	if result != nil {
		t.Error("expected nil hover when not on character")
	}
}

func TestComputeHover_ForcedCharacterUsesNameRange(t *testing.T) {
	dialogue := &ast.Dialogue{
		Character: "HAMLET",
		Range: token.Range{
			Start: token.Position{Line: 10, Column: 0},
			End:   token.Position{Line: 12, Column: 0},
		},
	}
	dialogue.SetNameRange(token.Range{
		Start: token.Position{Line: 10, Column: 1},
		End:   token.Position{Line: 10, Column: 7},
	})

	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:       ast.SectionDramatisPersonae,
				Characters: []ast.Character{{Name: "HAMLET"}},
			},
			dialogue,
		},
	}

	assert.Nil(t, computeHover(doc, nil, protocol.Position{Line: 10, Character: 0}))
	assert.NotNil(t, computeHover(doc, nil, protocol.Position{Line: 10, Character: 6}))
}
