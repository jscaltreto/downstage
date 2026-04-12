package lsp

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/migrate"
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
