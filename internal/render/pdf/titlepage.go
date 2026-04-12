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
		r.pdf.Ln(r.lineHeight)
		for _, author := range authors {
			r.centeredWrappedText(author, r.lineHeight)
			r.pdf.Ln(r.lineHeight)
		}
	}

	// Remaining metadata near the bottom
	if len(other) > 0 {
		r.pdf.SetY(r.pageH * 0.70)
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
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

	b.pdf.SetFont(b.cfg.FontFamily, "B", titleSize)
	b.centeredWrappedText(strings.ToUpper(strings.TrimSpace(render.SectionDisplayTitle(section))), b.lineHeight)
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
