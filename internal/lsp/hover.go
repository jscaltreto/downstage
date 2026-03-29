package lsp

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

// computeHover returns hover information when the cursor is on a character name in dialogue.
func computeHover(doc *ast.Document, _ []*parser.ParseError, pos protocol.Position) *protocol.Hover {
	if doc == nil {
		return nil
	}

	// Find the character name at the hover position.
	charName := findCharacterAtPosition(doc, pos)
	if charName == "" {
		return nil
	}

	// Look up the character in dramatis personae.
	dp := ast.FindDramatisPersonae(doc.Body)
	if dp == nil {
		return nil
	}

	ch, group := findCharacterByName(dp, charName)
	if ch == nil {
		return nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: formatCharacterHover(ch, group),
		},
	}
}

// findCharacterAtPosition searches body nodes for a dialogue or song node
// whose character name spans the given cursor position.
func findCharacterAtPosition(doc *ast.Document, pos protocol.Position) string {
	for _, n := range doc.Body {
		if name := findCharNameInNode(n, pos); name != "" {
			return name
		}
	}
	return ""
}

func findCharNameInNode(n ast.Node, pos protocol.Position) string {
	line := int(pos.Line)

	switch v := n.(type) {
	case *ast.DualDialogue:
		if name := findCharNameInNode(v.Left, pos); name != "" {
			return name
		}
		return findCharNameInNode(v.Right, pos)
	case *ast.Dialogue:
		r := v.NameRange()
		if r.Start.Line == line && int(pos.Character) >= r.Start.Column && int(pos.Character) < r.End.Column {
			return v.Character
		}
	case *ast.Song:
		// Check dialogue inside song content
		for _, child := range v.Content {
			if name := findCharNameInNode(child, pos); name != "" {
				return name
			}
		}
	case *ast.Section:
		for _, child := range v.Children {
			if name := findCharNameInNode(child, pos); name != "" {
				return name
			}
		}
	}
	return ""
}

// findCharacterByName looks up a character in the dramatis personae by name or alias.
// Returns the character and the group name (empty string if ungrouped).
func findCharacterByName(dp *ast.Section, name string) (*ast.Character, string) {
	upper := strings.ToUpper(name)
	for i, ch := range dp.Characters {
		if strings.ToUpper(ch.Name) == upper {
			return &dp.Characters[i], ""
		}
		for _, alias := range ch.Aliases {
			if strings.ToUpper(alias) == upper {
				return &dp.Characters[i], ""
			}
		}
	}
	for gi, g := range dp.Groups {
		for ci, ch := range g.Characters {
			if strings.ToUpper(ch.Name) == upper {
				return &dp.Groups[gi].Characters[ci], g.Name
			}
			for _, alias := range ch.Aliases {
				if strings.ToUpper(alias) == upper {
					return &dp.Groups[gi].Characters[ci], g.Name
				}
			}
		}
	}
	return nil, ""
}

func formatCharacterHover(ch *ast.Character, group string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s**", ch.Name))

	if ch.Description != "" {
		sb.WriteString(fmt.Sprintf("\n\n%s", ch.Description))
	}

	if len(ch.Aliases) > 0 {
		sb.WriteString(fmt.Sprintf("\n\n*Aliases:* %s", strings.Join(ch.Aliases, ", ")))
	}

	if group != "" {
		sb.WriteString(fmt.Sprintf("\n\n*Group:* %s", group))
	}

	return sb.String()
}
