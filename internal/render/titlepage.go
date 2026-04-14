package render

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

func PlayableTopLevelSections(doc *ast.Document) []*ast.Section {
	if doc == nil {
		return nil
	}

	sections := make([]*ast.Section, 0)
	for _, node := range doc.Body {
		section, ok := node.(*ast.Section)
		if !ok || section.Level != 1 {
			continue
		}
		if sectionHasPlayableContent(section) {
			sections = append(sections, section)
		}
	}
	return sections
}

func IsInlinePlaySection(doc *ast.Document, section *ast.Section) bool {
	if section == nil || section.Level != 1 {
		return false
	}
	if !sectionHasPlayableContent(section) {
		return false
	}
	topLevelCount := 0
	if doc != nil {
		for _, node := range doc.Body {
			candidate, ok := node.(*ast.Section)
			if !ok || candidate.Level != 1 {
				continue
			}
			topLevelCount++
			if topLevelCount > 1 {
				return true
			}
		}
	}
	return false
}

func SectionTitlePage(doc *ast.Document, section *ast.Section) *ast.TitlePage {
	if section == nil || section.Level != 1 || strings.TrimSpace(section.Title) == "" || IsInlinePlaySection(doc, section) {
		return nil
	}
	if section.Metadata == nil && !sectionHasPlayableContent(section) {
		return nil
	}

	metadataEntries := 0
	if section.Metadata != nil {
		metadataEntries = len(section.Metadata.Entries)
	}

	entries := make([]ast.KeyValue, 0, metadataEntries+1)
	hasTitle := false
	if section.Metadata != nil {
		for _, entry := range section.Metadata.Entries {
			if strings.EqualFold(strings.TrimSpace(entry.Key), "title") {
				hasTitle = true
			}
			entries = append(entries, entry)
		}
	}

	if !hasTitle {
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

// CharacterDisplayName returns "NAME" or "NAME/ALIAS[/ALIAS2]" using the
// authoring syntax, so rendered output matches how users write aliases in
// the source file.
func CharacterDisplayName(ch ast.Character) string {
	name := strings.TrimSpace(ch.Name)
	if len(ch.Aliases) == 0 {
		return name
	}
	var b strings.Builder
	b.WriteString(name)
	for _, alias := range ch.Aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		b.WriteByte('/')
		b.WriteString(alias)
	}
	return b.String()
}

// SectionDisplayTitle returns the title used to display a section in rendered
// output. When Metadata carries an explicit `Title:` entry it wins over the
// heading text, matching SPEC §4.
func SectionDisplayTitle(section *ast.Section) string {
	if section == nil {
		return ""
	}
	if section.Metadata != nil {
		for _, entry := range section.Metadata.Entries {
			if strings.EqualFold(strings.TrimSpace(entry.Key), "title") {
				if v := strings.TrimSpace(entry.Value); v != "" {
					return v
				}
			}
		}
	}
	return section.Title
}

// DramatisPersonaeDisplayTitle returns the DP heading text or the default.
func DramatisPersonaeDisplayTitle(section *ast.Section) string {
	if section == nil {
		return "Dramatis Personae"
	}
	if title := strings.TrimSpace(section.Title); title != "" {
		return title
	}
	return "Dramatis Personae"
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
		if tp := SectionTitlePage(doc, section); tp != nil {
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
		if section.Level == 1 && !sectionHasPlayableContent(section) {
			continue
		}
		return true
	}
	return false
}

func sectionHasPlayableContent(section *ast.Section) bool {
	if section == nil || section.Level != 1 {
		return false
	}
	for _, child := range section.Children {
		switch node := child.(type) {
		case *ast.Section:
			switch node.Kind {
			case ast.SectionDramatisPersonae, ast.SectionAct, ast.SectionScene:
				return true
			}
			if node.Level == 0 {
				return true
			}
		case *ast.Comment:
			continue
		default:
			return true
		}
	}
	return false
}
