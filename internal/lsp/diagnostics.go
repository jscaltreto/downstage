package lsp

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

// buildDiagnostics converts parser errors and additional warnings into LSP diagnostics.
func buildDiagnostics(doc *ast.Document, errors []*parser.ParseError) []protocol.Diagnostic {
	if doc == nil && len(errors) == 0 {
		return nil
	}

	var diags []protocol.Diagnostic

	// Convert parser errors to diagnostics.
	for _, e := range errors {
		diags = append(diags, protocol.Diagnostic{
			Range:    toLSPRange(e.Range),
			Severity: protocol.DiagnosticSeverityError,
			Source:   "downstage",
			Message:  e.Message,
		})
	}

	// Add warnings for unknown character names.
	if doc != nil {
		diags = append(diags, checkUnknownCharacters(doc)...)
	}

	return diags
}

// checkUnknownCharacters warns when dialogue references a character not in dramatis personae.
func checkUnknownCharacters(doc *ast.Document) []protocol.Diagnostic {
	dp := ast.FindDramatisPersonae(doc.Body)
	if dp == nil {
		return nil
	}

	known := make(map[string]bool)
	for _, ch := range dp.AllCharacters() {
		known[strings.ToUpper(ch.Name)] = true
		for _, alias := range ch.Aliases {
			known[strings.ToUpper(alias)] = true
		}
	}

	var diags []protocol.Diagnostic
	for _, n := range doc.Body {
		diags = append(diags, checkNodeCharacters(n, known)...)
	}
	return diags
}

func checkNodeCharacters(n ast.Node, known map[string]bool) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	switch v := n.(type) {
	case *ast.Dialogue:
		name := strings.ToUpper(v.Character)
		if name != "" && !known[name] {
			diags = append(diags, protocol.Diagnostic{
				Range:    toLSPRange(v.NameRange()),
				Severity: protocol.DiagnosticSeverityWarning,
				Source:   "downstage",
				Message:  "unknown character: " + v.Character,
			})
		}
	case *ast.Song:
		// Songs may contain dialogue as content
		for _, child := range v.Content {
			diags = append(diags, checkNodeCharacters(child, known)...)
		}
	case *ast.Section:
		for _, child := range v.Children {
			diags = append(diags, checkNodeCharacters(child, known)...)
		}
	}

	return diags
}

func toLSPRange(r token.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(r.Start.Line),
			Character: uint32(r.Start.Column),
		},
		End: protocol.Position{
			Line:      uint32(r.End.Line),
			Character: uint32(r.End.Column),
		},
	}
}
