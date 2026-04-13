package pdf

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

// --- Section ---

func (r *pdfRenderer) BeginSection(s *ast.Section) error {
	r.resetBodyBlockState()
	switch s.Kind {
	case ast.SectionAct:
		return r.beginAct(s)
	case ast.SectionScene:
		return r.beginScene(s)
	case ast.SectionDramatisPersonae:
		if r.activeTopLevelSection != nil && r.inlinePlaySections[r.activeTopLevelSection] {
			renderDramatisPersonae(&r.pdfBase, s, r.bodyW*0.15)
			return nil
		}
		if !r.consumePendingTitlePageBodyPage() {
			r.consumePendingDramatisBodyPage()
		}
		renderDramatisPersonae(&r.pdfBase, s, r.bodyW*0.15)
		r.finishDramatisPersonaePage()
		return nil
	default: // SectionGeneric
		if render.IsLegacyTopLevelDramatisPersonae(s) {
			return nil
		}
		if s.Level == 1 {
			r.activeTopLevelSection = s
		}
		// Skip play title heading if title page already rendered it
		if r.hasTitlePage && s.Level == 1 && strings.EqualFold(strings.TrimSpace(render.SectionDisplayTitle(s)), r.titlePageTitle) {
			return nil
		}
		if s.Level == 1 && r.inlinePlaySections[s] {
			if !r.consumePendingTitlePageBodyPage() && !r.consumePendingDramatisBodyPage() && !r.isFreshInitialPage() {
				r.pdf.AddPage()
			}
			renderInlinePlayHeader(&r.pdfBase, s, r.cfg.FontSize+8, r.cfg.FontSize)
			r.beginInlinePlaySection()
			return nil
		}
		if s.Level == 0 {
			// Forced heading: inline, no page break
			r.ensureSpace(r.lineHeight * 3)
			r.pdf.Ln(r.lineHeight)
			r.setStyle("B")
			r.centeredText(s.Title)
			r.setStyle("")
			r.pdf.Ln(r.lineHeight)
			return nil
		}
		if !r.consumePendingTitlePageBodyPage() && !r.consumePendingDramatisBodyPage() {
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

func (r *pdfRenderer) EndSection(s *ast.Section) error {
	r.resetBodyBlockState()
	if s.Level == 1 {
		r.activeTopLevelSection = nil
		r.pendingInlinePlayFirstBodyPage = false
	}
	return nil
}

func (r *pdfRenderer) BeginSectionLine(sl *ast.SectionLine) error {
	if len(sl.Content) == 0 {
		// Blank line = paragraph break
		r.pdf.Ln(r.lineHeight * 2)
	}
	return nil
}

func (r *pdfRenderer) EndSectionLine(sl *ast.SectionLine) error {
	if len(sl.Content) > 0 {
		// Single line break = space (prose reflow)
		r.pdf.Write(r.lineHeight, " ")
	}
	return nil
}

func (r *pdfRenderer) beginAct(s *ast.Section) error {
	// Numbered acts start on a new page. Title-only headings
	// (the play title parsed as an Act) are skipped when a title
	// page already rendered them.
	if s.Number != "" {
		if !r.consumePendingTitlePageBodyPage() && !r.consumePendingDramatisBodyPage() && !r.consumePendingInlinePlayFirstBodyPage() {
			r.pdf.AddPage()
		}
	} else {
		if r.hasTitlePage {
			return nil
		}
		r.ensureSpace(r.lineHeight * 4)
		r.pdf.Ln(r.lineHeight * 2)
	}

	bookmarkSection(&r.pdfBase, s)

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "ACT " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "ACT " + s.Number
	default:
		heading = s.Title
	}

	r.setStyle("BU")
	r.centeredText(strings.ToUpper(heading))
	r.setStyle("")
	r.pdf.Ln(r.lineHeight * 2)
	return nil
}

func (r *pdfRenderer) beginScene(s *ast.Section) error {
	if !r.consumePendingTitlePageBodyPage() && !r.consumePendingDramatisBodyPage() && !r.consumePendingInlinePlayFirstBodyPage() {
		r.ensureSpace(r.lineHeight * 3)
		r.pdf.Ln(r.lineHeight)
	}

	bookmarkSection(&r.pdfBase, s)

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

func (r *pdfRenderer) BeginDualDialogue(d *ast.DualDialogue) error {
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

func (r *pdfRenderer) EndDualDialogue(_ *ast.DualDialogue) error {
	if !r.inDualDialogue {
		r.dualSequential = false
		return nil
	}

	// Set Y to the bottom of the taller column
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

func (r *pdfRenderer) BeginDialogue(d *ast.Dialogue) error {
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

func (r *pdfRenderer) beginDualDialogueSide(d *ast.Dialogue) error {
	halfW := r.bodyW / 2
	gap := 4.0 // mm gap between columns
	colW := halfW - gap/2

	var leftM, rightM float64
	if r.dualSide == 0 {
		// Left column
		leftM = r.marginL
		rightM = r.pageW - r.marginL - colW
	} else {
		// Right column: restore Y to start, shift to right half
		r.dualMidY = r.pdf.GetY()
		r.pdf.SetY(r.dualStartY)
		leftM = r.marginL + halfW + gap/2
		rightM = r.marginR
	}

	// Set margins before Ln so X resets to the correct column
	r.pdf.SetLeftMargin(leftM)
	r.pdf.SetRightMargin(rightM)
	r.pdf.Ln(r.lineHeight)

	// Character name — centered in column, bold
	r.setStyle("B")
	name := strings.ToUpper(d.Character)
	nameW := r.pdf.GetStringWidth(name)
	r.pdf.SetX(leftM + (colW-nameW)/2)
	r.pdf.Write(r.lineHeight, name)
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")

	// Parenthetical — centered in column, italic
	if d.Parenthetical != "" {
		parenInlines := dialogueParentheticalInlines(d)
		r.setStyle("I")
		paren := parentheticalPlainText(d.Parenthetical, parenInlines)
		parenW := r.pdf.GetStringWidth(paren)
		r.pdf.SetX(leftM + (colW-parenW)/2)
		r.pdf.Write(r.lineHeight, "(")
		if err := r.renderInlineContent(parenInlines); err != nil {
			return err
		}
		r.pdf.Write(r.lineHeight, ")")
		r.pdf.Ln(r.lineHeight)
		r.setStyle("")
	}

	return nil
}

func (r *pdfRenderer) estimateDualDialogueHeight(d *ast.DualDialogue) float64 {
	halfW := r.bodyW / 2
	gap := 4.0
	colW := halfW - gap/2

	leftHeight := r.estimateDialogueHeight(d.Left, colW)
	rightHeight := r.estimateDialogueHeight(d.Right, colW)
	if rightHeight > leftHeight {
		return rightHeight
	}
	return leftHeight
}

func (r *pdfRenderer) estimateDialogueHeight(d *ast.Dialogue, width float64) float64 {
	if d == nil {
		return 0
	}

	height := r.lineHeight * 2 // leading blank line + character name
	if d.Parenthetical != "" {
		height += r.lineHeight
	}

	for _, line := range d.Lines {
		if len(line.Content) == 0 {
			height += r.lineHeight
			continue
		}
		lineWidth := width
		if line.IsVerse {
			lineWidth -= 10
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

func (r *pdfRenderer) EndDialogue(_ *ast.Dialogue) error {
	if r.inDualDialogue {
		r.dualSide++
		return nil
	}
	if r.activeDialogue == nil {
		return nil
	}
	dialogue := *r.activeDialogue
	r.activeDialogue = nil
	return r.renderBufferedDialogue(dialogue)
}

func (r *pdfRenderer) BeginDialogueLine(line *ast.DialogueLine) error {
	if r.inDualDialogue {
		r.captureDialogueLine = false
		if len(line.Content) == 0 {
			return nil
		}
		// In dual dialogue, lines flow within the current column margins
		leftM, _, _, _ := r.pdf.GetMargins()
		if line.IsVerse {
			r.pdf.SetX(leftM + 10)
		} else {
			r.pdf.SetX(leftM)
		}
		return nil
	}

	r.beginCapturedDialogueLine()
	return nil
}

func (r *pdfRenderer) EndDialogueLine(line *ast.DialogueLine) error {
	if r.inDualDialogue {
		r.pdf.Ln(r.lineHeight)
		return nil
	}
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

// --- Stage Direction ---

func (r *pdfRenderer) BeginStageDirection(sd *ast.StageDirection) error {
	switch {
	case sd.Continuation:
		// Adjacent line — regular line break, no extra gap
	case r.prevWasStageDirection:
		// Separated by blank lines — single blank line gap
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.lineHeight)
	default:
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.lineHeight / 2)
	}
	r.setStyle("I")
	r.pdf.SetX(r.marginL)
	return nil
}

func (r *pdfRenderer) EndStageDirection(_ *ast.StageDirection) error {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")
	r.prevWasStageDirection = true
	r.prevWasCallout = false
	return nil
}

// --- Callout ---

func (r *pdfRenderer) BeginCallout(c *ast.Callout) error {
	switch {
	case c.Continuation:
	case r.prevWasCallout:
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.lineHeight)
	default:
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.lineHeight / 2)
	}
	calloutIndent := halfInchPt * pointsToMM
	r.setStyle("B")
	r.pdf.SetLeftMargin(r.marginL + calloutIndent)
	r.pdf.SetX(r.marginL + calloutIndent)
	return nil
}

func (r *pdfRenderer) EndCallout(_ *ast.Callout) error {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")
	r.pdf.SetLeftMargin(r.marginL)
	r.prevWasStageDirection = false
	r.prevWasCallout = true
	return nil
}

// --- Song ---

func (r *pdfRenderer) BeginSong(song *ast.Song) error {
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
	r.pdf.Ln(r.lineHeight)
	return nil
}

func (r *pdfRenderer) EndSong(_ *ast.Song) error {
	r.resetBodyBlockState()
	r.pdf.Ln(r.lineHeight / 2)
	r.setStyle("B")
	r.centeredText("SONG END")
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Verse Block ---

func (r *pdfRenderer) BeginVerseBlock(_ *ast.VerseBlock) error {
	r.resetBodyBlockState()
	r.ensureSpace(r.lineHeight * 2)
	return nil
}

func (r *pdfRenderer) EndVerseBlock(_ *ast.VerseBlock) error {
	return nil
}

func (r *pdfRenderer) BeginVerseLine(_ *ast.VerseLine) error {
	verseX := r.marginL + r.bodyW*0.15 + 10
	r.pdf.SetX(verseX)
	return nil
}

func (r *pdfRenderer) EndVerseLine(_ *ast.VerseLine) error {
	r.pdf.Ln(r.lineHeight)
	return nil
}
