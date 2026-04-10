package pdf

import "strings"

func (r *condensedRenderer) renderBufferedDialogue(d bufferedDialogue) error {
	return paginateBufferedDialogue(r, d)
}

func (r *condensedRenderer) prepare(lines []bufferedDialogueLine, dialogue bufferedDialogue, continuation, firstSegment bool) []bufferedDialogueLine {
	_, firstLineWidth := r.condensedPrefixLayout(dialogue.character, dialogue.parenthetical, firstSegment)
	return r.prepareDialogueLines(lines, firstLineWidth)
}

func (r *condensedRenderer) availableWrappedLines(dialogue bufferedDialogue, continuation, firstSegment bool) int {
	prefixExtraLines, _ := r.condensedPrefixLayout(dialogue.character, dialogue.parenthetical, firstSegment)
	leadInHeight := 0.0
	if firstSegment {
		leadInHeight = r.lineHeight / 2
	}
	if leadInHeight > r.remainingPageHeight() {
		r.pdf.AddPage()
		leadInHeight = 0
		prefixExtraLines, _ = r.condensedPrefixLayout(dialogue.character, dialogue.parenthetical, firstSegment)
	}
	return max(int((r.remainingPageHeight()-leadInHeight)/r.lineHeight)-prefixExtraLines, 0)
}

func (r *condensedRenderer) renderSegment(dialogue bufferedDialogue, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool) {
	r.renderDialogueSegment(dialogue.character, dialogue.parenthetical, continuation, firstSegment, lines, showMore)
}

func (r *condensedRenderer) addPage() {
	r.pdf.AddPage()
}

func (r *condensedRenderer) showContinuationFooter() bool {
	return false
}

func (r *condensedRenderer) renderDialogueSegment(character, parenthetical string, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool) {
	if firstSegment {
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.lineHeight / 2)

		cue := strings.ToUpper(character) + "."
		r.pdf.SetX(r.marginL)
		r.setStyle("B")
		r.pdf.Write(r.lineHeight, cue)
		r.setStyle("")
		r.pdf.Write(r.lineHeight, "  ")

		if parenthetical != "" {
			r.setStyle("I")
			r.pdf.Write(r.lineHeight, parentheticalText(parenthetical))
			r.setStyle("")
			r.pdf.Write(r.lineHeight, " ")
		}
	}

	firstRenderedLine := true
	for _, line := range lines {
		if len(line.runs) == 0 {
			r.pdf.Ln(r.lineHeight / 2)
			continue
		}

		if firstRenderedLine {
			firstOffset := dialogueSplitOffset(line.plainText, line.wrappedText[:1])
			firstRuns, remainingRuns := splitDialogueRuns(line.runs, firstOffset)
			for _, run := range firstRuns {
				r.setStyle(run.style)
				r.pdf.CellFormat(r.pdf.GetStringWidth(run.text), r.lineHeight, run.text, "", 0, "", false, 0, "")
			}
			r.setStyle("")
			if len(remainingRuns) == 0 {
				r.pdf.Ln(r.lineHeight)
				firstRenderedLine = false
				continue
			}
			r.pdf.Ln(r.lineHeight)
			line.runs = remainingRuns
		}

		if !firstRenderedLine {
			if line.isVerse {
				r.pdf.SetX(r.marginL + 10)
			} else {
				r.pdf.SetX(r.marginL)
			}
		}
		for _, run := range line.runs {
			r.setStyle(run.style)
			r.pdf.Write(r.lineHeight, run.text)
		}
		r.setStyle("")
		r.pdf.Ln(r.lineHeight)
		firstRenderedLine = false
	}

	if showMore && r.showContinuationFooter() {
		r.setStyle("B")
		r.centeredText(continuedDialogueFooter)
		r.setStyle("")
	}
}

func (r *condensedRenderer) prepareDialogueLines(lines []bufferedDialogueLine, firstLineWidth float64) []bufferedDialogueLine {
	prepared := make([]bufferedDialogueLine, len(lines))
	copy(prepared, lines)

	firstTextLine := true

	for i := range prepared {
		prepared[i].plainText = dialogueRunsPlainText(prepared[i].runs)
		if prepared[i].plainText == "" {
			prepared[i].wrappedText = nil
			continue
		}

		width := r.condensedRegularLineWidth(prepared[i].isVerse)
		if firstTextLine {
			width = firstLineWidth
			firstTextLine = false
		}
		prepared[i].wrappedText = r.pdf.SplitText(prepared[i].plainText, width)
	}

	return prepared
}

func (r *condensedRenderer) condensedPrefixLayout(character, parenthetical string, firstSegment bool) (int, float64) {
	if !firstSegment {
		return 0, r.bodyW
	}

	cue := strings.ToUpper(character)
	cue += "."
	x := r.measureTextWidth("B", cue) + r.measureTextWidth("", "  ")
	extraLines := 0

	if parenthetical != "" {
		x, extraLines = r.layoutCondensedInlineText(x, extraLines, parentheticalText(parenthetical), "I")
		spaceWidth := r.measureTextWidth("", " ")
		if x+spaceWidth > r.bodyW {
			extraLines++
			x = 0
		} else {
			x += spaceWidth
		}
	}

	width := r.bodyW - x
	if width < 10 {
		extraLines++
		width = r.bodyW
	}
	return extraLines, width
}

func (r *condensedRenderer) measureTextWidth(style, text string) float64 {
	currentStyle := r.fontStyle
	r.setStyle(style)
	width := r.pdf.GetStringWidth(text)
	r.setStyle(currentStyle)
	return width
}

func (r *condensedRenderer) layoutCondensedInlineText(startX float64, extraLines int, text, style string) (float64, int) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return startX, extraLines
	}

	x := startX
	for i, word := range words {
		token := word
		if i > 0 {
			token = " " + word
		}
		tokenWidth := r.measureTextWidth(style, token)
		if x > 0 && x+tokenWidth > r.bodyW {
			extraLines++
			x = r.measureTextWidth(style, word)
			continue
		}
		x += tokenWidth
	}

	return x, extraLines
}

func parentheticalText(parenthetical string) string {
	paren := parenthetical
	if paren == "" {
		return ""
	}
	if paren[0] != '(' {
		paren = "(" + paren + ")"
	}
	return paren
}

func (r *condensedRenderer) condensedRegularLineWidth(isVerse bool) float64 {
	width := r.bodyW
	if isVerse {
		width -= 10
	}
	if width < 10 {
		return 10
	}
	return width
}
