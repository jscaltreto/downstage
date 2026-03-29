package lsp

import (
	"fmt"
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

	names := completionCandidates(doc, content, int(pos.Line))
	if len(names) == 0 {
		return emptyCompletionList()
	}

	items := make([]protocol.CompletionItem, 0, len(names))
	for i, name := range names {
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
			SortText:   completionSortText(i, insertText),
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

func completionSortText(index int, label string) string {
	return fmt.Sprintf("%04d:%s", index, strings.ToUpper(label))
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

func completionCandidates(doc *ast.Document, content string, line int) []string {
	if names, ok := sceneCompletionCandidates(doc, content, line); ok {
		return names
	}
	if !isCharacterCueLine(doc, line) {
		return nil
	}
	return documentCharacterNames(doc)
}

func sceneCompletionCandidates(doc *ast.Document, content string, line int) ([]string, bool) {
	if !followsBlankLine(content, line) {
		return nil, false
	}

	scene := findSceneForLine(doc.Body, line)
	if scene == nil {
		return nil, false
	}

	speakers := sceneSpeakersBeforeLine(scene, line)
	ranked := rankRecentSpeakers(speakers)
	return appendRemainingDPNames(doc, ranked), true
}

func documentCharacterNames(doc *ast.Document) []string {
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

	return names
}

func followsBlankLine(content string, line int) bool {
	if line <= 0 {
		return false
	}

	prevLine, ok := lineAt(content, line-1)
	if !ok {
		return false
	}

	return strings.TrimSpace(prevLine) == ""
}

func findSceneForLine(nodes []ast.Node, line int) *ast.Section {
	scenes := collectScenes(nodes)
	var current *ast.Section
	for _, scene := range scenes {
		if scene.Range.Start.Line <= line {
			current = scene
			continue
		}
		break
	}
	return current
}

func collectScenes(nodes []ast.Node) []*ast.Section {
	var scenes []*ast.Section
	for _, node := range nodes {
		scenes = append(scenes, collectScenesInNode(node)...)
	}
	return scenes
}

func collectScenesInNode(node ast.Node) []*ast.Section {
	var scenes []*ast.Section

	switch v := node.(type) {
	case *ast.Section:
		if v.Kind == ast.SectionScene {
			scenes = append(scenes, v)
		}
		for _, child := range v.Children {
			scenes = append(scenes, collectScenesInNode(child)...)
		}
	case *ast.Song:
		for _, child := range v.Content {
			scenes = append(scenes, collectScenesInNode(child)...)
		}
	}

	return scenes
}

func sceneSpeakersBeforeLine(scene *ast.Section, line int) []string {
	var speakers []string

	var walkNode func(ast.Node)
	walkNode = func(node ast.Node) {
		switch v := node.(type) {
		case *ast.Dialogue:
			if len(v.Lines) > 0 && v.NameRange().Start.Line < line {
				speakers = append(speakers, v.Character)
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

	for _, child := range scene.Children {
		walkNode(child)
	}

	return speakers
}

func rankRecentSpeakers(speakers []string) []string {
	if len(speakers) == 0 {
		return nil
	}

	lastSpeaker := strings.TrimSpace(speakers[len(speakers)-1])
	if lastSpeaker == "" {
		return nil
	}

	seen := map[string]struct{}{
		strings.ToUpper(lastSpeaker): {},
	}
	ranked := make([]string, 0, len(speakers))

	for i := len(speakers) - 2; i >= 0; i-- {
		name := strings.TrimSpace(speakers[i])
		if name == "" {
			continue
		}
		key := strings.ToUpper(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ranked = append(ranked, name)
	}

	return append(ranked, lastSpeaker)
}

func appendRemainingDPNames(doc *ast.Document, names []string) []string {
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		seen[strings.ToUpper(strings.TrimSpace(name))] = struct{}{}
	}

	if dp := ast.FindDramatisPersonae(doc.Body); dp != nil {
		for _, ch := range dp.AllCharacters() {
			name := strings.TrimSpace(ch.Name)
			if name == "" {
				continue
			}
			key := strings.ToUpper(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, name)
		}
	}

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
