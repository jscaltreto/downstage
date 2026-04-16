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
	diagnosticCodeUnknownCharacter            = "unknown-character"
	diagnosticCodeUnnumberedAct               = "unnumbered-act"
	diagnosticCodeUnnumberedScene             = "unnumbered-scene"
	diagnosticCodeMisnumberedAct              = "misnumbered-act"
	diagnosticCodeMisnumberedScene            = "misnumbered-scene"
	diagnosticCodeV1Document                  = "v1-document"
	diagnosticCodeMissingDramatisPersonae     = "missing-dramatis-personae"
	diagnosticCodeDPCharacterNoDialogue       = "dp-character-no-dialogue"
	diagnosticCodeDPDuplicateCharacterName    = "dp-duplicate-character-name"
	diagnosticCodeDPDuplicateAlias            = "dp-duplicate-alias"
	diagnosticCodeCueOrphaned                 = "cue-orphaned"
	diagnosticCodeCueConsecutiveSameCharacter = "cue-consecutive-same-character"
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
	parts := splitConjunctionCueWithOffsets(name)
	if len(parts) == 0 {
		return nil
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = p.Name
	}
	return out
}

// conjunctionPart describes one participant of a conjunction cue,
// alongside its byte offsets within the original cue string. Callers
// that need to edit a sub-name in place (rename) use the offsets to
// build a precise source range.
type conjunctionPart struct {
	Name  string
	Start int // inclusive byte offset of the trimmed name within the cue
	End   int // exclusive byte offset
}

// splitConjunctionCueWithOffsets returns participants of a conjunction
// cue together with their byte spans inside the original name string.
// Returns nil when the cue is a single name.
func splitConjunctionCueWithOffsets(name string) []conjunctionPart {
	upper := strings.ToUpper(name)

	type rawPart struct {
		start, end int // raw segment bounds, including any surrounding whitespace
	}
	var raws []rawPart
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
			raws = append(raws, rawPart{start: offset, end: len(name)})
			break
		}
		raws = append(raws, rawPart{start: offset, end: offset + bestIdx})
		offset += bestIdx + bestLen
	}

	if len(raws) <= 1 {
		return nil
	}

	parts := make([]conjunctionPart, 0, len(raws))
	for _, raw := range raws {
		segment := name[raw.start:raw.end]
		trimmedLeading := len(segment) - len(strings.TrimLeft(segment, " \t"))
		trimmed := strings.TrimSpace(segment)
		start := raw.start + trimmedLeading
		end := start + len(trimmed)
		parts = append(parts, conjunctionPart{Name: trimmed, Start: start, End: end})
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

// ComputeDiagnostics returns the same diagnostics surfaced by the Downstage LSP.
func ComputeDiagnostics(doc *ast.Document, errors []*parser.ParseError) []protocol.Diagnostic {
	return buildDiagnostics(doc, errors)
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
		diag := protocol.Diagnostic{
			Range:    toLSPRange(e.Range),
			Severity: protocol.DiagnosticSeverityError,
			Source:   "downstage",
			Message:  e.Message,
		}
		if e.Code != "" {
			diag.Code = e.Code
		}
		diags = append(diags, diag)
	}
	if diag := v1DocumentDiagnostic(errors); diag != nil {
		diags = append(diags, *diag)
	}

	// Add warnings for unknown character names.
	if doc != nil {
		diags = append(diags, checkUnnumberedSections(index)...)
		diags = append(diags, checkUnknownCharacters(index)...)
		diags = append(diags, checkMissingDramatisPersonae(doc, index)...)
		diags = append(diags, checkDPDuplicates(index)...)
		diags = append(diags, checkDPCharacterNoDialogue(index)...)
		diags = append(diags, checkOrphanedCues(index)...)
		diags = append(diags, checkConsecutiveSameCharacterCues(index)...)
	}

	if diags == nil {
		return []protocol.Diagnostic{}
	}

	return diags
}

func v1DocumentDiagnostic(errors []*parser.ParseError) *protocol.Diagnostic {
	var first *parser.ParseError
	for _, err := range errors {
		if !isV1ParseError(err) {
			continue
		}
		first = err
		break
	}
	if first == nil {
		return nil
	}

	return &protocol.Diagnostic{
		Range:    toLSPRange(first.Range),
		Severity: protocol.DiagnosticSeverityError,
		Code:     diagnosticCodeV1Document,
		Source:   "downstage",
		Message:  "this looks like a V1 Downstage document; update it to V2 to continue working",
	}
}

func isV1ParseError(err *parser.ParseError) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Message, "document-level metadata is a V1 pattern") ||
		strings.Contains(err.Message, "top-level Dramatis Personae is a V1 pattern")
}

// checkUnknownCharacters warns when dialogue references a character not in dramatis personae.
func checkUnknownCharacters(index *documentIndex) []protocol.Diagnostic {
	if !index.hasDramatisPersonae {
		return nil
	}

	var diags []protocol.Diagnostic
	for _, ref := range index.dialogues {
		scope := index.characterScopeForSection(ref.play)
		if scope.dp == nil {
			continue
		}

		// Forced cues (`@name`) opt out of DP membership checks. The `@`
		// marks the cue as intentional even when the character is not
		// listed in Dramatis Personae.
		if ref.dialogue.Forced {
			continue
		}

		name := strings.ToUpper(ref.dialogue.Character)
		if name == "" {
			continue
		}
		if _, ok := scope.known[name]; ok {
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
				if _, ok := scope.known[up]; ok {
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

	for _, act := range index.acts {
		actNumber := index.actNumbers[act]
		if d := unnumberedActDiagnostic(act, actNumber); d != nil {
			diags = append(diags, *d)
			continue
		}
		if d := misnumberedActDiagnostic(act, actNumber); d != nil {
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
