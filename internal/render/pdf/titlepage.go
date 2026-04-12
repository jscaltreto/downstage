package pdf

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

func (r *pdfRenderer) RenderTitlePage(tp *ast.TitlePage) error {
	r.beginTitlePage()

	title, subtitle, authors, other := partitionTitlePageEntries(tp)

	r.hasTitlePage = true
	r.titlePageTitle = title

	if t := strings.TrimSpace(title); t != "" {
		r.pdf.Bookmark(t, 0, -1)
	}

	// Center vertically: place title roughly at 35% down the page
	titleY := r.pageH * 0.35

	if title != "" {
		r.pdf.SetY(titleY)
		r.pdf.SetFont(r.cfg.FontFamily, "B", r.cfg.FontSize+8)
		r.centeredWrappedText(strings.ToUpper(title), r.lineHeight)
		r.pdf.Ln(r.lineHeight)
	}

	if subtitle != "" {
		r.pdf.SetFont(r.cfg.FontFamily, "I", r.cfg.FontSize+2)
		r.centeredWrappedText(subtitle, r.lineHeight)
		r.pdf.Ln(r.lineHeight)
	}

	if len(authors) > 0 {
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize+2)
		r.pdf.Ln(r.lineHeight * 2)
		r.centeredWrappedText("by", r.lineHeight)
		r.pdf.Ln(r.lineHeight * 0.5)
		for _, author := range authors {
			r.centeredWrappedText(author, r.lineHeight)
		}
	}

	// Remaining metadata near the bottom.
	if len(other) > 0 {
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
		placeBottomBlock(&r.pdfBase, len(other))
		for _, kv := range other {
			r.centeredWrappedText(kv.Key+": "+kv.Value, r.lineHeight)
		}
	}

	r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
	r.fontStyle = ""
	r.finishTitlePage()
	return nil
}

func renderInlinePlayHeader(b *pdfBase, section *ast.Section, titleSize float64, metadataSize float64) {
	if section == nil {
		return
	}
	_, _, authors, other := partitionTitlePageEntries(section.Metadata)

	displayTitle := strings.TrimSpace(render.SectionDisplayTitle(section))
	if displayTitle != "" {
		b.pdf.Bookmark(displayTitle, 0, -1)
	}

	b.pdf.SetFont(b.cfg.FontFamily, "B", titleSize)
	b.centeredWrappedText(strings.ToUpper(displayTitle), b.lineHeight)
	b.pdf.Ln(b.lineHeight * 2)

	b.pdf.SetFont(b.cfg.FontFamily, "", metadataSize)
	if len(authors) > 0 {
		b.centeredWrappedText("by", b.lineHeight)
		b.pdf.Ln(b.lineHeight)
		for _, author := range authors {
			b.centeredWrappedText(author, b.lineHeight)
			b.pdf.Ln(b.lineHeight)
		}
	}
	for _, kv := range other {
		b.centeredWrappedText(kv.Key+": "+kv.Value, b.lineHeight)
		b.pdf.Ln(b.lineHeight)
	}

	b.pdf.SetFont(b.cfg.FontFamily, "", b.cfg.FontSize)
	b.fontStyle = ""
}

// bookmarkSection records an outline entry for the given section at the
// current cursor position. Levels are precomputed from the AST so that
// scenes always attach to their real parent in the source, regardless
// of ordering within a mixed play (scenes-then-acts, acts-then-scenes,
// or interleaved).
func bookmarkSection(b *pdfBase, s *ast.Section) {
	if s == nil || b.pdf == nil {
		return
	}
	label := sectionOutlineLabel(s)
	if label == "" {
		return
	}
	level, ok := b.outlineLevels[s]
	if !ok {
		return
	}
	b.pdf.Bookmark(label, level, -1)
}

// buildOutlineLevels walks the AST once and assigns each structural
// section a PDF outline level based on its ancestors: level-1 generic
// sections (plays) are 0, acts are 1, and scenes are 2 when they live
// inside an act or 1 when they're direct children of a play.
func buildOutlineLevels(doc *ast.Document) map[*ast.Section]int {
	levels := make(map[*ast.Section]int)
	if doc == nil {
		return levels
	}
	var walk func(nodes []ast.Node, insideAct bool)
	walk = func(nodes []ast.Node, insideAct bool) {
		for _, node := range nodes {
			switch v := node.(type) {
			case *ast.Section:
				switch v.Kind {
				case ast.SectionGeneric:
					if v.Level == 1 {
						levels[v] = 0
					}
					walk(v.Children, false)
				case ast.SectionAct:
					levels[v] = 1
					walk(v.Children, true)
				case ast.SectionScene:
					if insideAct {
						levels[v] = 2
					} else {
						levels[v] = 1
					}
					walk(v.Children, insideAct)
				default:
					walk(v.Children, insideAct)
				}
			case *ast.Song:
				walk(v.Content, insideAct)
			}
		}
	}
	walk(doc.Body, false)
	return levels
}

func sectionOutlineLabel(s *ast.Section) string {
	switch s.Kind {
	case ast.SectionAct:
		return strings.TrimSpace(buildOutlineHeader("ACT", s.Number, s.Title))
	case ast.SectionScene:
		return strings.TrimSpace(buildOutlineHeader("SCENE", s.Number, s.Title))
	case ast.SectionGeneric:
		if s.Level == 1 {
			return strings.TrimSpace(render.SectionDisplayTitle(s))
		}
	}
	return ""
}

func buildOutlineHeader(keyword, number, title string) string {
	number = strings.TrimSpace(number)
	title = strings.TrimSpace(title)
	switch {
	case number != "" && title != "":
		return keyword + " " + number + ": " + title
	case number != "":
		return keyword + " " + number
	case title != "":
		return keyword + ": " + title
	default:
		return keyword
	}
}

// applyDocumentMetadata copies the document's title-page fields into
// the PDF's document-info dictionary (Title / Author / Subject /
// Keywords) so readers and libraries can see structured metadata.
func applyDocumentMetadata(b *pdfBase, tp *ast.TitlePage) {
	title, subtitle, authors, other := partitionTitlePageEntries(tp)
	if t := strings.TrimSpace(title); t != "" {
		b.pdf.SetTitle(t, true)
	}
	if len(authors) > 0 {
		b.pdf.SetAuthor(strings.Join(authors, ", "), true)
	}
	if s := strings.TrimSpace(subtitle); s != "" {
		b.pdf.SetSubject(s, true)
	}
	var keywords []string
	for _, kv := range other {
		switch strings.ToLower(strings.TrimSpace(kv.Key)) {
		case "keywords", "tags":
			keywords = append(keywords, kv.Value)
		}
	}
	if len(keywords) > 0 {
		b.pdf.SetKeywords(strings.Join(keywords, ", "), true)
	}
	b.pdf.SetCreator("Downstage", true)
}

// placeBottomBlock anchors a footer block of entryCount lines just
// above the bottom margin. The reserve allows for a two-line wrap per
// entry so long values (e.g. a draft note) don't spill into the
// bottom margin and trip the auto page break. If the author block
// above has already spilled into the reserved area, fall back to a
// short gap below them instead of backtracking over live content.
func placeBottomBlock(b *pdfBase, entryCount int) {
	reserve := float64(entryCount) * b.lineHeight * 2
	target := b.pageH - b.marginB - reserve
	if current := b.pdf.GetY(); current > target {
		b.pdf.Ln(b.lineHeight)
		return
	}
	b.pdf.SetY(target)
}

func partitionTitlePageEntries(tp *ast.TitlePage) (title string, subtitle string, authors []string, other []ast.KeyValue) {
	if tp == nil {
		return "", "", nil, nil
	}
	for _, kv := range tp.Entries {
		switch strings.ToLower(strings.TrimSpace(kv.Key)) {
		case "title":
			title = kv.Value
		case "subtitle":
			subtitle = kv.Value
		case "author":
			if strings.TrimSpace(kv.Value) != "" {
				authors = append(authors, kv.Value)
			}
		default:
			other = append(other, kv)
		}
	}
	return title, subtitle, authors, other
}
