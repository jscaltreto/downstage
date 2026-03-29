package lsp

import (
	"fmt"
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
	diagnosticCodeMisnumberedAct   = "misnumbered-act"
	diagnosticCodeMisnumberedScene = "misnumbered-scene"
)

// collectiveCues are conventional ensemble cue names that should not
// produce unknown-character warnings even if absent from Dramatis Personae.
// Keys must be uppercase.
var collectiveCues = map[string]bool{
	"ALL":      true,
	"CHORUS":   true,
	"ENSEMBLE": true,
}

// conjunctionSeps are the delimiters used to split multi-speaker cues.
// Surrounding spaces prevent matching substrings like "SANDY".
var conjunctionSeps = []string{" AND ", " & "}

// splitConjunctionCue splits a cue like "BOB AND JANE" or "BOB & JANE"
// into individual names. Returns nil if no conjunction is found.
func splitConjunctionCue(name string) []string {
	upper := strings.ToUpper(name)

	var parts []string
	offset := 0
	for offset < len(upper) {
		bestIdx := -1
		bestLen := 0
		for _, sep := range conjunctionSeps {
			if idx := strings.Index(upper[offset:], sep); idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
				bestIdx = idx
				bestLen = len(sep)
			}
		}
		if bestIdx < 0 {
			parts = append(parts, strings.TrimSpace(name[offset:]))
			break
		}
		parts = append(parts, strings.TrimSpace(name[offset:offset+bestIdx]))
		offset += bestIdx + bestLen
	}

	if len(parts) <= 1 {
		return nil
	}
	return parts
}

func unknownCharacterDiag(r token.Range, name string) protocol.Diagnostic {
	return protocol.Diagnostic{
		Range:    toLSPRange(r),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeUnknownCharacter,
		Source:   "downstage",
		Message:  "unknown character: " + name + " (add to Dramatis Personae)",
		Data:     map[string]string{"character": name},
	}
}

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
		if collectiveCues[name] {
			continue
		}

		if parts := splitConjunctionCue(ref.dialogue.Character); parts != nil {
			for _, part := range parts {
				up := strings.ToUpper(part)
				if up == "" || collectiveCues[up] {
					continue
				}
				if _, ok := index.knownCharacters[up]; ok {
					continue
				}
				diags = append(diags, unknownCharacterDiag(ref.dialogue.NameRange(), part))
			}
			continue
		}

		diags = append(diags, unknownCharacterDiag(ref.dialogue.NameRange(), ref.dialogue.Character))
	}
	return diags
}

func checkUnnumberedSections(index *documentIndex) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	for actNumber, act := range index.acts {
		if d := unnumberedActDiagnostic(act, actNumber+1); d != nil {
			diags = append(diags, *d)
			continue
		}
		if d := misnumberedActDiagnostic(act, actNumber+1); d != nil {
			diags = append(diags, *d)
		}
	}

	for _, scene := range index.scenes {
		number := index.sceneNumbers[scene]
		if d := unnumberedSceneDiagnostic(scene, number); d != nil {
			diags = append(diags, *d)
			continue
		}
		if d := misnumberedSceneDiagnostic(scene, number); d != nil {
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

func misnumberedActDiagnostic(section *ast.Section, expected int) *protocol.Diagnostic {
	actual, ok := parseRomanNumeral(section.Number)
	if ok && actual == expected {
		return nil
	}

	replacement := formatSectionHeading(section, romanNumeral(expected))
	return &protocol.Diagnostic{
		Range:    toLSPRange(section.HeadingRange()),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeMisnumberedAct,
		Source:   "downstage",
		Message:  fmt.Sprintf("act heading should be ACT %s in document order", romanNumeral(expected)),
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

func misnumberedSceneDiagnostic(section *ast.Section, expected int) *protocol.Diagnostic {
	actual, ok := parseSceneNumber(section.Number)
	if ok && actual == expected {
		return nil
	}

	replacement := formatSectionHeading(section, strconv.Itoa(expected))
	return &protocol.Diagnostic{
		Range:    toLSPRange(section.HeadingRange()),
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     diagnosticCodeMisnumberedScene,
		Source:   "downstage",
		Message:  fmt.Sprintf("scene heading should be SCENE %d in sequence", expected),
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

func parseSceneNumber(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}

	number, err := strconv.Atoi(raw)
	if err != nil || number <= 0 {
		return 0, false
	}

	return number, true
}

func parseRomanNumeral(raw string) (int, bool) {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if raw == "" {
		return 0, false
	}

	values := map[byte]int{
		'I': 1,
		'V': 5,
		'X': 10,
		'L': 50,
		'C': 100,
		'D': 500,
		'M': 1000,
	}

	total := 0
	for i := 0; i < len(raw); i++ {
		value, ok := values[raw[i]]
		if !ok {
			return 0, false
		}
		if i+1 < len(raw) {
			nextValue := values[raw[i+1]]
			if value < nextValue {
				total -= value
				continue
			}
		}
		total += value
	}

	if total <= 0 || romanNumeral(total) != raw {
		return 0, false
	}

	return total, true
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
