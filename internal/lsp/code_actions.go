package lsp

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"go.lsp.dev/protocol"
)

func computeCodeActions(
	doc *ast.Document,
	content string,
	uri protocol.DocumentURI,
	diagnostics []protocol.Diagnostic,
) []protocol.CodeAction {
	if doc == nil || len(diagnostics) == 0 {
		return []protocol.CodeAction{}
	}

	actions := make([]protocol.CodeAction, 0, len(diagnostics))
	seenCharacters := make(map[string]struct{})

	var (
		dp      = ast.FindDramatisPersonae(doc.Body)
		edit    protocol.TextEdit
		hasEdit bool
	)
	if dp != nil {
		edit, hasEdit = dramatisPersonaeInsertEdit(doc, content)
	}

	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case diagnosticCodeUnknownCharacter:
			if !hasEdit {
				continue
			}

			character := diagnosticCharacterName(diagnostic)
			if character == "" {
				continue
			}

			key := strings.ToUpper(character)
			if _, ok := seenCharacters[key]; ok {
				continue
			}
			seenCharacters[key] = struct{}{}

			textEdit := edit
			prefix := strings.TrimSuffix(edit.NewText, "\n")
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
		case diagnosticCodeUnnumberedAct, diagnosticCodeUnnumberedScene:
			replacement := diagnosticReplacement(diagnostic)
			if replacement == "" {
				continue
			}

			actions = append(actions, protocol.CodeAction{
				Title:       fmt.Sprintf("Number heading as %s", replacement),
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {{
							Range:   diagnostic.Range,
							NewText: replacement,
						}},
					},
				},
			})
		}
	}

	return actions
}

func diagnosticCharacterName(diagnostic protocol.Diagnostic) string {
	return diagnosticStringData(diagnostic, "character")
}

func diagnosticReplacement(diagnostic protocol.Diagnostic) string {
	return diagnosticStringData(diagnostic, "replacement")
}

func diagnosticStringData(diagnostic protocol.Diagnostic, key string) string {
	switch data := diagnostic.Data.(type) {
	case map[string]interface{}:
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

func dramatisPersonaeInsertEdit(doc *ast.Document, content string) (protocol.TextEdit, bool) {
	dp := ast.FindDramatisPersonae(doc.Body)
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
