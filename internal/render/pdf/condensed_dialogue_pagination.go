package pdf

import (
	"strings"
	"unicode"

	"github.com/jscaltreto/downstage/internal/ast"
)

func (r *condensedRenderer) renderBufferedDialogue(d bufferedDialogue) error {
	return paginateBufferedDialogue(r, d)
}

func (r *condensedRenderer) prepare(lines []bufferedDialogueLine, dialogue bufferedDialogue, continuation, firstSegment bool) []bufferedDialogueLine {
	_, firstLineWidth := r.condensedPrefixLayout(dialogue.character, dialogue.parenthetical, dialogue.parentheticalInlines, firstSegment)
	return r.prepareDialogueLines(lines, firstLineWidth)
}

func (r *condensedRenderer) availableWrappedLines(dialogue bufferedDialogue, continuation, firstSegment bool) int {
	prefixExtraLines, _ := r.condensedPrefixLayout(dialogue.character, dialogue.parenthetical, dialogue.parentheticalInlines, firstSegment)
	leadInHeight := 0.0
	if firstSegment {
		leadInHeight = r.condensedSmallGap()
	}
	if leadInHeight > r.remainingPageHeight() {
		r.pdf.AddPage()
		leadInHeight = 0
		prefixExtraLines, _ = r.condensedPrefixLayout(dialogue.character, dialogue.parenthetical, dialogue.parentheticalInlines, firstSegment)
	}
	return max(int((r.remainingPageHeight()-leadInHeight)/r.lineHeight)-prefixExtraLines, 0)
}

func (r *condensedRenderer) renderSegment(dialogue bufferedDialogue, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool) {
	r.renderDialogueSegment(dialogue.character, dialogue.parenthetical, dialogue.parentheticalInlines, continuation, firstSegment, lines, showMore)
}

func (r *condensedRenderer) addPage() {
	r.pdf.AddPage()
}

func (r *condensedRenderer) showContinuationFooter() bool {
	return false
}

func (r *condensedRenderer) renderDialogueSegment(character, parenthetical string, parentheticalInlines []ast.Inline, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool) {
	parentheticalInlines = parentheticalInlineContent(parenthetical, parentheticalInlines)
	if firstSegment {
		r.ensureSpace(r.lineHeight * 2)
		r.pdf.Ln(r.condensedSmallGap())

		cue := strings.ToUpper(character) + "."
		x := r.measureTextWidth("B", cue) + r.measureTextWidth("", "  ")
		r.pdf.SetX(r.marginL)
		r.setStyle("B")
		r.pdf.Write(r.lineHeight, cue)
		r.setStyle("")
		r.pdf.Write(r.lineHeight, "  ")

		if parenthetical != "" || len(parentheticalInlines) > 0 {
			runs, err := r.captureInlineRuns(parentheticalInlines, "I")
			if err != nil {
				return
			}
			runs = append([]dialogueTextRun{{text: "(", style: "I"}}, runs...)
			runs = append(runs, dialogueTextRun{text: ")", style: "I"})
			x = r.renderWrappedStyledRuns(x, runs, r.bodyW)
			spaceWidth := r.measureTextWidth("", " ")
			if x > 0 && x+spaceWidth > r.bodyW {
				r.pdf.Ln(r.lineHeight)
				r.pdf.SetX(r.marginL)
			} else {
				r.pdf.Write(r.lineHeight, " ")
			}
		}
	}

	firstRenderedLine := true
	for _, line := range lines {
		if len(line.runs) == 0 {
			r.pdf.Ln(r.condensedSmallGap())
			continue
		}

		if firstRenderedLine {
			startX := r.pdf.GetX() - r.marginL
			if startX < 0 {
				startX = 0
			}
			firstRuns, remainingRuns := r.splitRunsForWidth(startX, line.runs, r.bodyW)
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

		prepared[i].wrappedText = r.wrapCondensedDialogueText(
			prepared[i].plainText,
			prepared[i].isVerse,
			firstTextLine,
			firstLineWidth,
		)
		if firstTextLine {
			firstTextLine = false
		}
	}

	return prepared
}

func (r *condensedRenderer) wrapCondensedDialogueText(text string, isVerse, useReducedFirstLine bool, firstLineWidth float64) []string {
	if text == "" {
		return nil
	}

	regularWidth := r.condensedRegularLineWidth(isVerse)
	if !useReducedFirstLine {
		return r.pdf.SplitText(text, regularWidth)
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	firstWrapped := r.pdf.SplitText(text, firstLineWidth)
	if len(firstWrapped) == 0 {
		return nil
	}

	firstLine := firstWrapped[0]
	offset := dialogueSplitOffset(text, []string{firstLine})
	lines := []string{firstLine}

	remaining := strings.TrimLeftFunc(string(runes[offset:]), unicode.IsSpace)
	if remaining == "" {
		return lines
	}

	return append(lines, r.pdf.SplitText(remaining, regularWidth)...)
}

func (r *condensedRenderer) condensedPrefixLayout(character, parenthetical string, parentheticalInlines []ast.Inline, firstSegment bool) (int, float64) {
	parentheticalInlines = parentheticalInlineContent(parenthetical, parentheticalInlines)
	if !firstSegment {
		return 0, r.bodyW
	}

	cue := strings.ToUpper(character)
	cue += "."
	x := r.measureTextWidth("B", cue) + r.measureTextWidth("", "  ")
	extraLines := 0

	if parenthetical != "" || len(parentheticalInlines) > 0 {
		x, extraLines = r.layoutCondensedInlineText(x, extraLines, parentheticalPlainText(parenthetical, parentheticalInlines), "I")
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

func (r *condensedRenderer) captureInlineRuns(inlines []ast.Inline, baseStyle string) ([]dialogueTextRun, error) {
	r.beginCapturedDialogueLine()
	r.captureStyle = baseStyle
	if err := r.renderInlineContent(inlines); err != nil {
		return nil, err
	}
	return r.endCapturedDialogueLine(), nil
}

func (r *condensedRenderer) renderWrappedStyledRuns(startX float64, runs []dialogueTextRun, maxWidth float64) float64 {
	leftM, _, _, _ := r.pdf.GetMargins()
	tokens := tokenizeDialogueRuns(runs)
	x := startX

	for len(tokens) > 0 {
		lineRuns, remainingTokens, endX := r.fitStyledRunsToWidth(x, tokens, maxWidth)
		if len(lineRuns) == 0 {
			r.pdf.Ln(r.lineHeight)
			r.pdf.SetX(leftM)
			x = 0
			tokens = trimLeadingWhitespaceTokens(tokens)
			continue
		}

		for _, run := range lineRuns {
			r.setStyle(run.style)
			r.pdf.Write(r.lineHeight, run.text)
		}
		x = endX
		tokens = remainingTokens
		if len(tokens) == 0 {
			break
		}

		r.pdf.Ln(r.lineHeight)
		r.pdf.SetX(leftM)
		x = 0
	}

	r.setStyle("")
	return x
}

type styledDialogueToken struct {
	text       string
	style      string
	whitespace bool
}

func tokenizeDialogueRuns(runs []dialogueTextRun) []styledDialogueToken {
	var tokens []styledDialogueToken
	for _, run := range runs {
		for len(run.text) > 0 {
			split := strings.IndexFunc(run.text, unicode.IsSpace)
			if split == -1 {
				tokens = append(tokens, styledDialogueToken{text: run.text, style: run.style})
				break
			}
			if split > 0 {
				tokens = append(tokens, styledDialogueToken{text: run.text[:split], style: run.style})
				run.text = run.text[split:]
				continue
			}
			end := strings.IndexFunc(run.text, func(r rune) bool { return !unicode.IsSpace(r) })
			if end == -1 {
				tokens = append(tokens, styledDialogueToken{text: run.text, style: run.style, whitespace: true})
				break
			}
			tokens = append(tokens, styledDialogueToken{text: run.text[:end], style: run.style, whitespace: true})
			run.text = run.text[end:]
		}
	}
	return tokens
}

func (r *condensedRenderer) splitRunsForWidth(startX float64, runs []dialogueTextRun, maxWidth float64) ([]dialogueTextRun, []dialogueTextRun) {
	first, remaining, _ := r.fitStyledRunsToWidth(startX, tokenizeDialogueRuns(runs), maxWidth)
	return first, collapseStyledTokens(remaining)
}

func collapseStyledTokens(tokens []styledDialogueToken) []dialogueTextRun {
	var runs []dialogueTextRun
	for _, token := range tokens {
		runs = appendDialogueRun(runs, token.text, token.style)
	}
	return runs
}

func appendDialogueRun(runs []dialogueTextRun, text, style string) []dialogueTextRun {
	if text == "" {
		return runs
	}
	if n := len(runs); n > 0 && runs[n-1].style == style {
		runs[n-1].text += text
		return runs
	}
	return append(runs, dialogueTextRun{text: text, style: style})
}

func (r *condensedRenderer) fitStyledRunsToWidth(startX float64, tokens []styledDialogueToken, maxWidth float64) ([]dialogueTextRun, []styledDialogueToken, float64) {
	x := startX
	var lineRuns []dialogueTextRun
	var pendingSpaces []styledDialogueToken

	for i, token := range tokens {
		if token.whitespace {
			pendingSpaces = append(pendingSpaces, token)
			continue
		}

		spaceWidth := r.measureStyledTokenWidth(pendingSpaces)
		tokenWidth := r.measureTextWidth(token.style, token.text)
		if x > 0 && x+spaceWidth+tokenWidth > maxWidth {
			return lineRuns, trimLeadingWhitespaceTokens(tokens[i:]), x
		}

		for _, ws := range pendingSpaces {
			lineRuns = appendDialogueRun(lineRuns, ws.text, ws.style)
			x += r.measureTextWidth(ws.style, ws.text)
		}
		lineRuns = appendDialogueRun(lineRuns, token.text, token.style)
		x += tokenWidth
		pendingSpaces = nil
	}

	return lineRuns, nil, x
}

func (r *condensedRenderer) measureStyledTokenWidth(tokens []styledDialogueToken) float64 {
	width := 0.0
	for _, token := range tokens {
		width += r.measureTextWidth(token.style, token.text)
	}
	return width
}

func trimLeadingWhitespaceTokens(tokens []styledDialogueToken) []styledDialogueToken {
	for i, token := range tokens {
		if !token.whitespace {
			return tokens[i:]
		}
	}
	return nil
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
