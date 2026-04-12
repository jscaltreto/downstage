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
	return computeCompletionWithIndex(doc, newDocumentIndex(doc), content, pos)
}

func computeCompletionWithIndex(doc *ast.Document, index *documentIndex, content string, pos protocol.Position) *protocol.CompletionList {
	if doc == nil {
		return emptyCompletionList()
	}
	if index == nil {
		index = newDocumentIndex(doc)
	}

	ctx, ok := completionContextAt(content, pos)
	if !ok {
		return emptyCompletionList()
	}

	items := completionItems(doc, index, content, int(pos.Line), pos, ctx)
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

func completionItems(doc *ast.Document, index *documentIndex, content string, line int, pos protocol.Position, ctx completionContext) []protocol.CompletionItem {
	switch ctx.kind {
	case completionKindHeading:
		return headingCompletionItems(doc, index, line, pos, ctx)
	case completionKindCharacter:
		return characterCompletionItems(doc, index, content, line, pos, ctx)
	default:
		return nil
	}
}

func characterCompletionItems(doc *ast.Document, index *documentIndex, content string, line int, pos protocol.Position, ctx completionContext) []protocol.CompletionItem {
	names := characterCompletionCandidates(doc, index, content, line)
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

func headingCompletionItems(doc *ast.Document, index *documentIndex, line int, pos protocol.Position, ctx completionContext) []protocol.CompletionItem {
	labels := headingCompletionCandidates(doc, index, line, ctx.headingLevel)
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

func characterCompletionCandidates(doc *ast.Document, index *documentIndex, content string, line int) []string {
	if names, ok := sceneCompletionCandidates(doc, index, content, line); ok {
		return names
	}
	if !index.isCharacterCueLine(line) {
		return nil
	}
	scope := index.characterScopeForLine(doc, line)
	if len(scope.names) > 0 {
		return scope.names
	}
	return index.documentCharacterNames
}

func headingCompletionCandidates(doc *ast.Document, index *documentIndex, line, level int) []string {
	switch level {
	case 1:
		return nil
	case 2:
		section := topLevelSectionForLine(doc, line)
		candidates := make([]string, 0, 2)
		if scope := index.characterScopeForSection(section); section != nil && scope.dp == nil {
			candidates = append(candidates, "Dramatis Personae")
		}
		candidates = append(candidates, nextActHeading(index, line))
		return candidates
	case 3:
		return []string{nextSceneHeading(index, line)}
	default:
		return nil
	}
}

func sceneCompletionCandidates(doc *ast.Document, index *documentIndex, content string, line int) ([]string, bool) {
	if !followsBlankLine(content, line) {
		return nil, false
	}

	scene := index.sceneForLine(line)
	if scene == nil {
		return nil, false
	}

	speakers := index.sceneSpeakersBeforeLine(scene, line)
	ranked := rankRecentSpeakers(speakers)
	return appendRemainingDPNames(doc, line, ranked), true
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

func appendRemainingDPNames(doc *ast.Document, line int, names []string) []string {
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		seen[strings.ToUpper(strings.TrimSpace(name))] = struct{}{}
	}

	if dp := scopedDramatisPersonae(doc, line); dp != nil {
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

func nextActHeading(index *documentIndex, line int) string {
	count := 0
	for _, act := range index.acts {
		if act.Range.Start.Line < line {
			count++
		}
	}
	return "ACT " + romanNumeral(count+1)
}

func nextSceneHeading(index *documentIndex, line int) string {
	if act := index.actForLine(line); act != nil {
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
	for _, scene := range index.scenes {
		if scene.Range.Start.Line < line {
			count++
		}
	}
	return fmt.Sprintf("SCENE %d", count+1)
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

func lineAt(content string, lineNum int) (string, bool) {
	if lineNum < 0 {
		return "", false
	}
	start := 0
	for i := 0; i < lineNum; i++ {
		idx := strings.IndexByte(content[start:], '\n')
		if idx < 0 {
			return "", false
		}
		start += idx + 1
	}
	end := strings.IndexByte(content[start:], '\n')
	if end < 0 {
		return content[start:], true
	}
	return content[start : start+end], true
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
