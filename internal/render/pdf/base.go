package pdf

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

const pointsToMM = 0.3528 // 1 pt in mm

// pdfBase holds shared state and helpers for all PDF-based renderers.
type pdfBase struct {
	cfg            render.Config
	pdf            *fpdf.Fpdf
	w              io.Writer
	pageW          float64
	pageH          float64
	marginL        float64
	marginR        float64
	marginT        float64
	marginB        float64
	bodyW          float64 // pageW - marginL - marginR
	fontStyle      string  // tracks current accumulated style
	dirDepth       int     // nesting depth of InlineDirectionNodes
	hasTitlePage   bool    // whether a title page was rendered
	hasBody        bool    // whether the document has body content after front matter
	lineHeight     float64 // vertical line spacing in mm
	titlePageTitle string

	// Body block adjacency tracking
	prevWasStageDirection bool
	prevWasCallout        bool

	// Dual dialogue state
	inDualDialogue bool    // true when rendering inside a DualDialogue node
	dualSequential bool    // true when a DualDialogue falls back to normal sequential rendering
	dualSide       int     // 0 = left, 1 = right
	dualStartY     float64 // Y position at start of dual dialogue
	dualMidY       float64 // Y after left column, to compute max height
}

func (b *pdfBase) initPDF(fontLoader func(*fpdf.Fpdf), defaultFamily string) {
	size := string(b.cfg.PageSize)
	if b.cfg.PageSize == render.PageLetter {
		size = "Letter"
	}

	b.pdf = fpdf.New("P", "mm", size, "")

	b.marginL = b.cfg.MarginLeft * pointsToMM
	b.marginR = b.cfg.MarginRight * pointsToMM
	b.marginT = b.cfg.MarginTop * pointsToMM
	b.marginB = b.cfg.MarginBottom * pointsToMM

	b.pdf.SetMargins(b.marginL, b.marginT, b.marginR)
	b.pdf.SetAutoPageBreak(true, b.marginB)

	b.pageW, b.pageH = b.pdf.GetPageSize()
	b.bodyW = b.pageW - b.marginL - b.marginR

	if b.cfg.FontPath != "" {
		if loadCustomFont(b.pdf, "CustomFont", b.cfg.FontPath) {
			b.cfg.FontFamily = "CustomFont"
		} else {
			slog.Warn("failed to load custom font, falling back to default", "path", b.cfg.FontPath)
			fontLoader(b.pdf)
			b.cfg.FontFamily = defaultFamily
		}
	} else {
		fontLoader(b.pdf)
		b.cfg.FontFamily = defaultFamily
	}
	b.pdf.SetFont(b.cfg.FontFamily, "", b.cfg.FontSize)

	// Page numbers
	b.pdf.AliasNbPages("")
	b.pdf.SetFooterFunc(func() {
		b.pdf.SetY(-b.marginB + 5)
		b.pdf.SetFont(b.cfg.FontFamily, "", b.cfg.FontSize-2)
		b.pdf.CellFormat(0, 10, fmt.Sprintf("%d", b.pdf.PageNo()),
			"", 0, "C", false, 0, "")
		b.pdf.SetFont(b.cfg.FontFamily, "", b.cfg.FontSize)
	})

	b.pdf.AddPage()
	b.fontStyle = ""
}

// --- Inline rendering (shared by all PDF renderers) ---

func (b *pdfBase) RenderText(t *ast.TextNode) error {
	b.pdf.Write(b.lineHeight, t.Value)
	return nil
}

func (b *pdfBase) BeginBold(_ *ast.BoldNode) error {
	b.pushStyle("B")
	return nil
}

func (b *pdfBase) EndBold(_ *ast.BoldNode) error {
	b.popStyle("B")
	return nil
}

func (b *pdfBase) BeginItalic(_ *ast.ItalicNode) error {
	b.pushStyle("I")
	return nil
}

func (b *pdfBase) EndItalic(_ *ast.ItalicNode) error {
	b.popStyle("I")
	return nil
}

func (b *pdfBase) BeginBoldItalic(_ *ast.BoldItalicNode) error {
	b.pushStyle("BI")
	return nil
}

func (b *pdfBase) EndBoldItalic(_ *ast.BoldItalicNode) error {
	b.popStyle("BI")
	return nil
}

func (b *pdfBase) BeginUnderline(_ *ast.UnderlineNode) error {
	b.pushStyle("U")
	return nil
}

func (b *pdfBase) EndUnderline(_ *ast.UnderlineNode) error {
	b.popStyle("U")
	return nil
}

func (b *pdfBase) BeginStrikethrough(_ *ast.StrikethroughNode) error {
	b.pushStyle("S")
	return nil
}

func (b *pdfBase) EndStrikethrough(_ *ast.StrikethroughNode) error {
	b.popStyle("S")
	return nil
}

func (b *pdfBase) BeginInlineDirection(_ *ast.InlineDirectionNode) error {
	b.pushStyle("I")
	b.dirDepth++
	if b.dirDepth == 1 {
		b.pdf.Write(b.lineHeight, "(")
	}
	return nil
}

func (b *pdfBase) EndInlineDirection(_ *ast.InlineDirectionNode) error {
	b.dirDepth--
	if b.dirDepth == 0 {
		b.pdf.Write(b.lineHeight, ")")
	}
	b.popStyle("I")
	return nil
}

func (b *pdfBase) RenderPageBreak(_ *ast.PageBreak) error {
	b.pdf.AddPage()
	b.prevWasStageDirection = false
	b.prevWasCallout = false
	return nil
}

func (b *pdfBase) resetBodyBlockState() {
	b.prevWasStageDirection = false
	b.prevWasCallout = false
}

func (b *pdfBase) RenderComment(_ *ast.Comment) error {
	return nil
}

// --- Style helpers ---

func (b *pdfBase) setStyle(style string) {
	if b.fontStyle != style {
		b.fontStyle = style
		b.pdf.SetFont(b.cfg.FontFamily, style, b.cfg.FontSize)
	}
}

func (b *pdfBase) pushStyle(add string) {
	merged := mergeStyles(b.fontStyle, add)
	b.setStyle(merged)
}

func (b *pdfBase) popStyle(remove string) {
	result := removeStyles(b.fontStyle, remove)
	b.setStyle(result)
}

func (b *pdfBase) ensureSpace(mm float64) {
	if b.pdf.GetY()+mm > b.pageH-b.marginB {
		b.pdf.AddPage()
	}
}

func (b *pdfBase) centeredText(text string) {
	b.pdf.CellFormat(b.bodyW, b.lineHeight, text, "", 1, "C", false, 0, "")
}

func (b *pdfBase) remainingPageHeight() float64 {
	return b.pageH - b.marginB - b.pdf.GetY()
}

func (b *pdfBase) usablePageHeight() float64 {
	return b.pageH - b.marginT - b.marginB
}

// newCustomSizePDF creates an fpdf instance with a custom page size in mm.
func newCustomSizePDF(widthMM, heightMM float64) *fpdf.Fpdf {
	return fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		Size:           fpdf.SizeType{Wd: widthMM, Ht: heightMM},
	})
}

// mergeStyles combines two fpdf style strings (e.g. "B" + "I" = "BI").
func mergeStyles(a, b string) string {
	flags := make(map[byte]bool)
	for i := range len(a) {
		flags[a[i]] = true
	}
	for i := range len(b) {
		flags[b[i]] = true
	}
	var out strings.Builder
	for _, ch := range []byte{'B', 'I', 'U', 'S'} {
		if flags[ch] {
			out.WriteByte(ch)
		}
	}
	return out.String()
}

// removeStyles removes specific style flags from a style string.
func removeStyles(style, remove string) string {
	removeSet := make(map[byte]bool)
	for i := range len(remove) {
		removeSet[remove[i]] = true
	}
	var out strings.Builder
	for i := range len(style) {
		if !removeSet[style[i]] {
			out.WriteByte(style[i])
		}
	}
	return out.String()
}
