package pdf

import (
	"strings"
	"unicode"
)

const minContinuedDialogueLines = 3
const continuedDialogueSuffix = " (CONT'D)"
const continuedDialogueFooter = "(MORE)"

type bufferedDialogue struct {
	character     string
	parenthetical string
	lines         []bufferedDialogueLine
}

type bufferedDialogueLine struct {
	runs        []dialogueTextRun
	isVerse     bool
	plainText   string
	wrappedText []string
}

func (r *pdfRenderer) prepareDialogueLine(line bufferedDialogueLine) bufferedDialogueLine {
	line.plainText = dialogueRunsPlainText(line.runs)
	if line.plainText == "" {
		line.wrappedText = nil
		return line
	}
	width := r.dialogueContentWidth(line.isVerse)
	line.wrappedText = r.pdf.SplitText(line.plainText, width)
	return line
}

func (r *pdfRenderer) renderBufferedDialogue(d bufferedDialogue) error {
	return paginateBufferedDialogue(r, d)
}

func (r *pdfRenderer) prepare(lines []bufferedDialogueLine, _ bufferedDialogue, _, _ bool) []bufferedDialogueLine {
	prepared := make([]bufferedDialogueLine, len(lines))
	for i := range lines {
		prepared[i] = r.prepareDialogueLine(lines[i])
	}
	return prepared
}

func (r *pdfRenderer) availableWrappedLines(dialogue bufferedDialogue, _ bool, firstSegment bool) int {
	headerHeight := r.dialogueHeaderHeight(firstSegment, dialogue.parenthetical != "")
	if headerHeight > r.remainingPageHeight() {
		r.pdf.AddPage()
	}
	return max(int((r.remainingPageHeight()-headerHeight)/r.lineHeight), 0)
}

func (r *pdfRenderer) renderSegment(dialogue bufferedDialogue, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool) {
	r.renderDialogueSegment(dialogue.character, dialogue.parenthetical, continuation, firstSegment, lines, showMore)
}

func (r *pdfRenderer) addPage() {
	r.pdf.AddPage()
}

func (r *pdfRenderer) showContinuationFooter() bool {
	return true
}

func (r *pdfRenderer) renderDialogueSegment(character, parenthetical string, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool) {
	r.renderDialogueHeader(character, parenthetical, continuation, firstSegment)
	dialogueMargin := r.bodyW * 0.15
	dialogueX := r.marginL + dialogueMargin
	r.pdf.SetLeftMargin(dialogueX)
	r.pdf.SetRightMargin(r.marginR + dialogueMargin)
	for _, line := range lines {
		if len(line.runs) == 0 {
			r.pdf.Ln(r.lineHeight)
			continue
		}

		r.setDialogueLineX(line.isVerse)
		for _, run := range line.runs {
			r.setStyle(run.style)
			r.pdf.Write(r.lineHeight, run.text)
		}
		r.setStyle("")
		r.pdf.Ln(r.lineHeight)
	}
	if showMore {
		r.setStyle("B")
		r.centeredText(continuedDialogueFooter)
		r.setStyle("")
	}
	r.pdf.SetLeftMargin(r.marginL)
	r.pdf.SetRightMargin(r.marginR)
}

func (r *pdfRenderer) renderDialogueHeader(character, parenthetical string, continuation, firstSegment bool) {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("B")
	name := strings.ToUpper(character)
	if continuation {
		name += continuedDialogueSuffix
	}
	r.centeredText(name)
	r.setStyle("")

	if firstSegment && parenthetical != "" {
		r.setStyle("I")
		r.centeredText(parentheticalText(parenthetical))
		r.setStyle("")
	}
}

func (r *pdfRenderer) dialogueHeaderHeight(firstSegment, hasParenthetical bool) float64 {
	height := r.lineHeight * 2
	if firstSegment && hasParenthetical {
		height += r.lineHeight
	}
	return height
}

func (r *pdfRenderer) dialogueContentWidth(isVerse bool) float64 {
	dialogueMargin := r.bodyW * 0.15
	width := r.bodyW - (dialogueMargin * 2)
	if isVerse {
		width -= 10
	}
	if width < 10 {
		return 10
	}
	return width
}

func (r *pdfRenderer) setDialogueLineX(isVerse bool) {
	dialogueMargin := r.bodyW * 0.15
	dialogueX := r.marginL + dialogueMargin
	if isVerse {
		r.pdf.SetX(dialogueX + 10)
		return
	}
	r.pdf.SetX(dialogueX)
}

func fitCompleteDialogueLines(lines []bufferedDialogueLine, maxWrappedLines int) (count int, used int) {
	if maxWrappedLines <= 0 {
		return 0, 0
	}
	for _, line := range lines {
		next := line.wrappedLineCount()
		if used+next > maxWrappedLines {
			break
		}
		used += next
		count++
	}
	return count, used
}

func splitRawDialogueLine(raw, prepared bufferedDialogueLine, leftWrappedLines, minLeftWrappedLines int) (bufferedDialogueLine, bufferedDialogueLine, bool) {
	if leftWrappedLines <= 0 || leftWrappedLines >= len(prepared.wrappedText) {
		return bufferedDialogueLine{}, raw, false
	}

	// `prepared` carries wrap measurements only; split offsets must still map
	// 1:1 onto the original captured runs in `raw`.
	offset, ok := preferredSplitOffset(prepared, leftWrappedLines, minLeftWrappedLines)
	if !ok {
		return bufferedDialogueLine{}, raw, false
	}

	leftRuns, rightRuns := splitDialogueRuns(raw.runs, offset)
	return bufferedDialogueLine{runs: leftRuns, isVerse: raw.isVerse}, bufferedDialogueLine{runs: rightRuns, isVerse: raw.isVerse}, true
}

func preferredSplitOffset(line bufferedDialogueLine, preferred, minLeft int) (int, bool) {
	if preferred < minLeft {
		return 0, false
	}
	ranges := wrappedLineRuneRanges(line.plainText, line.wrappedText)

	boundaries := sentenceBoundaryOffsets(line.plainText)
	if len(boundaries) > 0 {
		for i := len(boundaries) - 1; i >= 0; i-- {
			offset := boundaries[i]
			leftCount, _ := wrappedLineCountsForOffset(ranges, offset)
			if leftCount > preferred {
				continue
			}
			if leftCount < minLeft {
				break
			}
			return offset, true
		}
	}

	rawOffset := dialogueSplitOffset(line.plainText, line.wrappedText[:preferred])
	leftCount, rightCount := wrappedLineCountsForOffset(ranges, rawOffset)
	if leftCount < minLeft || rightCount < minContinuedDialogueLines {
		return 0, false
	}
	return rawOffset, true
}

func dialogueRunsPlainText(runs []dialogueTextRun) string {
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.text)
	}
	return b.String()
}

func dialogueSplitOffset(text string, wrapped []string) int {
	runes := []rune(text)
	offset := 0
	for _, part := range wrapped {
		partRunes := []rune(part)
		offset += len(partRunes)
		for offset < len(runes) && runes[offset] == ' ' {
			offset++
		}
	}
	return offset
}

type wrappedLineRuneRange struct {
	start int
	end   int
}

func wrappedLineRuneRanges(text string, wrapped []string) []wrappedLineRuneRange {
	ranges := make([]wrappedLineRuneRange, 0, len(wrapped))
	runes := []rune(text)
	offset := 0
	for _, part := range wrapped {
		start := offset
		end := start + len([]rune(part))
		ranges = append(ranges, wrappedLineRuneRange{start: start, end: end})
		offset = end
		for offset < len(runes) && runes[offset] == ' ' {
			offset++
		}
	}
	return ranges
}

func sentenceBoundaryOffsets(text string) []int {
	runes := []rune(text)
	var offsets []int
	for i := 0; i < len(runes); i++ {
		if !isSentenceTerminal(runes[i]) {
			continue
		}
		j := i + 1
		for j < len(runes) && isSentenceCloser(runes[j]) {
			j++
		}
		offsets = append(offsets, skipSentenceBoundarySpacing(text, j))
	}
	return offsets
}

func wrappedLineCountsForOffset(ranges []wrappedLineRuneRange, offset int) (int, int) {
	leftCount := 0
	rightCount := 0
	for _, r := range ranges {
		switch {
		case offset >= r.end:
			leftCount++
		case offset <= r.start:
			rightCount++
		default:
			leftCount++
			rightCount++
		}
	}
	return leftCount, rightCount
}

func skipSentenceBoundarySpacing(text string, offset int) int {
	runes := []rune(text)
	for offset < len(runes) && unicode.IsSpace(runes[offset]) {
		offset++
	}
	return offset
}

func splitDialogueRuns(runs []dialogueTextRun, offset int) ([]dialogueTextRun, []dialogueTextRun) {
	if offset <= 0 {
		return nil, append([]dialogueTextRun(nil), runs...)
	}

	var left []dialogueTextRun
	var right []dialogueTextRun
	remaining := offset

	for _, run := range runs {
		runes := []rune(run.text)
		if remaining >= len(runes) {
			left = append(left, run)
			remaining -= len(runes)
			continue
		}
		if remaining > 0 {
			left = append(left, dialogueTextRun{text: string(runes[:remaining]), style: run.style})
			right = append(right, dialogueTextRun{text: string(runes[remaining:]), style: run.style})
			remaining = 0
			continue
		}
		right = append(right, run)
	}

	return trimDialogueRunsRight(left), trimDialogueRunsLeft(right)
}

func (l bufferedDialogueLine) wrappedLineCount() int {
	if len(l.runs) == 0 {
		return 1
	}
	if len(l.wrappedText) == 0 {
		return 1
	}
	return len(l.wrappedText)
}

func isSentenceTerminal(r rune) bool {
	switch r {
	case '.', '!', '?':
		return true
	default:
		return false
	}
}

func isSentenceCloser(r rune) bool {
	switch r {
	case '"', '\'', ')', ']', '}':
		return true
	default:
		return false
	}
}

func trimDialogueRunsLeft(runs []dialogueTextRun) []dialogueTextRun {
	for i := 0; i < len(runs); i++ {
		trimmed := strings.TrimLeftFunc(runs[i].text, unicode.IsSpace)
		if trimmed == "" {
			continue
		}
		runs[i].text = trimmed
		return runs[i:]
	}
	return nil
}

func trimDialogueRunsRight(runs []dialogueTextRun) []dialogueTextRun {
	for i := len(runs) - 1; i >= 0; i-- {
		trimmed := strings.TrimRightFunc(runs[i].text, unicode.IsSpace)
		if trimmed == "" {
			continue
		}
		runs[i].text = trimmed
		return runs[:i+1]
	}
	return nil
}
