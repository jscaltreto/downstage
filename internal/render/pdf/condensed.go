package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

const condensedLineHeight = 4.5 // mm
const halfInchPt = 36.0         // 0.5 inch in points
const condensedSmallGapFactor = 0.25

// Half-letter page size: 5.5" x 8.5"
const (
	halfLetterW = 139.7 // mm
	halfLetterH = 215.9 // mm
)

var _ render.NodeRenderer = (*condensedRenderer)(nil)
var _ dialoguePaginationStrategy = (*condensedRenderer)(nil)

// NewCondensedRenderer creates an acting-edition PDF NodeRenderer.
// Uses half-letter page, Libre Baskerville serif font, and compact layout
// with character name + dialogue on the same line.
func NewCondensedRenderer(cfg render.Config) render.NodeRenderer {
	// Override config for acting edition defaults
	cfg.FontSize = 10
	cfg.MarginTop = halfInchPt
	cfg.MarginBottom = halfInchPt
	cfg.MarginLeft = halfInchPt
	cfg.MarginRight = halfInchPt
	return &condensedRenderer{
		pdfBase: pdfBase{cfg: cfg, lineHeight: condensedLineHeight},
	}
}

type condensedRenderer struct {
	pdfBase
	dualDialogueInlineFirstLine bool
	activeDialogue              *bufferedDialogue
}

func (r *condensedRenderer) condensedSmallGap() float64 {
	return r.lineHeight * condensedSmallGapFactor
}

// --- Lifecycle ---

func (r *condensedRenderer) BeginDocument(doc *ast.Document, w io.Writer) error {
	r.w = w
	tp := render.DocumentTitlePage(doc)
	r.hasTitlePage = tp != nil
	r.hasBody = render.DocumentHasRenderableBody(doc)
	r.titlePageTitle = titlePageTitle(tp)
	r.initCondensedPDF()
	r.inlinePlaySections = make(map[*ast.Section]bool)
	for _, section := range render.PlayableTopLevelSections(doc) {
		if render.IsInlinePlaySection(doc, section) {
			r.inlinePlaySections[section] = true
		}
	}
	return nil
}

func (r *condensedRenderer) EndDocument(_ *ast.Document) error {
	return r.pdf.Output(r.w)
}

func (r *condensedRenderer) initCondensedPDF() {
	r.pdf = newCustomSizePDF(halfLetterW, halfLetterH)

	r.marginL = r.cfg.MarginLeft * pointsToMM
	r.marginR = r.cfg.MarginRight * pointsToMM
	r.marginT = r.cfg.MarginTop * pointsToMM
	r.marginB = r.cfg.MarginBottom * pointsToMM

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
		r.renderPageNumberFooter(fmt.Sprintf("%d", r.pdf.PageNo()), 8)
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
	})

	r.pdf.AddPage()
	r.fontStyle = ""
}

// --- Front matter ---

func (r *condensedRenderer) RenderTitlePage(tp *ast.TitlePage) error {
	r.beginTitlePage()

	title, subtitle, authors, other := partitionTitlePageEntries(tp)

	r.hasTitlePage = true
	r.titlePageTitle = title

	titleY := r.pageH * 0.30

	if title != "" {
		r.pdf.SetY(titleY)
		r.pdf.SetFont(r.cfg.FontFamily, "B", r.cfg.FontSize+6)
		r.centeredWrappedText(strings.ToUpper(title), r.lineHeight)
		r.pdf.Ln(r.lineHeight)
	}

	if subtitle != "" {
		r.pdf.SetFont(r.cfg.FontFamily, "I", r.cfg.FontSize+1)
		r.centeredWrappedText(subtitle, r.lineHeight)
		r.pdf.Ln(r.lineHeight)
	}

	if len(authors) > 0 {
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize+1)
		r.pdf.Ln(r.lineHeight * 2)
		r.centeredWrappedText("by", r.lineHeight)
		r.pdf.Ln(r.lineHeight)
		for _, author := range authors {
			r.centeredWrappedText(author, r.lineHeight)
			r.pdf.Ln(r.lineHeight)
		}
	}

	if len(other) > 0 {
		r.pdf.SetY(r.pageH * 0.70)
		r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize-1)
		for _, kv := range other {
			r.centeredWrappedText(kv.Key+": "+kv.Value, r.lineHeight)
		}
	}

	r.pdf.SetFont(r.cfg.FontFamily, "", r.cfg.FontSize)
	r.fontStyle = ""
	r.finishTitlePage()
	return nil
}

// --- Structural ---

func (r *condensedRenderer) BeginSection(s *ast.Section) error {
	r.resetBodyBlockState()
	switch s.Kind {
	case ast.SectionAct:
		return r.beginAct(s)
	case ast.SectionScene:
		return r.beginScene(s)
	case ast.SectionDramatisPersonae:
		if r.activeTopLevelSection != nil && r.inlinePlaySections[r.activeTopLevelSection] {
			renderDramatisPersonae(&r.pdfBase, s, 0)
			return nil
		}
		if !r.consumePendingTitlePageBodyPage() {
			r.consumePendingDramatisBodyPage()
		}
		renderDramatisPersonae(&r.pdfBase, s, 0)
		r.finishDramatisPersonaePage()
		return nil
	default: // SectionGeneric
		if render.IsLegacyTopLevelDramatisPersonae(s) {
			return nil
		}
		if s.Level == 1 {
			r.activeTopLevelSection = s
		}
		if r.hasTitlePage && s.Level == 1 && strings.EqualFold(strings.TrimSpace(s.Title), r.titlePageTitle) {
			return nil
		}
		if s.Level == 1 && r.inlinePlaySections[s] {
			if !r.consumePendingTitlePageBodyPage() && !r.consumePendingDramatisBodyPage() && !r.isFreshInitialPage() {
				r.pdf.AddPage()
			}
			renderInlinePlayHeader(&r.pdfBase, s, r.cfg.FontSize+6, r.cfg.FontSize-1)
			r.beginInlinePlaySection()
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
		if !r.consumePendingDramatisBodyPage() && !r.consumePendingInlinePlayFirstBodyPage() {
			r.pdf.AddPage()
		}
		if s.Title != "" {
			r.setStyle("B")
			r.centeredText(strings.ToUpper(s.Title))
			r.setStyle("")
			r.pdf.Ln(r.lineHeight * 2)
		}
		return nil
	}
}

func (r *condensedRenderer) EndSection(s *ast.Section) error {
	r.resetBodyBlockState()
	if s.Level == 1 {
		r.activeTopLevelSection = nil
		r.pendingInlinePlayFirstBodyPage = false
	}
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
		if !r.consumePendingTitlePageBodyPage() && !r.consumePendingDramatisBodyPage() && !r.consumePendingInlinePlayFirstBodyPage() {
			r.pdf.AddPage()
		}
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
	if r.consumePendingTitlePageBodyPage() || r.consumePendingDramatisBodyPage() || r.consumePendingInlinePlayFirstBodyPage() {
		// Fresh body page after a title page does not need extra leading space.
	} else {
		r.ensureSpace(r.lineHeight * 3)
		r.pdf.Ln(r.lineHeight)
	}

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

// --- Dual Dialogue ---

func (r *condensedRenderer) BeginDualDialogue(d *ast.DualDialogue) error {
	r.resetBodyBlockState()
	r.dualSequential = false
	r.dualMidY = 0
	estimatedHeight := r.estimateDualDialogueHeight(d)
	if estimatedHeight > r.usablePageHeight() {
		return nil
	}
	if estimatedHeight > r.remainingPageHeight() {
		r.pdf.AddPage()
	}
	r.inDualDialogue = true
	r.dualSide = 0
	r.dualStartY = r.pdf.GetY()
	return nil
}

func (r *condensedRenderer) EndDualDialogue(_ *ast.DualDialogue) error {
	if !r.inDualDialogue {
		r.dualSequential = false
		return nil
	}

	endY := r.pdf.GetY()
	if r.dualMidY > endY {
		endY = r.dualMidY
	}
	r.pdf.SetY(endY)
	r.pdf.SetLeftMargin(r.marginL)
	r.pdf.SetRightMargin(r.marginR)
	r.inDualDialogue = false
	r.dualSequential = false
	return nil
}

// --- Dialogue ---
// Acting edition: "HAMLET. (aside) A piece of work is man."
// Character name bold, parenthetical italic, dialogue regular — all on one line.

func (r *condensedRenderer) BeginDialogue(d *ast.Dialogue) error {
	r.resetBodyBlockState()
	if r.inDualDialogue {
		return r.beginDualDialogueSide(d)
	}
	r.activeDialogue = &bufferedDialogue{
		character:            d.Character,
		parenthetical:        d.Parenthetical,
		parentheticalInlines: dialogueParentheticalInlines(d),
		lines:                make([]bufferedDialogueLine, 0, len(d.Lines)),
	}
	return nil
}

func (r *condensedRenderer) beginDualDialogueSide(d *ast.Dialogue) error {
	halfW := r.bodyW / 2
	gap := 3.0 // mm gap between columns
	colW := halfW - gap/2

	var leftM, rightM float64
	if r.dualSide == 0 {
		leftM = r.marginL
		rightM = r.pageW - r.marginL - colW
		r.pdf.Ln(r.condensedSmallGap())
	} else {
		r.dualMidY = r.pdf.GetY()
		r.pdf.SetY(r.dualStartY)
		leftM = r.marginL + halfW + gap/2
		rightM = r.marginR
		r.pdf.Ln(r.condensedSmallGap())
	}

	r.pdf.SetLeftMargin(leftM)
	r.pdf.SetRightMargin(rightM)

	r.dualDialogueInlineFirstLine = true

	// Character name — bold, inline
	r.pdf.SetX(leftM)
	r.setStyle("B")
	r.pdf.Write(r.lineHeight, strings.ToUpper(d.Character)+".")
	r.setStyle("")
	r.pdf.Write(r.lineHeight, "  ")

	// Parenthetical
	if d.Parenthetical != "" {
		parenInlines := dialogueParentheticalInlines(d)
		r.setStyle("I")
		r.pdf.Write(r.lineHeight, "(")
		if err := r.renderInlineContent(parenInlines); err != nil {
			return err
		}
		r.pdf.Write(r.lineHeight, ")")
		r.setStyle("")
		r.pdf.Write(r.lineHeight, " ")
	}

	return nil
}

func (r *condensedRenderer) estimateDualDialogueHeight(d *ast.DualDialogue) float64 {
	halfW := r.bodyW / 2
	gap := 3.0
	colW := halfW - gap/2

	leftHeight := r.estimateDialogueHeight(d.Left, colW)
	rightHeight := r.estimateDialogueHeight(d.Right, colW)
	if rightHeight > leftHeight {
		return rightHeight
	}
	return leftHeight
}

func (r *condensedRenderer) estimateDialogueHeight(d *ast.Dialogue, width float64) float64 {
	if d == nil {
		return 0
	}

	height := r.condensedSmallGap()
	firstLineWidth := width - r.pdf.GetStringWidth(strings.ToUpper(d.Character)+".  ")
	if d.Parenthetical != "" {
		firstLineWidth -= r.pdf.GetStringWidth(parentheticalPlainText(d.Parenthetical, dialogueParentheticalInlines(d)) + " ")
	}
	if firstLineWidth < 10 {
		firstLineWidth = 10
	}

	if len(d.Lines) == 0 {
		return height + r.lineHeight
	}

	for i, line := range d.Lines {
		if len(line.Content) == 0 {
			height += r.condensedSmallGap()
			continue
		}
		lineWidth := width
		if line.IsVerse {
			lineWidth -= 10
		}
		if i == 0 {
			lineWidth = firstLineWidth
		}
		if lineWidth < 10 {
			lineWidth = 10
		}

		text := render.PlainText(line.Content)
		wrapped := r.pdf.SplitText(text, lineWidth)
		if len(wrapped) == 0 {
			height += r.lineHeight
			continue
		}
		height += float64(len(wrapped)) * r.lineHeight
	}

	return height
}

func (r *condensedRenderer) EndDialogue(_ *ast.Dialogue) error {
	if r.inDualDialogue {
		r.dualSide++
		return nil
	}
	if r.activeDialogue != nil {
		dialogue := *r.activeDialogue
		r.activeDialogue = nil
		if err := r.renderBufferedDialogue(dialogue); err != nil {
			return err
		}
	}
	r.dualDialogueInlineFirstLine = false
	return nil
}

func (r *condensedRenderer) BeginDialogueLine(line *ast.DialogueLine) error {
	if !r.inDualDialogue {
		r.beginCapturedDialogueLine()
		return nil
	}
	r.captureDialogueLine = false

	if len(line.Content) == 0 {
		r.dualDialogueInlineFirstLine = false
		return nil
	}
	if r.dualDialogueInlineFirstLine {
		// First line continues after character name on the same line
		r.dualDialogueInlineFirstLine = false
		if line.IsVerse {
			// Even verse on the first line starts inline
		}
	} else {
		leftM, _, _, _ := r.pdf.GetMargins()
		// Subsequent lines start at left margin
		if line.IsVerse {
			r.pdf.SetX(leftM + 10) // verse indent
		} else {
			r.pdf.SetX(leftM)
		}
	}
	return nil
}

func (r *condensedRenderer) EndDialogueLine(line *ast.DialogueLine) error {
	if !r.inDualDialogue {
		runs := r.endCapturedDialogueLine()
		if r.activeDialogue == nil {
			return nil
		}
		r.activeDialogue.lines = append(r.activeDialogue.lines, bufferedDialogueLine{
			runs:    runs,
			isVerse: line.IsVerse,
		})
		return nil
	}

	if len(line.Content) == 0 {
		r.pdf.Ln(r.condensedSmallGap())
		return nil
	}
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Stage Direction ---
// Italic, indented 0.5" further.

func (r *condensedRenderer) BeginStageDirection(sd *ast.StageDirection) error {
	switch {
	case sd.Continuation:
		// Adjacent line — regular line break, no extra gap
	case r.prevWasStageDirection:
		// Separated by blank lines — small paragraph break
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.condensedSmallGap())
	default:
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.condensedSmallGap())
	}

	stageIndent := halfInchPt * pointsToMM
	r.setStyle("I")
	r.pdf.SetLeftMargin(r.marginL + stageIndent)
	r.pdf.SetX(r.marginL + stageIndent)
	return nil
}

func (r *condensedRenderer) EndStageDirection(_ *ast.StageDirection) error {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")
	r.pdf.SetLeftMargin(r.marginL)
	r.prevWasStageDirection = true
	r.prevWasCallout = false
	return nil
}

// --- Callout ---

func (r *condensedRenderer) BeginCallout(c *ast.Callout) error {
	switch {
	case c.Continuation:
	case r.prevWasCallout:
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.condensedSmallGap())
	default:
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.condensedSmallGap())
	}

	calloutIndent := 2 * halfInchPt * pointsToMM
	r.setStyle("B")
	r.pdf.SetLeftMargin(r.marginL + calloutIndent)
	r.pdf.SetX(r.marginL + calloutIndent)
	return nil
}

func (r *condensedRenderer) EndCallout(_ *ast.Callout) error {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")
	r.pdf.SetLeftMargin(r.marginL)
	r.prevWasStageDirection = false
	r.prevWasCallout = true
	return nil
}

// --- Song ---

func (r *condensedRenderer) BeginSong(song *ast.Song) error {
	r.resetBodyBlockState()
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
	r.resetBodyBlockState()
	r.pdf.Ln(r.lineHeight / 2)
	r.setStyle("B")
	r.centeredText("SONG END")
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Verse Block ---

func (r *condensedRenderer) BeginVerseBlock(_ *ast.VerseBlock) error {
	r.resetBodyBlockState()
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
