package lsp

import (
	"errors"
	"strings"
	"unicode"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

// errRenameInvalid signals that the cursor is not on a renameable symbol,
// or that the requested new name is not acceptable. The handler converts
// this into an LSP error response.
var errRenameInvalid = errors.New("rename: invalid request")

// renameTarget describes the specific spelling the cursor is on. A
// character may be referenced by its primary name or any of its aliases;
// each spelling is treated as its own renameable symbol so writers can
// rename a single alias without rewriting the rest.
type renameTarget struct {
	character *ast.Character
	// oldName is the spelling being renamed (case-preserved as declared
	// in the dramatis personae).
	oldName string
	// upperKey is the case-folded form used to match cues.
	upperKey string
	// cursorRange is the range under the cursor — used by prepareRename
	// to highlight the symbol. It is the dramatis personae entry when
	// the cursor is on a declaration, or the cue range when the cursor
	// is on a dialogue cue.
	cursorRange token.Range
	// kind tracks whether the declaration is the primary name or an alias.
	kind renameKind
	// aliasIndex is the alias position when kind == renameKindAlias.
	aliasIndex int
}

// declRange returns the canonical declaration range for the target —
// always the dramatis personae entry (primary name or alias), never a
// cue. The cue walk in computeRename relies on this so the declaration
// edit does not overlap with the cue edit when rename is triggered
// from a cue.
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

// computePrepareRename returns the source range of the symbol under the
// cursor when rename is supported there, or nil otherwise.
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

// computeRename builds a workspace edit that renames every structural
// reference to the symbol at the cursor. Returns errRenameInvalid when
// the symbol is not renameable or the new name is not acceptable.
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

	// Restrict cue updates to the play that owns the target's DP scope.
	// In a compilation document each top-level play has its own dramatis
	// personae, so a "BOB" in Play A is a different character than
	// "BOB" in Play B. Walking the whole document would rewrite cues
	// belonging to other plays. When the document uses the legacy
	// document-wide DP (no top-level section owns one), play is nil and
	// we fall back to a doc-wide walk.
	scope := renameScope(doc, declRange.Start.Line)
	visitDialoguesInScope(doc, scope, func(dlg *ast.Dialogue) {
		cueName := strings.TrimSpace(dlg.Character)
		if cueName == "" {
			return
		}
		if strings.ToUpper(cueName) != target.upperKey {
			return
		}
		replacement := matchCueCasing(cueName, cleaned)
		if replacement == cueName {
			return
		}
		edits = append(edits, protocol.TextEdit{
			Range:   toLSPRange(dlg.NameRange()),
			NewText: replacement,
		})
	})

	return &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			uri: edits,
		},
	}, nil
}

// findRenameTarget identifies the character spelling under the cursor.
// Supported positions:
//   - primary name in a dramatis personae entry
//   - alias spelling in a dramatis personae entry
//   - dialogue cue whose spelling matches a known character primary name
//     or alias (excluding conjunction cues like "BOB AND JANE")
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
				oldName:     ch.Name,
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
					oldName:     alias,
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
	// Skip conjunction cues — sub-name boundaries aren't ranged so we
	// cannot rename a single participant safely. The user can rename
	// from the dramatis personae entry instead.
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
				oldName:     ch.Name,
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
					oldName:     alias,
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

// collectAllCharacters returns pointers to every character in the section,
// including those nested in groups, so callers can mutate via the same
// slice the parser populated.
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

// findCueAtPosition is a sibling of findCharacterAtPosition that also
// returns the cue's source range, which prepareRename needs.
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

// hasNameConflict returns true when renaming target to newName would
// collide with another known spelling in the same dramatis personae —
// e.g. renaming BOB to JANE when JANE already exists, or renaming an
// alias to a spelling already used by the same entry.
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
			// skip the spelling being renamed
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

// renameScope returns the top-level play section that owns the DP
// resolving the rename target. Returns nil when the document uses a
// document-wide DP (legacy / single-play layout), signalling a doc-wide
// walk is correct.
func renameScope(doc *ast.Document, line int) *ast.Section {
	if !hasScopedDramatisPersonae(doc) {
		return nil
	}
	return topLevelSectionForLine(doc, line)
}

// visitDialoguesInScope walks every dialogue node within the given play
// section. When play is nil, it walks the entire document.
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
		// Character names always sit on a single source line; reject
		// multi-line or off-line positions outright.
		return false
	}
	col := int(pos.Character)
	return col >= r.Start.Column && col <= r.End.Column
}

// isValidCharacterName accepts identifiers a writer could realistically
// type at a cue position. Names must contain at least one letter and may
// include letters, digits, spaces, and a small punctuation set commonly
// used in stage names (apostrophe, hyphen, period). The check stays
// conservative — false rejections are easier to recover from than a
// rename that produces an unparseable script.
func isValidCharacterName(name string) bool {
	hasLetter := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r), r == ' ', r == '\'', r == '-', r == '.':
			// permitted
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
		// Would be parsed as the start of a description.
		return false
	}
	return true
}

// matchCueCasing keeps the casing convention of the existing cue when
// generating the replacement. ALL CAPS cues stay ALL CAPS, lowercase
// stays lowercase, otherwise the user-typed spelling is used as-is.
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
