package pdf

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

func (r *pdfRenderer) RenderTitlePage(tp *ast.TitlePage) error {
	r.beginTitlePage()

	var title, subtitle, author string
	var other []ast.KeyValue

	for _, kv := range tp.Entries {
		switch strings.ToLower(kv.Key) {
		case "title":
			title = kv.Value
		case "subtitle":
			subtitle = kv.Value
		case "author":
			author = kv.Value
		default:
			other = append(other, kv)
		}
	}

	r.hasTitlePage = true
	r.titlePageTitle = title

	// Center vertically: place title roughly at 35% down the page
	titleY := r.pageH * 0.35

	if title != "" {
		r.pdf.SetY(titleY)
		r.pdf.SetFont(r.cfg.FontFamily, "B", r.cfg.FontSize+8)
		r.centeredText(strings.ToUpper(title))
		r.pdf.Ln(r.lineHeight)
	}

	if subtitle != "" {
		r.pdf.SetFont(r.cfg.FontFamily, "I", r.cfg.FontSize+2)
		r.pdf.Ln(r.lineHeight)
		r.centeredText(subtitle)
		r.pdf.Ln(r.lineHeight)
	}

	if author != "" {
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize+2)
		r.pdf.Ln(r.lineHeight * 2)
		r.centeredText("by")
		r.pdf.Ln(r.lineHeight)
		r.centeredText(author)
		r.pdf.Ln(r.lineHeight)
	}

	// Remaining metadata near the bottom
	if len(other) > 0 {
		r.pdf.SetY(r.pageH * 0.70)
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
		for _, kv := range other {
			r.centeredText(kv.Key + ": " + kv.Value)
		}
	}

	r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
	r.fontStyle = ""
	r.finishTitlePage()
	return nil
}
