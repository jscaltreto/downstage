package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

const condensedLineHeight = 4.5 // mm

// Half-letter page size: 5.5" x 8.5"
const (
	halfLetterW = 139.7 // mm
	halfLetterH = 215.9 // mm
)

var _ render.NodeRenderer = (*condensedRenderer)(nil)

// NewCondensedRenderer creates an acting-edition PDF NodeRenderer.
// Uses half-letter page, Libre Baskerville serif font, and compact layout
// with character name + dialogue on the same line.
func NewCondensedRenderer(cfg render.Config) render.NodeRenderer {
	// Override config for acting edition defaults
	cfg.FontSize = 10
	cfg.MarginTop = 36 // 0.5 inch
	cfg.MarginBottom = 36
	cfg.MarginLeft = 36
	cfg.MarginRight = 36
	return &condensedRenderer{
		pdfBase: pdfBase{cfg: cfg, lineHeight: condensedLineHeight},
	}
}

type condensedRenderer struct {
	pdfBase
	inDialogue bool // tracks whether we're inside a dialogue block
	firstLine  bool // tracks first line of a dialogue (continues after character name)
}

// --- Lifecycle ---

func (r *condensedRenderer) BeginDocument(doc *ast.Document, w io.Writer) error {
	r.w = w
	r.hasTitlePage = doc.TitlePage != nil
	r.hasBody = len(doc.Body) > 0
	r.titlePageTitle = titlePageTitle(doc.TitlePage)
	r.initCondensedPDF()
	return nil
}

func (r *condensedRenderer) EndDocument(_ *ast.Document) error {
	return r.pdf.Output(r.w)
}

func (r *condensedRenderer) initCondensedPDF() {
	r.pdf = newCustomSizePDF(halfLetterW, halfLetterH)

	// Convert points to mm
	r.marginL = r.cfg.MarginLeft * 0.3528
	r.marginR = r.cfg.MarginRight * 0.3528
	r.marginT = r.cfg.MarginTop * 0.3528
	r.marginB = r.cfg.MarginBottom * 0.3528

	r.pdf.SetMargins(r.marginL, r.marginT, r.marginR)
	r.pdf.SetAutoPageBreak(true, r.marginB)

	r.pageW, r.pageH = r.pdf.GetPageSize()
	r.bodyW = r.pageW - r.marginL - r.marginR

	if r.cfg.FontPath != "" {
		if loadCustomFont(r.pdf, "CustomFont", r.cfg.FontPath) {
			r.cfg.FontFamily = "CustomFont"
		} else {
			loadSerifFont(r.pdf)
			r.cfg.FontFamily = serifFontFamily
		}
	} else {
		loadSerifFont(r.pdf)
		r.cfg.FontFamily = serifFontFamily
	}
	r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)

	// Page numbers
	r.pdf.AliasNbPages("")
	r.pdf.SetFooterFunc(func() {
		r.pdf.SetY(-r.marginB + 3)
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize-2)
		r.pdf.CellFormat(0, 8, fmt.Sprintf("%d", r.pdf.PageNo()),
			"", 0, "C", false, 0, "")
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
	})

	r.pdf.AddPage()
	r.fontStyle = ""
}

// --- Front matter ---

func (r *condensedRenderer) RenderTitlePage(tp *ast.TitlePage) error {
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

	titleY := r.pageH * 0.30

	if title != "" {
		r.pdf.SetY(titleY)
		r.pdf.SetFont(r.cfg.FontFamily, "B", r.cfg.FontSize+6)
		r.centeredText(strings.ToUpper(title))
		r.pdf.Ln(r.lineHeight)
	}

	if subtitle != "" {
		r.pdf.SetFont(r.cfg.FontFamily, "I", r.cfg.FontSize+1)
		r.pdf.Ln(r.lineHeight)
		r.centeredText(subtitle)
		r.pdf.Ln(r.lineHeight)
	}

	if author != "" {
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize+1)
		r.pdf.Ln(r.lineHeight * 2)
		r.centeredText("by")
		r.pdf.Ln(r.lineHeight)
		r.centeredText(author)
		r.pdf.Ln(r.lineHeight)
	}

	if len(other) > 0 {
		r.pdf.SetY(r.pageH * 0.70)
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize-1)
		for _, kv := range other {
			r.centeredText(kv.Key + ": " + kv.Value)
		}
	}

	r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
	r.fontStyle = ""
	if r.hasBody {
		r.pdf.AddPage()
	}
	return nil
}

// --- Structural ---

func (r *condensedRenderer) BeginSection(s *ast.Section) error {
	switch s.Kind {
	case ast.SectionAct:
		return r.beginAct(s)
	case ast.SectionScene:
		return r.beginScene(s)
	case ast.SectionDramatisPersonae:
		renderDramatisPersonae(&r.pdfBase, s, 0)
		return nil
	default: // SectionGeneric
		if r.hasTitlePage && s.Level == 1 && strings.EqualFold(strings.TrimSpace(s.Title), r.titlePageTitle) {
			return nil
		}
		if s.Level == 0 {
			r.ensureSpace(r.lineHeight * 3)
			r.pdf.Ln(r.lineHeight)
			r.setStyle("B")
			r.centeredText(s.Title)
			r.setStyle("")
			r.pdf.Ln(r.lineHeight)
			return nil
		}
		r.pdf.AddPage()
		if s.Title != "" {
			r.setStyle("B")
			r.centeredText(strings.ToUpper(s.Title))
			r.setStyle("")
			r.pdf.Ln(r.lineHeight * 2)
		}
		return nil
	}
}

func (r *condensedRenderer) EndSection(_ *ast.Section) error {
	return nil
}

func (r *condensedRenderer) BeginSectionLine(sl *ast.SectionLine) error {
	if len(sl.Content) == 0 {
		r.pdf.Ln(r.lineHeight * 2)
	}
	return nil
}

func (r *condensedRenderer) EndSectionLine(sl *ast.SectionLine) error {
	if len(sl.Content) > 0 {
		r.pdf.Write(r.lineHeight, " ")
	}
	return nil
}

func (r *condensedRenderer) beginAct(s *ast.Section) error {
	if s.Number != "" {
		r.pdf.AddPage()
	} else {
		if r.hasTitlePage {
			return nil
		}
		r.ensureSpace(r.lineHeight * 3)
		r.pdf.Ln(r.lineHeight)
	}

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "ACT " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "ACT " + s.Number
	default:
		heading = s.Title
	}

	r.setStyle("B")
	r.centeredText(strings.ToUpper(heading))
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

func (r *condensedRenderer) beginScene(s *ast.Section) error {
	r.ensureSpace(r.lineHeight * 3)
	r.pdf.Ln(r.lineHeight)

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "SCENE " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "SCENE " + s.Number
	default:
		heading = s.Title
	}

	r.setStyle("B")
	r.centeredText(strings.ToUpper(heading))
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Dialogue ---
// Acting edition: "HAMLET. (aside) A piece of work is man."
// Character name bold, parenthetical italic, dialogue regular — all on one line.

func (r *condensedRenderer) BeginDialogue(d *ast.Dialogue) error {
	r.ensureSpace(r.lineHeight * 2)
	r.pdf.Ln(r.lineHeight / 2)

	r.inDialogue = true
	r.firstLine = true

	// Character name — bold, followed by period
	r.pdf.SetX(r.marginL)
	r.setStyle("B")
	r.pdf.Write(r.lineHeight, strings.ToUpper(d.Character)+".")
	r.setStyle("")
	r.pdf.Write(r.lineHeight, "  ")

	// Parenthetical — italic, inline
	if d.Parenthetical != "" {
		r.setStyle("I")
		paren := d.Parenthetical
		if len(paren) == 0 || paren[0] != '(' {
			paren = "(" + paren + ")"
		}
		r.pdf.Write(r.lineHeight, paren)
		r.setStyle("")
		r.pdf.Write(r.lineHeight, " ")
	}

	return nil
}

func (r *condensedRenderer) EndDialogue(_ *ast.Dialogue) error {
	r.inDialogue = false
	r.firstLine = false
	return nil
}

func (r *condensedRenderer) BeginDialogueLine(line *ast.DialogueLine) error {
	if r.firstLine {
		// First line continues after character name on the same line
		r.firstLine = false
		if line.IsVerse {
			// Even verse on the first line starts inline
		}
	} else {
		// Subsequent lines start at left margin
		if line.IsVerse {
			r.pdf.SetX(r.marginL + 10) // verse indent
		} else {
			r.pdf.SetX(r.marginL)
		}
	}
	return nil
}

func (r *condensedRenderer) EndDialogueLine(_ *ast.DialogueLine) error {
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Stage Direction ---
// Italic, indented 0.5" further, one blank line above and below.

func (r *condensedRenderer) BeginStageDirection(_ *ast.StageDirection) error {
	r.ensureSpace(r.lineHeight * 3)
	r.pdf.Ln(r.lineHeight)

	stageIndent := 0.5 * 72 * 0.3528 // 0.5 inch in mm
	r.setStyle("I")
	r.pdf.SetLeftMargin(r.marginL + stageIndent)
	r.pdf.SetX(r.marginL + stageIndent)
	return nil
}

func (r *condensedRenderer) EndStageDirection(_ *ast.StageDirection) error {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")
	r.pdf.SetLeftMargin(r.marginL)
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Song ---

func (r *condensedRenderer) BeginSong(song *ast.Song) error {
	r.ensureSpace(r.lineHeight * 3)
	r.pdf.Ln(r.lineHeight)

	header := "SONG"
	if song.Number != "" {
		header = fmt.Sprintf("SONG %s", song.Number)
	}
	if song.Title != "" {
		header += ": " + song.Title
	}

	r.setStyle("B")
	r.centeredText(header)
	r.setStyle("")
	r.pdf.Ln(r.lineHeight / 2)
	return nil
}

func (r *condensedRenderer) EndSong(_ *ast.Song) error {
	r.pdf.Ln(r.lineHeight / 2)
	r.setStyle("B")
	r.centeredText("SONG END")
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Verse Block ---

func (r *condensedRenderer) BeginVerseBlock(_ *ast.VerseBlock) error {
	r.ensureSpace(r.lineHeight * 2)
	return nil
}

func (r *condensedRenderer) EndVerseBlock(_ *ast.VerseBlock) error {
	return nil
}

func (r *condensedRenderer) BeginVerseLine(_ *ast.VerseLine) error {
	r.pdf.SetX(r.marginL + 10)
	return nil
}

func (r *condensedRenderer) EndVerseLine(_ *ast.VerseLine) error {
	r.pdf.Ln(r.lineHeight)
	return nil
}
