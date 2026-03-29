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

	items := completionItems(doc, content, int(pos.Line), pos, ctx)
	if len(items) == 0 {
		return emptyCompletionList()
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
	kind            completionKind
	filterPrefix    string
	hasForcedPrefix bool
	headingLevel    int
}

type completionKind int

const (
	completionKindCharacter completionKind = iota
	completionKindHeading
)

func completionContextAt(content string, pos protocol.Position) (completionContext, bool) {
	line, ok := lineAt(content, int(pos.Line))
	if !ok {
		return completionContext{}, false
	}

	cursor := int(pos.Character)
	if cursor < 0 || cursor > utf16Len(line) {
		return completionContext{}, false
	}

	if ctx, ok := headingCompletionContext(line, cursor); ok {
		return ctx, true
	}

	if strings.TrimSpace(utf16Prefix(line, cursor)) == "" && strings.TrimSpace(line) == "" {
		return completionContext{
			kind:         completionKindCharacter,
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
			kind:            completionKindCharacter,
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
		kind:            completionKindCharacter,
		filterPrefix:    strings.ToUpper(prefix),
		hasForcedPrefix: forced,
	}, true
}

func completionItems(doc *ast.Document, content string, line int, pos protocol.Position, ctx completionContext) []protocol.CompletionItem {
	switch ctx.kind {
	case completionKindHeading:
		return headingCompletionItems(doc, line, pos, ctx)
	case completionKindCharacter:
		return characterCompletionItems(doc, content, line, pos, ctx)
	default:
		return nil
	}
}

func characterCompletionItems(doc *ast.Document, content string, line int, pos protocol.Position, ctx completionContext) []protocol.CompletionItem {
	names := characterCompletionCandidates(doc, content, line)
	if len(names) == 0 {
		return nil
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

	return items
}

func headingCompletionItems(doc *ast.Document, line int, pos protocol.Position, ctx completionContext) []protocol.CompletionItem {
	labels := headingCompletionCandidates(doc, line, ctx.headingLevel)
	if len(labels) == 0 {
		return nil
	}

	items := make([]protocol.CompletionItem, 0, len(labels))
	for i, label := range labels {
		if !strings.HasPrefix(strings.ToUpper(label), ctx.filterPrefix) {
			continue
		}

		insertText := strings.Repeat("#", ctx.headingLevel) + " " + label
		items = append(items, protocol.CompletionItem{
			Label:      insertText,
			Kind:       protocol.CompletionItemKindKeyword,
			Detail:     "Structural heading",
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

	return items
}

func headingCompletionContext(line string, cursor int) (completionContext, bool) {
	prefix := utf16Prefix(line, cursor)
	if strings.TrimSpace(utf16Suffix(line, cursor)) != "" {
		return completionContext{}, false
	}
	if strings.TrimLeft(prefix, " \t") != prefix {
		return completionContext{}, false
	}

	for level := 3; level >= 1; level-- {
		marker := strings.Repeat("#", level) + " "
		if !strings.HasPrefix(prefix, marker) {
			continue
		}

		filter := strings.TrimSpace(strings.TrimPrefix(prefix, marker))
		for _, r := range filter {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == ':' || r == '-' || r == '\'' || r == ',' {
				continue
			}
			return completionContext{}, false
		}

		return completionContext{
			kind:         completionKindHeading,
			filterPrefix: strings.ToUpper(filter),
			headingLevel: level,
		}, true
	}

	return completionContext{}, false
}

func characterCompletionCandidates(doc *ast.Document, content string, line int) []string {
	if names, ok := sceneCompletionCandidates(doc, content, line); ok {
		return names
	}
	if !isCharacterCueLine(doc, line) {
		return nil
	}
	return documentCharacterNames(doc)
}

func headingCompletionCandidates(doc *ast.Document, line, level int) []string {
	switch level {
	case 1:
		if ast.FindDramatisPersonae(doc.Body) != nil {
			return nil
		}
		return []string{"Dramatis Personae"}
	case 2:
		return []string{nextActHeading(doc, line)}
	case 3:
		return []string{nextSceneHeading(doc, line)}
	default:
		return nil
	}
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

func nextActHeading(doc *ast.Document, line int) string {
	count := 0
	for _, act := range collectSectionsOfKind(doc.Body, ast.SectionAct) {
		if act.Range.Start.Line < line {
			count++
		}
	}
	return "ACT " + romanNumeral(count+1)
}

func nextSceneHeading(doc *ast.Document, line int) string {
	if act := findActForLine(doc.Body, line); act != nil {
		count := 0
		for _, child := range act.Children {
			section, ok := child.(*ast.Section)
			if !ok || section.Kind != ast.SectionScene {
				continue
			}
			if section.Range.Start.Line < line {
				count++
			}
		}
		return fmt.Sprintf("SCENE %d", count+1)
	}

	count := 0
	for _, scene := range collectSectionsOfKind(doc.Body, ast.SectionScene) {
		if scene.Range.Start.Line < line {
			count++
		}
	}
	return fmt.Sprintf("SCENE %d", count+1)
}

func findActForLine(nodes []ast.Node, line int) *ast.Section {
	acts := collectSectionsOfKind(nodes, ast.SectionAct)
	var current *ast.Section
	for _, act := range acts {
		if act.Range.Start.Line <= line {
			current = act
			continue
		}
		break
	}
	return current
}

func collectSectionsOfKind(nodes []ast.Node, kind ast.SectionKind) []*ast.Section {
	var sections []*ast.Section
	for _, node := range nodes {
		sections = append(sections, collectSectionsOfKindInNode(node, kind)...)
	}
	return sections
}

func collectSectionsOfKindInNode(node ast.Node, kind ast.SectionKind) []*ast.Section {
	var sections []*ast.Section

	switch v := node.(type) {
	case *ast.Section:
		if v.Kind == kind {
			sections = append(sections, v)
		}
		for _, child := range v.Children {
			sections = append(sections, collectSectionsOfKindInNode(child, kind)...)
		}
	case *ast.Song:
		for _, child := range v.Content {
			sections = append(sections, collectSectionsOfKindInNode(child, kind)...)
		}
	}

	return sections
}

func romanNumeral(n int) string {
	if n <= 0 {
		return "I"
	}

	values := []struct {
		value   int
		numeral string
	}{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}

	var b strings.Builder
	for _, value := range values {
		for n >= value.value {
			b.WriteString(value.numeral)
			n -= value.value
		}
	}

	return b.String()
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
