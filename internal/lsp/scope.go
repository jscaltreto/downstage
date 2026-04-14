package lsp

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

type characterScope struct {
	dp    *ast.Section
	names []string
	known map[string]struct{}
	// entries preserves DP entry order with per-entry key data for
	// duplicate detection and no-dialogue tracking. Each entry corresponds
	// to one Character row in the DP section.
	entries []scopeEntry
	// nameKeyOccurrences maps uppercase name key → entries whose primary
	// name matches. A second occurrence is a duplicate primary name.
	nameKeyOccurrences map[string][]int
	// aliasKeyOccurrences maps uppercase key → entries whose alias (or
	// primary name) matches. Collisions across entries flag duplicate
	// aliases, including name-vs-alias collisions.
	aliasKeyOccurrences map[string][]aliasOccurrence
}

type scopeEntry struct {
	character ast.Character
	// nameKey is the uppercase primary name.
	nameKey string
	// aliasKeys are uppercase alias keys (not including the primary name).
	aliasKeys []string
}

type aliasOccurrence struct {
	entryIndex int
	// aliasIndex is the position of the alias within the entry's Aliases
	// slice, or -1 when the occurrence is the entry's primary name.
	aliasIndex int
}

func newCharacterScope(dp *ast.Section) characterScope {
	scope := characterScope{
		dp:                  dp,
		known:               make(map[string]struct{}),
		nameKeyOccurrences:  make(map[string][]int),
		aliasKeyOccurrences: make(map[string][]aliasOccurrence),
	}
	if dp == nil {
		return scope
	}

	seen := make(map[string]struct{})
	addName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToUpper(name)
		scope.known[key] = struct{}{}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		scope.names = append(scope.names, name)
	}

	for _, ch := range dp.AllCharacters() {
		addName(ch.Name)
		for _, alias := range ch.Aliases {
			addName(alias)
		}

		entry := scopeEntry{character: ch}
		if trimmed := strings.TrimSpace(ch.Name); trimmed != "" {
			entry.nameKey = strings.ToUpper(trimmed)
		}
		for _, alias := range ch.Aliases {
			if trimmed := strings.TrimSpace(alias); trimmed != "" {
				entry.aliasKeys = append(entry.aliasKeys, strings.ToUpper(trimmed))
			}
		}
		scope.entries = append(scope.entries, entry)

		entryIdx := len(scope.entries) - 1
		if entry.nameKey != "" {
			scope.nameKeyOccurrences[entry.nameKey] = append(scope.nameKeyOccurrences[entry.nameKey], entryIdx)
			scope.aliasKeyOccurrences[entry.nameKey] = append(scope.aliasKeyOccurrences[entry.nameKey], aliasOccurrence{entryIndex: entryIdx, aliasIndex: -1})
		}
		for i, key := range entry.aliasKeys {
			scope.aliasKeyOccurrences[key] = append(scope.aliasKeyOccurrences[key], aliasOccurrence{entryIndex: entryIdx, aliasIndex: i})
		}
	}

	return scope
}

func topLevelSectionForLine(doc *ast.Document, line int) *ast.Section {
	if doc == nil {
		return nil
	}
	return ast.FindTopLevelSection(doc.Body, line)
}

func hasScopedDramatisPersonae(doc *ast.Document) bool {
	if doc == nil {
		return false
	}
	for _, node := range doc.Body {
		section, ok := node.(*ast.Section)
		if !ok || section.Level != 1 {
			continue
		}
		if ast.FindDramatisPersonaeInSection(section) != nil {
			return true
		}
	}
	return false
}

func scopedDramatisPersonae(doc *ast.Document, line int) *ast.Section {
	if doc == nil {
		return nil
	}
	if section := topLevelSectionForLine(doc, line); section != nil {
		if dp := ast.FindDramatisPersonaeInSection(section); dp != nil {
			return dp
		}
	}
	if hasScopedDramatisPersonae(doc) {
		return nil
	}
	return ast.FindDramatisPersonae(doc.Body)
}
