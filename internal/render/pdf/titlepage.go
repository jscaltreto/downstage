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
// current cursor position. Sections that don't map cleanly to an outline
// level (forced headings, prose sections) are skipped.
func bookmarkSection(b *pdfBase, s *ast.Section) {
	if s == nil || b.pdf == nil {
		return
	}
	label := sectionOutlineLabel(s)
	if label == "" {
		return
	}
	level := sectionOutlineLevel(s)
	if level < 0 {
		return
	}
	b.pdf.Bookmark(label, level, -1)
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

func sectionOutlineLevel(s *ast.Section) int {
	switch s.Kind {
	case ast.SectionGeneric:
		if s.Level == 1 {
			return 0
		}
	case ast.SectionAct:
		return 1
	case ast.SectionScene:
		return 2
	}
	return -1
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

// placeBottomBlock positions the cursor for a footer-ish block of
// entryCount lines. The preferred anchor is 70% of the page height; if
// that would push the block past the bottom margin (e.g. condensed
// layouts with lots of wrapping), it's shifted up just far enough to
// fit. When the author block above has already spilled past the target,
// we fall back to a short gap below the authors instead of backtracking.
func placeBottomBlock(b *pdfBase, entryCount int) {
	target := b.pageH * 0.70
	// Reserve three line-heights per entry so the occasional wrapped
	// metadata line doesn't overshoot the bottom margin.
	reserve := float64(entryCount) * b.lineHeight * 3
	if bottom := b.pageH - b.marginB - reserve; bottom < target {
		target = bottom
	}
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
