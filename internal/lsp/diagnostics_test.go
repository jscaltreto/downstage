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
	if diags == nil {
		t.Fatal("expected empty diagnostics slice for nil doc and no errors")
	}
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for nil doc and no errors, got %d", len(diags))
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
	if diags[0].Message != "unknown character: GHOST (add to Dramatis Personae)" {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestBuildDiagnostics_NoDramatisPersonaeSuppressesUnknownCharacter(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "GHOST",
				Range: token.Range{
					Start: token.Position{Line: 2, Column: 0},
					End:   token.Position{Line: 4, Column: 0},
				},
			},
		},
	}

	diags := buildDiagnostics(doc, nil)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics without dramatis personae, got %d", len(diags))
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

func TestBuildDiagnostics_UnnumberedActAndSceneWarnings(t *testing.T) {
	content := `# Play

## ACT: Prologue

### The Kitchen`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diags := buildDiagnostics(doc, errs)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diags))
	}

	if diags[0].Code != diagnosticCodeUnnumberedAct {
		t.Fatalf("expected first diagnostic to be unnumbered act, got %v", diags[0].Code)
	}
	if diags[0].Message != "act headings should be numbered with Roman numerals" {
		t.Fatalf("unexpected act diagnostic message: %q", diags[0].Message)
	}
	if data, ok := diags[0].Data.(map[string]string); !ok || data["replacement"] != "## ACT I: Prologue" {
		t.Fatalf("unexpected act diagnostic data: %#v", diags[0].Data)
	}

	if diags[1].Code != diagnosticCodeUnnumberedScene {
		t.Fatalf("expected second diagnostic to be unnumbered scene, got %v", diags[1].Code)
	}
	if diags[1].Message != "scene headings should be numbered with Arabic numerals" {
		t.Fatalf("unexpected scene diagnostic message: %q", diags[1].Message)
	}
	if data, ok := diags[1].Data.(map[string]string); !ok || data["replacement"] != "### SCENE 1: The Kitchen" {
		t.Fatalf("unexpected scene diagnostic data: %#v", diags[1].Data)
	}
}

func TestBuildDiagnostics_UnnumberedSceneInActResetsNumbering(t *testing.T) {
	content := `# Play

## ACT I

### SCENE 1

## First Interlude

## ACT II

## Second Interlude`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	diags := buildDiagnostics(doc, errs)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diags))
	}

	firstData, ok := diags[0].Data.(map[string]string)
	if !ok {
		t.Fatalf("unexpected first diagnostic data: %#v", diags[0].Data)
	}
	if firstData["replacement"] != "## SCENE 2: First Interlude" {
		t.Fatalf("unexpected first scene replacement: %q", firstData["replacement"])
	}

	secondData, ok := diags[1].Data.(map[string]string)
	if !ok {
		t.Fatalf("unexpected second diagnostic data: %#v", diags[1].Data)
	}
	if secondData["replacement"] != "## SCENE 1: Second Interlude" {
		t.Fatalf("unexpected second scene replacement: %q", secondData["replacement"])
	}
}
