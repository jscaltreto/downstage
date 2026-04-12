package lsp

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

type characterScope struct {
	dp    *ast.Section
	names []string
	known map[string]struct{}
}

func newCharacterScope(dp *ast.Section) characterScope {
	scope := characterScope{
		dp:    dp,
		known: make(map[string]struct{}),
	}
	if dp == nil {
		return scope
	}

	seen := make(map[string]struct{})
	for _, ch := range dp.AllCharacters() {
		name := strings.TrimSpace(ch.Name)
		if name != "" {
			key := strings.ToUpper(name)
			scope.known[key] = struct{}{}
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				scope.names = append(scope.names, name)
			}
		}
		for _, alias := range ch.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			scope.known[strings.ToUpper(alias)] = struct{}{}
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
