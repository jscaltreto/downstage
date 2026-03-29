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

	dp := ast.FindDramatisPersonae(doc.Body)
	if dp == nil {
		return []protocol.CodeAction{}
	}

	edit, ok := dramatisPersonaeInsertEdit(doc, content)
	if !ok {
		return []protocol.CodeAction{}
	}

	actions := make([]protocol.CodeAction, 0, len(diagnostics))
	seen := make(map[string]struct{})

	for _, diagnostic := range diagnostics {
		if diagnostic.Code != diagnosticCodeUnknownCharacter {
			continue
		}

		character := diagnosticCharacterName(diagnostic)
		if character == "" {
			continue
		}

		key := strings.ToUpper(character)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

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
	}

	return actions
}

func diagnosticCharacterName(diagnostic protocol.Diagnostic) string {
	switch data := diagnostic.Data.(type) {
	case map[string]interface{}:
		raw, ok := data["character"]
		if !ok {
			return ""
		}

		name, ok := raw.(string)
		if !ok {
			return ""
		}

		return strings.TrimSpace(name)
	case map[string]string:
		return strings.TrimSpace(data["character"])
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
