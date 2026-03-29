package lsp

import (
	"strconv"
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
	return buildDiagnosticsWithIndex(doc, errors, newDocumentIndex(doc))
}

func buildDiagnosticsWithIndex(doc *ast.Document, errors []*parser.ParseError, index *documentIndex) []protocol.Diagnostic {
	if doc == nil && len(errors) == 0 {
		return []protocol.Diagnostic{}
	}
	if index == nil {
		index = newDocumentIndex(doc)
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
		diags = append(diags, checkUnnumberedSections(index)...)
		diags = append(diags, checkUnknownCharacters(index)...)
	}

	if diags == nil {
		return []protocol.Diagnostic{}
	}

	return diags
}

// checkUnknownCharacters warns when dialogue references a character not in dramatis personae.
func checkUnknownCharacters(index *documentIndex) []protocol.Diagnostic {
	if !index.hasDramatisPersonae {
		return nil
	}

	var diags []protocol.Diagnostic
	for _, ref := range index.dialogues {
		name := strings.ToUpper(ref.dialogue.Character)
		if name == "" {
			continue
		}
		if _, ok := index.knownCharacters[name]; ok {
			continue
		}
		diags = append(diags, protocol.Diagnostic{
			Range:    toLSPRange(ref.dialogue.NameRange()),
			Severity: protocol.DiagnosticSeverityWarning,
			Code:     diagnosticCodeUnknownCharacter,
			Source:   "downstage",
			Message:  "unknown character: " + ref.dialogue.Character + " (add to Dramatis Personae)",
			Data: map[string]string{
				"character": ref.dialogue.Character,
			},
		})
	}
	return diags
}

func checkUnnumberedSections(index *documentIndex) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	for actNumber, act := range index.acts {
		if d := unnumberedActDiagnostic(act, actNumber+1); d != nil {
			diags = append(diags, *d)
		}
	}

	for _, scene := range index.scenes {
		number := index.sceneNumbers[scene]
		if d := unnumberedSceneDiagnostic(scene, number); d != nil {
			diags = append(diags, *d)
		}
	}

	return diags
}

func unnumberedActDiagnostic(section *ast.Section, actNumber int) *protocol.Diagnostic {
	if strings.TrimSpace(section.Number) != "" {
		return nil
	}

	replacement := formatSectionHeading(section, romanNumeral(actNumber))
	return &protocol.Diagnostic{
		Range:    toLSPRange(section.HeadingRange()),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeUnnumberedAct,
		Source:   "downstage",
		Message:  "act headings should be numbered with Roman numerals",
		Data: map[string]string{
			"replacement": replacement,
		},
	}
}

func unnumberedSceneDiagnostic(section *ast.Section, sceneNumber int) *protocol.Diagnostic {
	if strings.TrimSpace(section.Number) != "" {
		return nil
	}

	replacement := formatSectionHeading(section, strconv.Itoa(sceneNumber))
	return &protocol.Diagnostic{
		Range:    toLSPRange(section.HeadingRange()),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeUnnumberedScene,
		Source:   "downstage",
		Message:  "scene headings should be numbered with Arabic numerals",
		Data: map[string]string{
			"replacement": replacement,
		},
	}
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
