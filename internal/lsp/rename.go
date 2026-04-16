package lsp

import (
	"errors"
	"strings"
	"unicode"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

var errRenameInvalid = errors.New("rename: invalid request")

type renameTarget struct {
	character   *ast.Character
	upperKey    string
	cursorRange token.Range
	kind        renameKind
	aliasIndex  int
}

func (t *renameTarget) declRange() token.Range {
	if t.character == nil {
		return token.Range{}
	}
	switch t.kind {
	case renameKindAlias:
		return t.character.AliasRange(t.aliasIndex)
	default:
		return t.character.NameRange()
	}
}

type renameKind int

const (
	renameKindPrimary renameKind = iota
	renameKindAlias
)

func computePrepareRename(doc *ast.Document, pos protocol.Position) *protocol.Range {
	if doc == nil {
		return nil
	}
	target := findRenameTarget(doc, pos)
	if target == nil {
		return nil
	}
	r := toLSPRange(target.cursorRange)
	return &r
}

func computeRename(doc *ast.Document, uri protocol.DocumentURI, pos protocol.Position, newName string) (*protocol.WorkspaceEdit, error) {
	if doc == nil {
		return nil, errRenameInvalid
	}
	target := findRenameTarget(doc, pos)
	if target == nil {
		return nil, errRenameInvalid
	}
	cleaned := strings.TrimSpace(newName)
	if cleaned == "" {
		return nil, errRenameInvalid
	}
	if !isValidCharacterName(cleaned) {
		return nil, errRenameInvalid
	}
	if hasNameConflict(doc, target, cleaned) {
		return nil, errRenameInvalid
	}

	declRange := target.declRange()
	if isZeroRange(declRange) {
		return nil, errRenameInvalid
	}
	edits := []protocol.TextEdit{
		{Range: toLSPRange(declRange), NewText: cleaned},
	}

	scope, ok := renameScope(doc, declRange.Start.Line)
	if !ok {
		return nil, errRenameInvalid
	}
	visitDialoguesInScope(doc, scope, func(dlg *ast.Dialogue) {
		cueName := strings.TrimSpace(dlg.Character)
		if cueName == "" {
			return
		}
		nameRange := dlg.NameRange()

		if strings.ToUpper(cueName) == target.upperKey {
			replacement := matchCueCasing(cueName, cleaned)
			if replacement != cueName {
				edits = append(edits, protocol.TextEdit{
					Range:   toLSPRange(nameRange),
					NewText: replacement,
				})
			}
			return
		}

		parts := splitConjunctionCueWithOffsets(dlg.Character)
		for _, part := range parts {
			if strings.ToUpper(part.Name) != target.upperKey {
				continue
			}
			replacement := matchCueCasing(part.Name, cleaned)
			if replacement == part.Name {
				continue
			}
			edits = append(edits, protocol.TextEdit{
				Range:   toLSPRange(subRangeWithinCue(nameRange, dlg.Character, part.Start, part.End)),
				NewText: replacement,
			})
		}
	})

	return &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			uri: edits,
		},
	}, nil
}

func findRenameTarget(doc *ast.Document, pos protocol.Position) *renameTarget {
	if target := findRenameTargetInDP(doc, pos); target != nil {
		return target
	}
	return findRenameTargetInCue(doc, pos)
}

func findRenameTargetInDP(doc *ast.Document, pos protocol.Position) *renameTarget {
	for _, node := range doc.Body {
		section, ok := node.(*ast.Section)
		if !ok {
			continue
		}
		if target := findRenameTargetInSection(section, pos); target != nil {
			return target
		}
	}
	return nil
}

func findRenameTargetInSection(section *ast.Section, pos protocol.Position) *renameTarget {
	if section.Kind == ast.SectionDramatisPersonae {
		if target := scanCharactersForPosition(section.Characters, pos); target != nil {
			return target
		}
		for gi := range section.Groups {
			if target := scanCharactersForPosition(section.Groups[gi].Characters, pos); target != nil {
				return target
			}
		}
	}
	for _, child := range section.Children {
		if cs, ok := child.(*ast.Section); ok {
			if target := findRenameTargetInSection(cs, pos); target != nil {
				return target
			}
		}
	}
	return nil
}

func scanCharactersForPosition(characters []ast.Character, pos protocol.Position) *renameTarget {
	for i := range characters {
		ch := &characters[i]
		if positionInRange(pos, ch.NameRange()) && strings.TrimSpace(ch.Name) != "" {
			return &renameTarget{
				character:   ch,
				upperKey:    strings.ToUpper(strings.TrimSpace(ch.Name)),
				cursorRange: ch.NameRange(),
				kind:        renameKindPrimary,
				aliasIndex:  -1,
			}
		}
		for ai, alias := range ch.Aliases {
			r := ch.AliasRange(ai)
			if isZeroRange(r) {
				continue
			}
			if positionInRange(pos, r) && strings.TrimSpace(alias) != "" {
				return &renameTarget{
					character:   ch,
					upperKey:    strings.ToUpper(strings.TrimSpace(alias)),
					cursorRange: r,
					kind:        renameKindAlias,
					aliasIndex:  ai,
				}
			}
		}
	}
	return nil
}

func findRenameTargetInCue(doc *ast.Document, pos protocol.Position) *renameTarget {
	cueName, cueRange := findCueAtPosition(doc, pos)
	if cueName == "" {
		return nil
	}
	if splitConjunctionCue(cueName) != nil {
		return nil
	}

	dp := scopedDramatisPersonae(doc, int(pos.Line))
	if dp == nil {
		return nil
	}
	upperCue := strings.ToUpper(strings.TrimSpace(cueName))
	for _, ch := range collectAllCharacters(dp) {
		if strings.ToUpper(strings.TrimSpace(ch.Name)) == upperCue {
			return &renameTarget{
				character:   ch,
				upperKey:    upperCue,
				cursorRange: cueRange,
				kind:        renameKindPrimary,
				aliasIndex:  -1,
			}
		}
		for ai, alias := range ch.Aliases {
			if strings.ToUpper(strings.TrimSpace(alias)) == upperCue {
				return &renameTarget{
					character:   ch,
					upperKey:    upperCue,
					cursorRange: cueRange,
					kind:        renameKindAlias,
					aliasIndex:  ai,
				}
			}
		}
	}
	return nil
}

func collectAllCharacters(dp *ast.Section) []*ast.Character {
	var out []*ast.Character
	for i := range dp.Characters {
		out = append(out, &dp.Characters[i])
	}
	for gi := range dp.Groups {
		for ci := range dp.Groups[gi].Characters {
			out = append(out, &dp.Groups[gi].Characters[ci])
		}
	}
	return out
}

func findCueAtPosition(doc *ast.Document, pos protocol.Position) (string, token.Range) {
	for _, n := range doc.Body {
		if name, r := findCueInNode(n, pos); name != "" {
			return name, r
		}
	}
	return "", token.Range{}
}

func findCueInNode(n ast.Node, pos protocol.Position) (string, token.Range) {
	line := int(pos.Line)
	switch v := n.(type) {
	case *ast.DualDialogue:
		if name, r := findCueInNode(v.Left, pos); name != "" {
			return name, r
		}
		return findCueInNode(v.Right, pos)
	case *ast.Dialogue:
		r := v.NameRange()
		if r.Start.Line == line && int(pos.Character) >= r.Start.Column && int(pos.Character) < r.End.Column {
			return v.Character, r
		}
	case *ast.Song:
		for _, child := range v.Content {
			if name, r := findCueInNode(child, pos); name != "" {
				return name, r
			}
		}
	case *ast.Section:
		for _, child := range v.Children {
			if name, r := findCueInNode(child, pos); name != "" {
				return name, r
			}
		}
	}
	return "", token.Range{}
}

func hasNameConflict(doc *ast.Document, target *renameTarget, newName string) bool {
	dp := scopedDramatisPersonae(doc, target.cursorRange.Start.Line)
	if dp == nil {
		return false
	}
	upperNew := strings.ToUpper(strings.TrimSpace(newName))
	if upperNew == target.upperKey {
		return false
	}
	for _, ch := range collectAllCharacters(dp) {
		nameKey := strings.ToUpper(strings.TrimSpace(ch.Name))
		if ch == target.character && target.kind == renameKindPrimary {
		} else if nameKey != "" && nameKey == upperNew {
			return true
		}
		for ai, alias := range ch.Aliases {
			aliasKey := strings.ToUpper(strings.TrimSpace(alias))
			if ch == target.character && target.kind == renameKindAlias && ai == target.aliasIndex {
				continue
			}
			if aliasKey != "" && aliasKey == upperNew {
				return true
			}
		}
	}
	return false
}

func renameScope(doc *ast.Document, line int) (*ast.Section, bool) {
	if play := topLevelSectionForLine(doc, line); play != nil {
		return play, true
	}
	if hasScopedDramatisPersonae(doc) {
		return nil, false
	}
	return nil, true
}

func visitDialoguesInScope(doc *ast.Document, play *ast.Section, fn func(*ast.Dialogue)) {
	var walk func(node ast.Node)
	walk = func(node ast.Node) {
		switch v := node.(type) {
		case *ast.Dialogue:
			fn(v)
		case *ast.DualDialogue:
			if v.Left != nil {
				walk(v.Left)
			}
			if v.Right != nil {
				walk(v.Right)
			}
		case *ast.Song:
			for _, child := range v.Content {
				walk(child)
			}
		case *ast.Section:
			for _, child := range v.Children {
				walk(child)
			}
		}
	}

	if play != nil {
		walk(play)
		return
	}
	for _, node := range doc.Body {
		walk(node)
	}
}

func positionInRange(pos protocol.Position, r token.Range) bool {
	if int(pos.Line) != r.Start.Line || r.Start.Line != r.End.Line {
		return false
	}
	col := int(pos.Character)
	return col >= r.Start.Column && col < r.End.Column
}

func isValidCharacterName(name string) bool {
	hasLetter := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r), r == ' ', r == '\'', r == '-', r == '.':
		default:
			return false
		}
	}
	if !hasLetter {
		return false
	}
	if strings.Contains(name, "/") {
		return false
	}
	if strings.Contains(name, " - ") {
		return false
	}
	return true
}

func matchCueCasing(existing, replacement string) string {
	trimmedReplacement := strings.TrimSpace(replacement)
	if trimmedReplacement == "" {
		return replacement
	}
	switch {
	case isAllUpper(existing):
		return strings.ToUpper(trimmedReplacement)
	case isAllLower(existing):
		return strings.ToLower(trimmedReplacement)
	default:
		return trimmedReplacement
	}
}

func isAllUpper(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}

func isAllLower(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsLower(r) {
				return false
			}
		}
	}
	return hasLetter
}

func isZeroRange(r token.Range) bool {
	return r.Start == (token.Position{}) && r.End == (token.Position{})
}

func subRangeWithinCue(cueRange token.Range, cueText string, startByte, endByte int) token.Range {
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(cueText) {
		endByte = len(cueText)
	}
	startUTF16 := token.UTF16Len(cueText[:startByte])
	endUTF16 := token.UTF16Len(cueText[:endByte])
	return token.Range{
		Start: token.Position{
			Line:   cueRange.Start.Line,
			Column: cueRange.Start.Column + startUTF16,
			Offset: cueRange.Start.Offset + startByte,
		},
		End: token.Position{
			Line:   cueRange.Start.Line,
			Column: cueRange.Start.Column + endUTF16,
			Offset: cueRange.Start.Offset + endByte,
		},
	}
}
