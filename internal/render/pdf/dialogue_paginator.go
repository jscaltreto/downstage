package pdf

import "fmt"

type dialoguePaginationStrategy interface {
	prepare(lines []bufferedDialogueLine, dialogue bufferedDialogue, continuation, firstSegment bool) []bufferedDialogueLine
	availableWrappedLines(dialogue bufferedDialogue, continuation, firstSegment bool) int
	// renderSegment receives lines already prepared for the current segment
	// context. Strategies must not re-wrap them.
	renderSegment(dialogue bufferedDialogue, continuation, firstSegment bool, lines []bufferedDialogueLine, showMore bool)
	addPage()
	showContinuationFooter() bool
}

func paginateBufferedDialogue(strategy dialoguePaginationStrategy, dialogue bufferedDialogue) error {
	remaining := append([]bufferedDialogueLine(nil), dialogue.lines...)
	firstSegment := true
	retriedFreshPage := false

	for len(remaining) > 0 {
		continuation := !firstSegment
		prepared := strategy.prepare(remaining, dialogue, continuation, firstSegment)
		linesFit := strategy.availableWrappedLines(dialogue, continuation, firstSegment)

		if fullCount, _ := fitCompleteDialogueLines(prepared, linesFit); fullCount == len(prepared) {
			strategy.renderSegment(dialogue, continuation, firstSegment, prepared, false)
			return nil
		}

		if strategy.showContinuationFooter() {
			linesFit = max(linesFit-1, 0)
		}
		fullCount, usedLines := fitCompleteDialogueLines(prepared, linesFit)

		if fullCount == 0 {
			left, right, ok := forceSplitDialogueLine(remaining[0], prepared[0], linesFit)
			if ok {
				leftPrepared := strategy.prepare([]bufferedDialogueLine{left}, dialogue, continuation, firstSegment)
				strategy.renderSegment(dialogue, continuation, firstSegment, leftPrepared, true)
				strategy.addPage()
				remaining = append([]bufferedDialogueLine{right}, remaining[1:]...)
				firstSegment = false
				retriedFreshPage = false
				continue
			}
			if retriedFreshPage {
				return fmt.Errorf("dialogue line cannot fit on a fresh page")
			}
			strategy.addPage()
			retriedFreshPage = true
			continue
		}

		if usedLines < minContinuedDialogueLines {
			if retriedFreshPage {
				return fmt.Errorf("dialogue line cannot leave the minimum fragment on a fresh page")
			}
			strategy.addPage()
			retriedFreshPage = true
			continue
		}

		currentPrepared := prepared[fullCount]
		if len(currentPrepared.wrappedText) == 0 || currentPrepared.wrappedLineCount() <= linesFit {
			strategy.renderSegment(dialogue, continuation, firstSegment, prepared[:fullCount], true)
			strategy.addPage()
			remaining = remaining[fullCount:]
			firstSegment = false
			retriedFreshPage = false
			continue
		}

		fitInCurrent := linesFit - usedLines
		if fitInCurrent < minContinuedDialogueLines {
			strategy.renderSegment(dialogue, continuation, firstSegment, prepared[:fullCount], true)
			strategy.addPage()
			remaining = remaining[fullCount:]
			firstSegment = false
			retriedFreshPage = false
			continue
		}

		totalCurrentLines := currentPrepared.wrappedLineCount()
		if totalCurrentLines-fitInCurrent < minContinuedDialogueLines {
			fitInCurrent = totalCurrentLines - minContinuedDialogueLines
		}
		if fitInCurrent < minContinuedDialogueLines {
			strategy.renderSegment(dialogue, continuation, firstSegment, prepared[:fullCount], true)
			strategy.addPage()
			remaining = remaining[fullCount:]
			firstSegment = false
			retriedFreshPage = false
			continue
		}

		left, right, ok := splitRawDialogueLine(remaining[fullCount], currentPrepared, fitInCurrent, minContinuedDialogueLines)
		if !ok || len(right.runs) == 0 {
			strategy.renderSegment(dialogue, continuation, firstSegment, prepared[:fullCount], true)
			strategy.addPage()
			remaining = remaining[fullCount:]
			firstSegment = false
			retriedFreshPage = false
			continue
		}

		segment := append(append([]bufferedDialogueLine(nil), remaining[:fullCount]...), left)
		segmentPrepared := strategy.prepare(segment, dialogue, continuation, firstSegment)
		strategy.renderSegment(dialogue, continuation, firstSegment, segmentPrepared, true)
		strategy.addPage()
		remaining = append(append([]bufferedDialogueLine(nil), right), remaining[fullCount+1:]...)
		firstSegment = false
		retriedFreshPage = false
	}

	return nil
}

func forceSplitDialogueLine(raw, prepared bufferedDialogueLine, linesFit int) (bufferedDialogueLine, bufferedDialogueLine, bool) {
	if linesFit < minContinuedDialogueLines {
		return bufferedDialogueLine{}, bufferedDialogueLine{}, false
	}

	totalLines := prepared.wrappedLineCount()
	leftLines := linesFit
	if totalLines-leftLines < minContinuedDialogueLines {
		leftLines = totalLines - minContinuedDialogueLines
	}
	if leftLines < minContinuedDialogueLines {
		return bufferedDialogueLine{}, bufferedDialogueLine{}, false
	}

	return splitRawDialogueLine(raw, prepared, leftLines, minContinuedDialogueLines)
}
