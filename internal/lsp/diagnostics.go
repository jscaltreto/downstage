package lsp

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

const (
	diagnosticCodeUnknownCharacter = "unknown-character"
	diagnosticCodeUnnumberedAct    = "unnumbered-act"
	diagnosticCodeUnnumberedScene  = "unnumbered-scene"
)

// buildDiagnostics converts parser errors and additional warnings into LSP diagnostics.
func buildDiagnostics(doc *ast.Document, errors []*parser.ParseError) []protocol.Diagnostic {
	if doc == nil && len(errors) == 0 {
		return []protocol.Diagnostic{}
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
		diags = append(diags, checkUnnumberedSections(doc)...)
		diags = append(diags, checkUnknownCharacters(doc)...)
	}

	if diags == nil {
		return []protocol.Diagnostic{}
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

func checkUnnumberedSections(doc *ast.Document) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	actCount := 0
	sceneCountOutsideActs := 0

	for _, node := range doc.Body {
		diags = append(diags, checkUnnumberedSectionsInNode(node, nil, &actCount, &sceneCountOutsideActs)...)
	}

	return diags
}

func checkUnnumberedSectionsInNode(
	node ast.Node,
	sceneCountInAct *int,
	actCount *int,
	sceneCountOutsideActs *int,
) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	switch v := node.(type) {
	case *ast.Section:
		switch v.Kind {
		case ast.SectionAct:
			*actCount = *actCount + 1
			diags = append(diags, unnumberedActDiagnostic(v, *actCount)...)

			sceneCount := 0
			for _, child := range v.Children {
				diags = append(diags, checkUnnumberedSectionsInNode(child, &sceneCount, actCount, sceneCountOutsideActs)...)
			}
		case ast.SectionScene:
			if sceneCountInAct != nil {
				*sceneCountInAct = *sceneCountInAct + 1
				diags = append(diags, unnumberedSceneDiagnostic(v, *sceneCountInAct)...)
			} else {
				*sceneCountOutsideActs = *sceneCountOutsideActs + 1
				diags = append(diags, unnumberedSceneDiagnostic(v, *sceneCountOutsideActs)...)
			}

			for _, child := range v.Children {
				diags = append(diags, checkUnnumberedSectionsInNode(child, sceneCountInAct, actCount, sceneCountOutsideActs)...)
			}
		default:
			for _, child := range v.Children {
				diags = append(diags, checkUnnumberedSectionsInNode(child, sceneCountInAct, actCount, sceneCountOutsideActs)...)
			}
		}
	case *ast.Song:
		for _, child := range v.Content {
			diags = append(diags, checkUnnumberedSectionsInNode(child, sceneCountInAct, actCount, sceneCountOutsideActs)...)
		}
	}

	return diags
}

func unnumberedActDiagnostic(section *ast.Section, actNumber int) []protocol.Diagnostic {
	if strings.TrimSpace(section.Number) != "" {
		return nil
	}

	replacement := formatSectionHeading(section, romanNumeral(actNumber))
	return []protocol.Diagnostic{{
		Range:    toLSPRange(section.HeadingRange()),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeUnnumberedAct,
		Source:   "downstage",
		Message:  "act headings should be numbered with Roman numerals",
		Data: map[string]string{
			"replacement": replacement,
		},
	}}
}

func unnumberedSceneDiagnostic(section *ast.Section, sceneNumber int) []protocol.Diagnostic {
	if strings.TrimSpace(section.Number) != "" {
		return nil
	}

	replacement := formatSectionHeading(section, fmt.Sprintf("%d", sceneNumber))
	return []protocol.Diagnostic{{
		Range:    toLSPRange(section.HeadingRange()),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeUnnumberedScene,
		Source:   "downstage",
		Message:  "scene headings should be numbered with Arabic numerals",
		Data: map[string]string{
			"replacement": replacement,
		},
	}}
}

func formatSectionHeading(section *ast.Section, number string) string {
	marker := strings.Repeat("#", section.Level)
	title := strings.TrimSpace(section.Title)

	switch section.Kind {
	case ast.SectionAct:
		return marker + " " + buildNumberedHeader("ACT", number, title)
	case ast.SectionScene:
		return marker + " " + buildNumberedHeader("SCENE", number, title)
	default:
		return marker + " " + title
	}
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
				Code:     diagnosticCodeUnknownCharacter,
				Source:   "downstage",
				Message:  "unknown character: " + v.Character + " (add to Dramatis Personae)",
				Data: map[string]string{
					"character": v.Character,
				},
			})
		}
	case *ast.DualDialogue:
		diags = append(diags, checkNodeCharacters(v.Left, known)...)
		diags = append(diags, checkNodeCharacters(v.Right, known)...)
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
