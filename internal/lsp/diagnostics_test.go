package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

func TestBuildDiagnostics_Nil(t *testing.T) {
	diags := buildDiagnostics(nil, nil)
	if diags != nil {
		t.Errorf("expected nil diagnostics for nil doc and no errors, got %d", len(diags))
	}
}

func TestBuildDiagnostics_ParserErrors(t *testing.T) {
	doc := &ast.Document{}
	errors := []*parser.ParseError{
		{
			Message: "unexpected token",
			Range: token.Range{
				Start: token.Position{Line: 5, Column: 0},
				End:   token.Position{Line: 5, Column: 10},
			},
		},
	}

	diags := buildDiagnostics(doc, errors)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Severity != protocol.DiagnosticSeverityError {
		t.Errorf("expected error severity, got %v", diags[0].Severity)
	}
	if diags[0].Message != "unexpected token" {
		t.Errorf("expected message %q, got %q", "unexpected token", diags[0].Message)
	}
	if diags[0].Source != "downstage" {
		t.Errorf("expected source %q, got %q", "downstage", diags[0].Source)
	}
}

func TestBuildDiagnostics_UnknownCharacter(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{Name: "HAMLET"},
				},
			},
			&ast.Dialogue{
				Character: "GHOST",
				Range: token.Range{
					Start: token.Position{Line: 10, Column: 0},
					End:   token.Position{Line: 12, Column: 0},
				},
			},
		},
	}

	diags := buildDiagnostics(doc, nil)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Severity != protocol.DiagnosticSeverityWarning {
		t.Errorf("expected warning severity, got %v", diags[0].Severity)
	}
	if diags[0].Message != "unknown character: GHOST" {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestBuildDiagnostics_KnownCharacter(t *testing.T) {
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

	diags := buildDiagnostics(doc, nil)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for known character, got %d", len(diags))
	}
}

func TestBuildDiagnostics_AliasMatch(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{
						Name:    "PRINCE HAMLET",
						Aliases: []string{"HAMLET"},
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

	diags := buildDiagnostics(doc, nil)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for alias match, got %d", len(diags))
	}
}
