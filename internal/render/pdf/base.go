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
	cfg                            render.Config
	pdf                            *fpdf.Fpdf
	w                              io.Writer
	pageW                          float64
	pageH                          float64
	marginL                        float64
	marginR                        float64
	marginT                        float64
	marginB                        float64
	bodyW                          float64 // pageW - marginL - marginR
	fontStyle                      string  // tracks current accumulated style
	styleStack                     []string
	dirDepth                       int     // nesting depth of InlineDirectionNodes
	hasTitlePage                   bool    // whether a title page was rendered
	hasBody                        bool    // whether the document has body content after front matter
	lineHeight                     float64 // vertical line spacing in mm
	titlePageTitle                 string
	titlePagesSeen                 int
	titlePagePages                 map[int]bool
	pendingTitlePageBodyPage       bool
	pendingDramatisBodyPage        bool
	pendingInlinePlayFirstBodyPage bool
	inlinePlaySections             map[*ast.Section]bool
	activeTopLevelSection          *ast.Section

	// Body block adjacency tracking
	prevWasStageDirection bool
	prevWasCallout        bool

	// Dual dialogue state
	inDualDialogue bool    // true when rendering inside a DualDialogue node
	dualSequential bool    // true when a DualDialogue falls back to normal sequential rendering
	dualSide       int     // 0 = left, 1 = right
	dualStartY     float64 // Y position at start of dual dialogue
	dualMidY       float64 // Y after left column, to compute max height

	// Buffered dialogue inline capture for custom pagination.
	captureDialogueLine bool
	captureStyle        string
	captureStyleStack   []string
	captureDirDepth     int
	capturedRuns        []dialogueTextRun
}

type dialogueTextRun struct {
	text  string
	style string
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

	// Page numbers. Title pages still count toward the page total but
	// don't show a number themselves.
	b.pdf.AliasNbPages("")
	b.titlePagePages = make(map[int]bool)
	b.installPageNumberFooter(5, 10)

	b.pdf.AddPage()
	b.fontStyle = ""
	b.styleStack = b.styleStack[:0]
	b.titlePagesSeen = 0
	b.pendingTitlePageBodyPage = false
	b.pendingDramatisBodyPage = false
	b.pendingInlinePlayFirstBodyPage = false
	b.inlinePlaySections = nil
	b.activeTopLevelSection = nil
}

func (b *pdfBase) installPageNumberFooter(offset, height float64) {
	b.pdf.SetFooterFunc(func() {
		if b.titlePagePages[b.pdf.PageNo()] {
			return
		}
		b.pdf.SetY(-b.marginB + offset)
		b.pdf.SetFont(b.cfg.FontFamily, "", b.cfg.FontSize-2)
		b.renderPageNumberFooter(fmt.Sprintf("%d", b.pdf.PageNo()), height)
		b.pdf.SetFont(b.cfg.FontFamily, "", b.cfg.FontSize)
	})
}

func (b *pdfBase) beginTitlePage() {
	if b.titlePagesSeen > 0 && !b.pendingTitlePageBodyPage {
		b.pdf.AddPage()
	}
	b.pendingTitlePageBodyPage = false
	b.titlePagesSeen++
	if b.titlePagePages != nil {
		b.titlePagePages[b.pdf.PageNo()] = true
	}
}

func (b *pdfBase) finishTitlePage() {
	if !b.hasBody {
		return
	}
	b.pdf.AddPage()
	b.pendingTitlePageBodyPage = true
}

func (b *pdfBase) consumePendingTitlePageBodyPage() bool {
	if !b.pendingTitlePageBodyPage {
		return false
	}
	b.pendingTitlePageBodyPage = false
	return true
}

func (b *pdfBase) finishDramatisPersonaePage() {
	b.pendingDramatisBodyPage = true
}

func (b *pdfBase) consumePendingDramatisBodyPage() bool {
	if !b.pendingDramatisBodyPage {
		return false
	}
	b.pendingDramatisBodyPage = false
	b.pdf.AddPage()
	return true
}

func (b *pdfBase) beginInlinePlaySection() {
	b.pendingInlinePlayFirstBodyPage = true
}

func (b *pdfBase) consumePendingInlinePlayFirstBodyPage() bool {
	if !b.pendingInlinePlayFirstBodyPage {
		return false
	}
	b.pendingInlinePlayFirstBodyPage = false
	b.pdf.AddPage()
	return true
}

func (b *pdfBase) isFreshInitialPage() bool {
	return b.pdf != nil && b.pdf.PageNo() == 1 && b.titlePagesSeen == 0 && b.pdf.GetY() == b.marginT
}

// --- Inline rendering (shared by all PDF renderers) ---

func (b *pdfBase) RenderText(t *ast.TextNode) error {
	if b.captureDialogueLine {
		b.appendCapturedText(t.Value)
		return nil
	}
	b.pdf.Write(b.lineHeight, t.Value)
	return nil
}

func (b *pdfBase) BeginBold(_ *ast.BoldNode) error {
	if b.captureDialogueLine {
		b.pushCaptureStyle("B")
		return nil
	}
	b.pushStyle("B")
	return nil
}

func (b *pdfBase) EndBold(_ *ast.BoldNode) error {
	if b.captureDialogueLine {
		b.popCaptureStyle()
		return nil
	}
	b.popStyle()
	return nil
}

func (b *pdfBase) BeginItalic(_ *ast.ItalicNode) error {
	if b.captureDialogueLine {
		b.pushCaptureStyle("I")
		return nil
	}
	b.pushStyle("I")
	return nil
}

func (b *pdfBase) EndItalic(_ *ast.ItalicNode) error {
	if b.captureDialogueLine {
		b.popCaptureStyle()
		return nil
	}
	b.popStyle()
	return nil
}

func (b *pdfBase) BeginBoldItalic(_ *ast.BoldItalicNode) error {
	if b.captureDialogueLine {
		b.pushCaptureStyle("BI")
		return nil
	}
	b.pushStyle("BI")
	return nil
}

func (b *pdfBase) EndBoldItalic(_ *ast.BoldItalicNode) error {
	if b.captureDialogueLine {
		b.popCaptureStyle()
		return nil
	}
	b.popStyle()
	return nil
}

func (b *pdfBase) BeginUnderline(_ *ast.UnderlineNode) error {
	if b.captureDialogueLine {
		b.pushCaptureStyle("U")
		return nil
	}
	b.pushStyle("U")
	return nil
}

func (b *pdfBase) EndUnderline(_ *ast.UnderlineNode) error {
	if b.captureDialogueLine {
		b.popCaptureStyle()
		return nil
	}
	b.popStyle()
	return nil
}

func (b *pdfBase) BeginStrikethrough(_ *ast.StrikethroughNode) error {
	if b.captureDialogueLine {
		b.pushCaptureStyle("S")
		return nil
	}
	b.pushStyle("S")
	return nil
}

func (b *pdfBase) EndStrikethrough(_ *ast.StrikethroughNode) error {
	if b.captureDialogueLine {
		b.popCaptureStyle()
		return nil
	}
	b.popStyle()
	return nil
}

func (b *pdfBase) BeginInlineDirection(_ *ast.InlineDirectionNode) error {
	if b.captureDialogueLine {
		b.pushCaptureStyle("I")
		b.captureDirDepth++
		if b.captureDirDepth == 1 {
			b.appendCapturedText("(")
		}
		return nil
	}
	b.pushStyle("I")
	b.dirDepth++
	if b.dirDepth == 1 {
		b.pdf.Write(b.lineHeight, "(")
	}
	return nil
}

func (b *pdfBase) EndInlineDirection(_ *ast.InlineDirectionNode) error {
	if b.captureDialogueLine {
		b.captureDirDepth--
		if b.captureDirDepth == 0 {
			b.appendCapturedText(")")
		}
		b.popCaptureStyle()
		return nil
	}
	b.dirDepth--
	if b.dirDepth == 0 {
		b.pdf.Write(b.lineHeight, ")")
	}
	b.popStyle()
	return nil
}

func (b *pdfBase) beginCapturedDialogueLine() {
	b.captureDialogueLine = true
	b.captureStyle = ""
	b.captureStyleStack = b.captureStyleStack[:0]
	b.captureDirDepth = 0
	b.capturedRuns = b.capturedRuns[:0]
}

func (b *pdfBase) endCapturedDialogueLine() []dialogueTextRun {
	runs := append([]dialogueTextRun(nil), b.capturedRuns...)
	b.captureDialogueLine = false
	b.captureStyle = ""
	b.captureStyleStack = b.captureStyleStack[:0]
	b.captureDirDepth = 0
	b.capturedRuns = b.capturedRuns[:0]
	return runs
}

func (b *pdfBase) appendCapturedText(text string) {
	if text == "" {
		return
	}
	if n := len(b.capturedRuns); n > 0 && b.capturedRuns[n-1].style == b.captureStyle {
		b.capturedRuns[n-1].text += text
		return
	}
	b.capturedRuns = append(b.capturedRuns, dialogueTextRun{text: text, style: b.captureStyle})
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
	b.styleStack = append(b.styleStack, b.fontStyle)
	merged := mergeStyles(b.fontStyle, add)
	b.setStyle(merged)
}

func (b *pdfBase) popStyle() {
	if len(b.styleStack) == 0 {
		b.setStyle("")
		return
	}
	prev := b.styleStack[len(b.styleStack)-1]
	b.styleStack = b.styleStack[:len(b.styleStack)-1]
	b.setStyle(prev)
}

func (b *pdfBase) pushCaptureStyle(add string) {
	b.captureStyleStack = append(b.captureStyleStack, b.captureStyle)
	b.captureStyle = mergeStyles(b.captureStyle, add)
}

func (b *pdfBase) popCaptureStyle() {
	if len(b.captureStyleStack) == 0 {
		b.captureStyle = ""
		return
	}
	prev := b.captureStyleStack[len(b.captureStyleStack)-1]
	b.captureStyleStack = b.captureStyleStack[:len(b.captureStyleStack)-1]
	b.captureStyle = prev
}

func (b *pdfBase) ensureSpace(mm float64) {
	if b.pdf.GetY()+mm > b.pageH-b.marginB {
		b.pdf.AddPage()
	}
}

func (b *pdfBase) centeredText(text string) {
	b.pdf.CellFormat(b.bodyW, b.lineHeight, text, "", 1, "C", false, 0, "")
}

func (b *pdfBase) centeredWrappedText(text string, lineHeight float64) {
	b.pdf.SetX(b.marginL)
	b.pdf.MultiCell(b.bodyW, lineHeight, text, "", "C", false)
}

func (b *pdfBase) renderPageNumberFooter(text string, height float64) {
	width := b.pdf.GetStringWidth(text)
	b.pdf.SetX((b.pageW - width) / 2)
	b.pdf.CellFormat(width, height, text, "", 0, "", false, 0, "")
}

func (b *pdfBase) centeredInlines(inlines []ast.Inline, prefix, suffix string) error {
	text := prefix + render.PlainText(inlines) + suffix
	width := b.pdf.GetStringWidth(text)
	b.pdf.SetX(b.marginL + (b.bodyW-width)/2)
	if prefix != "" {
		b.pdf.Write(b.lineHeight, prefix)
	}
	if err := b.renderInlineContent(inlines); err != nil {
		return err
	}
	if suffix != "" {
		b.pdf.Write(b.lineHeight, suffix)
	}
	b.pdf.Ln(b.lineHeight)
	return nil
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

func (b *pdfBase) renderInlineContent(inlines []ast.Inline) error {
	for _, inline := range inlines {
		switch n := inline.(type) {
		case *ast.TextNode:
			if err := b.RenderText(n); err != nil {
				return err
			}
		case *ast.BoldNode:
			if err := b.BeginBold(n); err != nil {
				return err
			}
			if err := b.renderInlineContent(n.Content); err != nil {
				return err
			}
			if err := b.EndBold(n); err != nil {
				return err
			}
		case *ast.ItalicNode:
			if err := b.BeginItalic(n); err != nil {
				return err
			}
			if err := b.renderInlineContent(n.Content); err != nil {
				return err
			}
			if err := b.EndItalic(n); err != nil {
				return err
			}
		case *ast.BoldItalicNode:
			if err := b.BeginBoldItalic(n); err != nil {
				return err
			}
			if err := b.renderInlineContent(n.Content); err != nil {
				return err
			}
			if err := b.EndBoldItalic(n); err != nil {
				return err
			}
		case *ast.UnderlineNode:
			if err := b.BeginUnderline(n); err != nil {
				return err
			}
			if err := b.renderInlineContent(n.Content); err != nil {
				return err
			}
			if err := b.EndUnderline(n); err != nil {
				return err
			}
		case *ast.StrikethroughNode:
			if err := b.BeginStrikethrough(n); err != nil {
				return err
			}
			if err := b.renderInlineContent(n.Content); err != nil {
				return err
			}
			if err := b.EndStrikethrough(n); err != nil {
				return err
			}
		case *ast.InlineDirectionNode:
			if err := b.BeginInlineDirection(n); err != nil {
				return err
			}
			if err := b.renderInlineContent(n.Content); err != nil {
				return err
			}
			if err := b.EndInlineDirection(n); err != nil {
				return err
			}
		}
	}
	return nil
}

func dialogueParentheticalInlines(d *ast.Dialogue) []ast.Inline {
	return parentheticalInlineContent(d.Parenthetical, d.ParentheticalInlines())
}

func parentheticalInlineContent(parenthetical string, inlines []ast.Inline) []ast.Inline {
	if len(inlines) > 0 {
		return inlines
	}
	paren := strings.TrimSpace(parenthetical)
	if strings.HasPrefix(paren, "(") && strings.HasSuffix(paren, ")") {
		paren = strings.TrimSuffix(strings.TrimPrefix(paren, "("), ")")
	}
	if paren == "" {
		return nil
	}
	return []ast.Inline{&ast.TextNode{Value: paren}}
}

func parentheticalPlainText(parenthetical string, inlines []ast.Inline) string {
	if len(inlines) > 0 {
		return "(" + render.PlainText(inlines) + ")"
	}
	paren := parenthetical
	if paren == "" {
		return ""
	}
	if paren[0] != '(' {
		paren = "(" + paren + ")"
	}
	return paren
}
