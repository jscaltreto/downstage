package lsp

import (
	"slices"
	"strings"
	"unicode"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func computeCompletion(doc *ast.Document, _ []*parser.ParseError, content string, pos protocol.Position) *protocol.CompletionList {
	if doc == nil {
		return emptyCompletionList()
	}

	ctx, ok := completionContextAt(content, pos)
	if !ok {
		return emptyCompletionList()
	}
	if !isCharacterCueLine(doc, int(pos.Line)) {
		return emptyCompletionList()
	}

	names := collectCharacterNames(doc)
	if len(names) == 0 {
		return emptyCompletionList()
	}

	items := make([]protocol.CompletionItem, 0, len(names))
	for _, name := range names {
		if !strings.HasPrefix(strings.ToUpper(name), ctx.filterPrefix) {
			continue
		}

		insertText := name
		if ctx.hasForcedPrefix {
			insertText = "@" + name
		}

		items = append(items, protocol.CompletionItem{
			Label:      insertText,
			Kind:       protocol.CompletionItemKindVariable,
			Detail:     "Character cue",
			FilterText: insertText,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      pos.Line,
						Character: 0,
					},
					End: pos,
				},
				NewText: insertText,
			},
		})
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

type completionContext struct {
	filterPrefix    string
	hasForcedPrefix bool
}

func completionContextAt(content string, pos protocol.Position) (completionContext, bool) {
	line, ok := lineAt(content, int(pos.Line))
	if !ok {
		return completionContext{}, false
	}

	cursor := int(pos.Character)
	if cursor < 0 || cursor > utf16Len(line) {
		return completionContext{}, false
	}

	if strings.TrimSpace(utf16Prefix(line, cursor)) == "" && strings.TrimSpace(line) == "" {
		return completionContext{
			filterPrefix: "",
		}, true
	}

	if strings.TrimSpace(utf16Suffix(line, cursor)) != "" {
		return completionContext{}, false
	}

	prefix := utf16Prefix(line, cursor)
	if strings.TrimLeft(prefix, " \t") != prefix {
		return completionContext{}, false
	}

	forced := strings.HasPrefix(prefix, "@")
	if forced {
		prefix = strings.TrimPrefix(prefix, "@")
	}

	if prefix == "" {
		return completionContext{
			filterPrefix:    "",
			hasForcedPrefix: forced,
		}, true
	}

	for _, r := range prefix {
		if unicode.IsUpper(r) || unicode.IsDigit(r) || r == ' ' || r == '.' || r == ',' || r == '-' || r == '\'' || r == '/' {
			continue
		}
		return completionContext{}, false
	}

	return completionContext{
		filterPrefix:    strings.ToUpper(prefix),
		hasForcedPrefix: forced,
	}, true
}

func collectCharacterNames(doc *ast.Document) []string {
	seen := make(map[string]struct{})
	names := make([]string, 0)

	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToUpper(name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		names = append(names, name)
	}

	if dp := ast.FindDramatisPersonae(doc.Body); dp != nil {
		for _, ch := range dp.AllCharacters() {
			add(ch.Name)
			for _, alias := range ch.Aliases {
				add(alias)
			}
		}
	}

	var walkNode func(ast.Node)
	walkNode = func(n ast.Node) {
		switch v := n.(type) {
		case *ast.Dialogue:
			if len(v.Lines) > 0 {
				add(v.Character)
			}
		case *ast.DualDialogue:
			walkNode(v.Left)
			walkNode(v.Right)
		case *ast.Section:
			for _, child := range v.Children {
				walkNode(child)
			}
		case *ast.Song:
			for _, child := range v.Content {
				walkNode(child)
			}
		}
	}

	for _, node := range doc.Body {
		walkNode(node)
	}

	slices.SortFunc(names, func(a, b string) int {
		return strings.Compare(strings.ToUpper(a), strings.ToUpper(b))
	})

	return names
}

func isCharacterCueLine(doc *ast.Document, line int) bool {
	var found bool

	var walkNode func(ast.Node)
	walkNode = func(n ast.Node) {
		if found {
			return
		}

		switch v := n.(type) {
		case *ast.Dialogue:
			if v.NameRange().Start.Line == line {
				found = true
			}
		case *ast.DualDialogue:
			walkNode(v.Left)
			walkNode(v.Right)
		case *ast.Section:
			for _, child := range v.Children {
				walkNode(child)
			}
		case *ast.Song:
			for _, child := range v.Content {
				walkNode(child)
			}
		}
	}

	for _, node := range doc.Body {
		walkNode(node)
		if found {
			return true
		}
	}

	return false
}

func lineAt(content string, lineNum int) (string, bool) {
	lines := strings.Split(content, "\n")
	if lineNum < 0 || lineNum >= len(lines) {
		return "", false
	}
	return lines[lineNum], true
}

func utf16Prefix(s string, count int) string {
	if count <= 0 {
		return ""
	}

	var b strings.Builder
	used := 0
	for _, r := range s {
		width := utf16RuneLen(r)
		if used+width > count {
			break
		}
		b.WriteRune(r)
		used += width
	}
	return b.String()
}

func utf16Suffix(s string, start int) string {
	prefix := utf16Prefix(s, start)
	return s[len(prefix):]
}

func utf16RuneLen(r rune) int {
	if r <= 0xFFFF {
		return 1
	}
	return 2
}

func emptyCompletionList() *protocol.CompletionList {
	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        []protocol.CompletionItem{},
	}
}
