package render

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

func SectionTitlePage(section *ast.Section) *ast.TitlePage {
	if section == nil || section.Metadata == nil || section.Level != 1 {
		return nil
	}

	entries := make([]ast.KeyValue, 0, len(section.Metadata.Entries)+1)
	hasTitle := false
	for _, entry := range section.Metadata.Entries {
		if strings.EqualFold(strings.TrimSpace(entry.Key), "title") {
			hasTitle = true
		}
		entries = append(entries, entry)
	}

	if !hasTitle && strings.TrimSpace(section.Title) != "" {
		entries = append([]ast.KeyValue{{
			Key:   "Title",
			Value: section.Title,
			Range: section.HeadingRange(),
		}}, entries...)
	}

	return &ast.TitlePage{
		Entries: entries,
		Range:   section.Range,
	}
}

func DocumentTitlePage(doc *ast.Document) *ast.TitlePage {
	if doc == nil {
		return nil
	}
	if doc.TitlePage != nil {
		return doc.TitlePage
	}
	for _, node := range doc.Body {
		section, ok := node.(*ast.Section)
		if !ok {
			continue
		}
		if tp := SectionTitlePage(section); tp != nil {
			return tp
		}
	}
	return nil
}

func DocumentHasRenderableBody(doc *ast.Document) bool {
	if doc == nil {
		return false
	}
	for _, node := range doc.Body {
		section, ok := node.(*ast.Section)
		if !ok {
			return true
		}
		if section.Level == 1 && section.Metadata != nil && len(section.Children) == 0 && len(section.Lines) == 0 {
			continue
		}
		return true
	}
	return false
}
