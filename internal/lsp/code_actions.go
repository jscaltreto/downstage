package lsp

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/migrate"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

// ComputeCodeActions returns the quick-fix actions surfaced by the Downstage LSP.
func ComputeCodeActions(
	doc *ast.Document,
	content string,
	uri protocol.DocumentURI,
	diagnostics []protocol.Diagnostic,
	allDiagnostics []protocol.Diagnostic,
) []protocol.CodeAction {
	return computeCodeActions(doc, content, uri, diagnostics, allDiagnostics)
}

func computeCodeActions(
	doc *ast.Document,
	content string,
	uri protocol.DocumentURI,
	diagnostics []protocol.Diagnostic,
	allDiagnostics []protocol.Diagnostic,
) []protocol.CodeAction {
	if doc == nil || len(diagnostics) == 0 {
		return []protocol.CodeAction{}
	}
	index := newDocumentIndex(doc)

	actions := make([]protocol.CodeAction, 0, len(diagnostics))
	seenCharacters := make(map[string]struct{})
	allActEdits := make([]protocol.TextEdit, 0)
	allSceneEdits := make([]protocol.TextEdit, 0)
	seenActEdits := make(map[string]struct{})
	seenSceneEdits := make(map[string]struct{})
	var hasMisnumberedAct, hasMisnumberedScene bool

	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case parser.ErrCodeDPUnicodeDash:
			edit := replaceUnicodeDashEdit(content, diagnostic.Range)
			if edit == nil {
				continue
			}
			actions = append(actions, protocol.CodeAction{
				Title:       "Replace Unicode dash with ASCII ` - `",
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {*edit},
					},
				},
			})
		case parser.ErrCodeDPStandaloneAlias:
			edit := inlineStandaloneAliasEdit(content, diagnostic.Range)
			if edit == nil {
				continue
			}
			actions = append(actions, protocol.CodeAction{
				Title:       "Rewrite alias as inline NAME/ALIAS",
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {*edit},
					},
				},
			})
		case diagnosticCodeV1Document:
			upgraded, changed := migrate.UpgradeV1ToV2(content)
			if !changed {
				continue
			}

			actions = append(actions, protocol.CodeAction{
				Title:       "Update script to V2",
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {fullDocumentEdit(content, upgraded)},
					},
				},
			})
		case diagnosticCodeUnknownCharacter:
			character := diagnosticCharacterName(diagnostic)
			if character == "" {
				continue
			}

			if forceEdit, ok := forceCharacterCueEdit(content, diagnostic.Range); ok {
				actions = append(actions, protocol.CodeAction{
					Title:       "Exclude cue from check",
					Kind:        protocol.QuickFix,
					Diagnostics: []protocol.Diagnostic{diagnostic},
					Edit: &protocol.WorkspaceEdit{
						Changes: map[protocol.DocumentURI][]protocol.TextEdit{
							uri: {forceEdit},
						},
					},
				})
			}

			textEdit, hasEdit := dramatisPersonaeInsertEdit(doc, index, content, int(diagnostic.Range.Start.Line))
			if !hasEdit {
				continue
			}

			key := strings.ToUpper(character)
			if _, ok := seenCharacters[key]; ok {
				continue
			}
			seenCharacters[key] = struct{}{}

			prefix := strings.TrimSuffix(textEdit.NewText, "\n")
			textEdit.NewText = prefix + character + "\n"

			actions = append(actions, protocol.CodeAction{
				Title:       fmt.Sprintf("Add %s to Dramatis Personae", character),
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {textEdit},
					},
				},
			})
		case diagnosticCodeDPDuplicateCharacterName:
			edit := deleteDuplicateDPEntryEdit(content, diagnostic.Range)
			if edit == nil {
				continue
			}
			actions = append(actions, protocol.CodeAction{
				Title:       "Delete duplicate Dramatis Personae entry",
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {*edit},
					},
				},
			})
		case diagnosticCodeMissingDramatisPersonae:
			edit := insertMissingDramatisPersonaeEdit(doc, content, index)
			if edit == nil {
				continue
			}
			actions = append(actions, protocol.CodeAction{
				Title:       "Add Dramatis Personae section",
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {*edit},
					},
				},
			})
		case diagnosticCodeUnnumberedAct, diagnosticCodeUnnumberedScene,
			diagnosticCodeMisnumberedAct, diagnosticCodeMisnumberedScene:
			textEdit := numberingEdit(diagnostic)
			if textEdit == nil {
				continue
			}

			switch diagnostic.Code {
			case diagnosticCodeUnnumberedAct, diagnosticCodeMisnumberedAct:
				if diagnostic.Code == diagnosticCodeMisnumberedAct {
					hasMisnumberedAct = true
				}
				if registerEdit(seenActEdits, *textEdit) {
					allActEdits = append(allActEdits, *textEdit)
				}
			case diagnosticCodeUnnumberedScene, diagnosticCodeMisnumberedScene:
				if diagnostic.Code == diagnosticCodeMisnumberedScene {
					hasMisnumberedScene = true
				}
				if registerEdit(seenSceneEdits, *textEdit) {
					allSceneEdits = append(allSceneEdits, *textEdit)
				}
			}

			actions = append(actions, protocol.CodeAction{
				Title:       numberingActionTitle(diagnostic.Code, textEdit.NewText),
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {*textEdit},
					},
				},
			})
		}
	}

	// Collect bulk edits from remaining diagnostics not in the context set.
	for _, diagnostic := range allDiagnostics {
		switch diagnostic.Code {
		case diagnosticCodeUnnumberedAct, diagnosticCodeMisnumberedAct:
			if diagnostic.Code == diagnosticCodeMisnumberedAct {
				hasMisnumberedAct = true
			}
			if edit := numberingEdit(diagnostic); edit != nil {
				if registerEdit(seenActEdits, *edit) {
					allActEdits = append(allActEdits, *edit)
				}
			}
		case diagnosticCodeUnnumberedScene, diagnosticCodeMisnumberedScene:
			if diagnostic.Code == diagnosticCodeMisnumberedScene {
				hasMisnumberedScene = true
			}
			if edit := numberingEdit(diagnostic); edit != nil {
				if registerEdit(seenSceneEdits, *edit) {
					allSceneEdits = append(allSceneEdits, *edit)
				}
			}
		}
	}

	if len(allActEdits) > 1 {
		title := "Number all acts in document"
		if hasMisnumberedAct {
			title = "Normalize all acts in document"
		}
		actions = append(actions, protocol.CodeAction{
			Title: title,
			Kind:  protocol.QuickFix,
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					uri: allActEdits,
				},
			},
		})
	}

	if len(allSceneEdits) > 1 {
		title := "Number all scenes in document"
		if hasMisnumberedScene {
			title = "Normalize all scenes in document"
		}
		actions = append(actions, protocol.CodeAction{
			Title: title,
			Kind:  protocol.QuickFix,
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					uri: allSceneEdits,
				},
			},
		})
	}

	return actions
}

func numberingActionTitle(code interface{}, replacement string) string {
	switch code {
	case diagnosticCodeMisnumberedAct, diagnosticCodeMisnumberedScene:
		return fmt.Sprintf("Renumber heading as %s", replacement)
	default:
		return fmt.Sprintf("Number heading as %s", replacement)
	}
}

func registerEdit(seen map[string]struct{}, edit protocol.TextEdit) bool {
	key := fmt.Sprintf(
		"%d:%d:%d:%d:%s",
		edit.Range.Start.Line,
		edit.Range.Start.Character,
		edit.Range.End.Line,
		edit.Range.End.Character,
		edit.NewText,
	)
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = struct{}{}
	return true
}

func numberingEdit(diagnostic protocol.Diagnostic) *protocol.TextEdit {
	replacement := diagnosticReplacement(diagnostic)
	if replacement == "" {
		return nil
	}
	return &protocol.TextEdit{
		Range:   diagnostic.Range,
		NewText: replacement,
	}
}

func fullDocumentEdit(content string, replacement string) protocol.TextEdit {
	lines := strings.Split(content, "\n")
	endLine := len(lines) - 1
	endChar := 0
	if endLine >= 0 {
		endChar = len(lines[endLine])
	}
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End: protocol.Position{
				Line:      uint32(maxInt(endLine, 0)),
				Character: uint32(maxInt(endChar, 0)),
			},
		},
		NewText: replacement,
	}
}

func diagnosticCharacterName(diagnostic protocol.Diagnostic) string {
	return diagnosticStringData(diagnostic, "character")
}

func diagnosticReplacement(diagnostic protocol.Diagnostic) string {
	return diagnosticStringData(diagnostic, "replacement")
}

func diagnosticStringData(diagnostic protocol.Diagnostic, key string) string {
	switch data := diagnostic.Data.(type) {
	case map[string]any:
		raw, ok := data[key]
		if !ok {
			return ""
		}

		name, ok := raw.(string)
		if !ok {
			return ""
		}

		return strings.TrimSpace(name)
	case map[string]string:
		return strings.TrimSpace(data[key])
	default:
		return ""
	}
}

func dramatisPersonaeInsertEdit(doc *ast.Document, index *documentIndex, content string, line int) (protocol.TextEdit, bool) {
	if index == nil {
		index = newDocumentIndex(doc)
	}
	dp := index.characterScopeForLine(doc, line).dp
	if dp == nil {
		return protocol.TextEdit{}, false
	}

	insertAt := dp.Range.End
	lines := strings.Split(content, "\n")
	if insertAt.Line < 0 || insertAt.Line >= len(lines) {
		return protocol.TextEdit{}, false
	}

	newText := ""
	if dpHasNoEntries(dp) {
		newText = "\n"
	}

	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(insertAt.Line),
				Character: uint32(insertAt.Column),
			},
			End: protocol.Position{
				Line:      uint32(insertAt.Line),
				Character: uint32(insertAt.Column),
			},
		},
		NewText: newText + "\n",
	}, true
}

func dpHasNoEntries(dp *ast.Section) bool {
	if dp == nil {
		return true
	}
	return len(dp.Characters) == 0 && len(dp.Groups) == 0
}

// replaceUnicodeDashEdit returns a TextEdit that rewrites em-dashes and
// en-dashes around the DP separator with the ASCII ` - ` form expected
// by SPEC §5.
func replaceUnicodeDashEdit(content string, r protocol.Range) *protocol.TextEdit {
	line, ok := lineAt(content, int(r.Start.Line))
	if !ok {
		return nil
	}
	replaced := strings.ReplaceAll(line, " \u2014 ", " - ")
	replaced = strings.ReplaceAll(replaced, " \u2013 ", " - ")
	if replaced == line {
		return nil
	}
	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: r.Start.Line, Character: 0},
			End:   protocol.Position{Line: r.Start.Line, Character: uint32(utf16Len(line))},
		},
		NewText: replaced,
	}
}

// forceCharacterCueEdit builds a TextEdit that prepends `@` to the cue on
// the diagnostic line, promoting it to a forced character that opts out of
// the "unknown character" check.
//
// The `@` is inserted at the first non-whitespace column so that indented
// cues (e.g. `  HAMLET`) become `  @HAMLET`, not `@  HAMLET` — the lexer
// strips leading whitespace when computing the character name, but only
// after `@` has been consumed, so misplacement would corrupt the parsed
// character name.
func forceCharacterCueEdit(content string, r protocol.Range) (protocol.TextEdit, bool) {
	line, ok := lineAt(content, int(r.Start.Line))
	if !ok {
		return protocol.TextEdit{}, false
	}
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return protocol.TextEdit{}, false
	}
	// Already forced? Nothing to do. The diagnostic shouldn't fire in that
	// case, but guard so a stale diagnostic can't produce `@@NAME`.
	if strings.HasPrefix(trimmed, "@") {
		return protocol.TextEdit{}, false
	}
	// Leading whitespace is always ASCII space/tab, so byte length equals
	// both the column count and the UTF-16 code-unit count.
	indent := uint32(len(line) - len(trimmed))
	insertAt := protocol.Position{Line: r.Start.Line, Character: indent}
	return protocol.TextEdit{
		Range:   protocol.Range{Start: insertAt, End: insertAt},
		NewText: "@",
	}, true
}

// inlineStandaloneAliasEdit rewrites a `[NAME/ALIAS]` line as the
// bracketless `NAME/ALIAS` form that the V2 parser accepts as a
// character entry.
func inlineStandaloneAliasEdit(content string, r protocol.Range) *protocol.TextEdit {
	line, ok := lineAt(content, int(r.Start.Line))
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return nil
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
	if inner == "" {
		return nil
	}
	// Preserve the line's leading whitespace so the edit is minimally invasive.
	lead := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: r.Start.Line, Character: 0},
			End:   protocol.Position{Line: r.Start.Line, Character: uint32(utf16Len(line))},
		},
		NewText: lead + inner,
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// deleteDuplicateDPEntryEdit removes an entire duplicate Dramatis Personae
// row. The diagnostic range covers the entry's character Range, which is
// typically a single line; the edit consumes the trailing newline so
// surrounding blank-line conventions survive.
func deleteDuplicateDPEntryEdit(content string, r protocol.Range) *protocol.TextEdit {
	startLine := int(r.Start.Line)
	endLine := int(r.End.Line)
	if endLine < startLine {
		return nil
	}
	if _, ok := lineAt(content, startLine); !ok {
		return nil
	}
	// Prefer consuming the trailing newline so the row vanishes cleanly.
	// If we're at EOF (no trailing newline), fall back to consuming the
	// preceding newline so we don't leave a blank row behind.
	if _, ok := lineAt(content, endLine+1); ok {
		return &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(startLine), Character: 0},
				End:   protocol.Position{Line: uint32(endLine + 1), Character: 0},
			},
			NewText: "",
		}
	}
	if startLine == 0 {
		// Only row in the file — just clear it.
		line, _ := lineAt(content, startLine)
		return &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(startLine), Character: 0},
				End:   protocol.Position{Line: uint32(endLine), Character: uint32(utf16Len(line))},
			},
			NewText: "",
		}
	}
	prev, _ := lineAt(content, startLine-1)
	endLineText, _ := lineAt(content, endLine)
	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(startLine - 1), Character: uint32(utf16Len(prev))},
			End:   protocol.Position{Line: uint32(endLine), Character: uint32(utf16Len(endLineText))},
		},
		NewText: "",
	}
}

// insertMissingDramatisPersonaeEdit produces a workspace edit that inserts a
// fresh Dramatis Personae section seeded with every character observed as a
// cue in the document. Names are uppercased, deduplicated, and ordered by
// first appearance.
func insertMissingDramatisPersonaeEdit(doc *ast.Document, content string, index *documentIndex) *protocol.TextEdit {
	if doc == nil {
		return nil
	}
	if index == nil {
		index = newDocumentIndex(doc)
	}

	names := observedCueNames(index)
	body := "## Dramatis Personae\n\n"
	if len(names) == 0 {
		body += "\n"
	} else {
		for _, name := range names {
			body += name + "\n"
		}
		body += "\n"
	}

	// Insert after the play heading and any attached metadata block.
	if len(index.topLevelSections) > 0 {
		play := index.topLevelSections[0]
		line := insertAfterPlayHeader(play, content)
		return &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(line), Character: 0},
				End:   protocol.Position{Line: uint32(line), Character: 0},
			},
			NewText: body,
		}
	}

	startLine := 0
	if doc.TitlePage != nil {
		startLine = doc.TitlePage.Range.End.Line + 1
		startLine = skipBlankLines(content, startLine)
	}
	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(startLine), Character: 0},
			End:   protocol.Position{Line: uint32(startLine), Character: 0},
		},
		NewText: body,
	}
}

// insertAfterPlayHeader returns the insertion line under a play heading.
func insertAfterPlayHeader(play *ast.Section, content string) int {
	line := play.HeadingRange().End.Line + 1
	if play.Metadata != nil {
		line = play.Metadata.Range.End.Line + 1
	}
	return skipBlankLines(content, line)
}

func skipBlankLines(content string, start int) int {
	line := start
	for {
		raw, ok := lineAt(content, line)
		if !ok {
			return line
		}
		if strings.TrimSpace(raw) != "" {
			return line
		}
		line++
	}
}

// observedCueNames returns unique cue character names in first-appearance
// order. Forced and conjunction-split cues are included; empty cues are
// skipped.
func observedCueNames(index *documentIndex) []string {
	seen := make(map[string]struct{})
	var names []string
	add := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		key := strings.ToUpper(trimmed)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		names = append(names, trimmed)
	}
	for _, ref := range index.dialogues {
		if ref.dialogue == nil {
			continue
		}
		if parts := splitConjunctionCue(ref.dialogue.Character); parts != nil {
			for _, p := range parts {
				if !collectiveCues[strings.ToUpper(strings.TrimSpace(p))] {
					add(p)
				}
			}
			continue
		}
		if collectiveCues[strings.ToUpper(strings.TrimSpace(ref.dialogue.Character))] {
			continue
		}
		add(ref.dialogue.Character)
	}
	return names
}
