package pdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitDialogueRunsPreservesStyles(t *testing.T) {
	runs := []dialogueTextRun{
		{text: "Hello ", style: ""},
		{text: "brave ", style: "B"},
		{text: "world", style: ""},
	}

	left, right := splitDialogueRuns(runs, len([]rune("Hello brave ")))

	require.Equal(t, []dialogueTextRun{
		{text: "Hello ", style: ""},
		{text: "brave", style: "B"},
	}, left)
	require.Equal(t, []dialogueTextRun{
		{text: "world", style: ""},
	}, right)
}

func TestSplitDialogueRunsTrimsBoundaryWhitespace(t *testing.T) {
	runs := []dialogueTextRun{
		{text: "Hello.   ", style: ""},
		{text: " next", style: "I"},
	}

	left, right := splitDialogueRuns(runs, len([]rune("Hello.   ")))

	require.Equal(t, []dialogueTextRun{{text: "Hello.", style: ""}}, left)
	require.Equal(t, []dialogueTextRun{{text: "next", style: "I"}}, right)
}

func TestPreferredSplitOffsetUsesSentenceBoundary(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: strings.Join([]string{
			"First sentence continues",
			"and keeps going",
			"until this sentence ends.",
			"next fragment carries on",
			"for a while longer",
			"before it stops",
		}, " "),
		wrappedText: []string{
			"First sentence continues",
			"and keeps going",
			"until this sentence ends.",
			"next fragment carries on",
			"for a while longer",
			"before it stops",
		},
	}

	split, ok := preferredSplitOffset(line, 5, minContinuedDialogueLines)
	require.True(t, ok)
	assert.Equal(t, dialogueSplitOffset(line.plainText, line.wrappedText[:3]), split)
}

func TestPreferredSplitWrappedLinesFallsBackWhenNoSentenceBoundary(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: strings.Join([]string{
			"First fragment",
			"Second fragment",
			"Third fragment",
			"Fourth fragment",
			"Fifth fragment",
			"Sixth fragment",
		}, " "),
		wrappedText: []string{
			"First fragment",
			"Second fragment",
			"Third fragment",
			"Fourth fragment",
			"Fifth fragment",
			"Sixth fragment",
		},
	}

	split, ok := preferredSplitOffset(line, 3, minContinuedDialogueLines)
	require.True(t, ok)
	assert.Equal(t, dialogueSplitOffset(line.plainText, line.wrappedText[:3]), split)
}

func TestPreferredSplitWrappedLinesUsesNearbySentenceBoundary(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: strings.Join([]string{
			"Opening fragment",
			"middle fragment",
			"Sentence ends here.",
			"Second sentence keeps going",
			"without punctuation",
			"to the end",
		}, " "),
		wrappedText: []string{
			"Opening fragment",
			"middle fragment",
			"Sentence ends here.",
			"Second sentence keeps going",
			"without punctuation",
			"to the end",
		},
	}

	split, ok := preferredSplitOffset(line, 5, minContinuedDialogueLines)
	require.True(t, ok)
	assert.Equal(t, dialogueSplitOffset(line.plainText, line.wrappedText[:3]), split)
}

func TestPreferredSplitOffsetRejectsNearbySentenceBoundaryBeforeMinimum(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: strings.Join([]string{
			"First sentence.",
			"continuation fragment one",
			"continuation fragment two",
			"continuation fragment three",
		}, " "),
		wrappedText: []string{
			"First sentence.",
			"continuation fragment one",
			"continuation fragment two",
			"continuation fragment three",
		},
	}

	_, ok := preferredSplitOffset(line, 3, minContinuedDialogueLines)
	assert.False(t, ok)
}

func TestPreferredSplitOffsetRejectsSentenceBoundaryInsideFirstWrappedLine(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: "I believe we are capable of extraordinary things. I also believe we are capable of repeating every error that condemned Porth, and every error we have made aboard this ship.",
		wrappedText: []string{
			"I believe we are capable of extraordinary things. I",
			"also believe we are capable of repeating every error that",
			"condemned Porth, and every error we",
			"have made aboard this ship.",
		},
	}

	_, ok := preferredSplitOffset(line, 3, minContinuedDialogueLines)
	assert.False(t, ok)
}

func TestPreferredSplitOffsetFallsBackAfterUnusableSentenceBoundary(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: strings.Join([]string{
			"First sentence.",
			"very long fragment one",
			"very long fragment two",
			"very long fragment three",
			"very long fragment four",
			"very long fragment five",
		}, " "),
		wrappedText: []string{
			"First sentence.",
			"very long fragment one",
			"very long fragment two",
			"very long fragment three",
			"very long fragment four",
			"very long fragment five",
		},
	}

	split, ok := preferredSplitOffset(line, 3, minContinuedDialogueLines)
	require.True(t, ok)
	assert.Equal(t, dialogueSplitOffset(line.plainText, line.wrappedText[:3]), split)
}

func TestPreferredSplitWrappedLinesAllowsShortRemainderAtSentenceBoundary(t *testing.T) {
	line := bufferedDialogueLine{
		plainText: strings.Join([]string{
			"long fragment one",
			"long fragment two",
			"long fragment three",
			"Sentence ends here.",
			"short remainder one",
			"short remainder two",
		}, " "),
		wrappedText: []string{
			"long fragment one",
			"long fragment two",
			"long fragment three",
			"Sentence ends here.",
			"short remainder one",
			"short remainder two",
		},
	}

	split, ok := preferredSplitOffset(line, 4, minContinuedDialogueLines)
	require.True(t, ok)
	assert.Equal(t, dialogueSplitOffset(line.plainText, line.wrappedText[:4]), split)
}

func TestRenderBufferedDialogueStartsContinuationOnNewPage(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	var buf bytes.Buffer
	require.NoError(t, r.BeginDocument(&ast.Document{}, &buf))
	r.pdf.SetCompression(false)

	r.pdf.SetY(r.pageH - r.marginB - (r.lineHeight * 8))
	dialogue := bufferedDialogue{
		character: "HAMLET",
		lines: []bufferedDialogueLine{
			{runs: []dialogueTextRun{{text: "Short line.", style: ""}}},
			{runs: []dialogueTextRun{{text: strings.Repeat("word ", 80), style: ""}}},
		},
	}

	require.NoError(t, r.renderBufferedDialogue(dialogue))
	require.NoError(t, r.EndDocument(&ast.Document{}))
	assert.Greater(t, r.pdf.PageNo(), 1)
}

func TestRenderBufferedDialogueForceSplitsSingleLongLine(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	dialogue := bufferedDialogue{
		character: "HAMLET",
		lines: []bufferedDialogueLine{
			{runs: []dialogueTextRun{{text: strings.Repeat("word ", 1200), style: ""}}},
		},
	}

	require.NoError(t, r.renderBufferedDialogue(dialogue))
	assert.Greater(t, r.pdf.PageNo(), 1)
}

func TestRenderBufferedDialogueStartsContinuationOnNewPageCondensed(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	r.pdf.SetY(r.pageH - r.marginB - (r.lineHeight * 8))
	dialogue := bufferedDialogue{
		character: "HAMLET",
		lines: []bufferedDialogueLine{
			{runs: []dialogueTextRun{{text: "Short line.", style: ""}}},
			{runs: []dialogueTextRun{{text: strings.Repeat("word ", 80), style: ""}}},
		},
	}

	require.NoError(t, r.renderBufferedDialogue(dialogue))
	assert.Greater(t, r.pdf.PageNo(), 1)
}

func TestRenderBufferedDialogueWithCapturedStyles(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	dialogue := &ast.Dialogue{Character: "HAMLET"}
	require.NoError(t, r.BeginDialogue(dialogue))

	line := &ast.DialogueLine{Content: []ast.Inline{
		&ast.TextNode{Value: strings.Repeat("word ", 80)},
		&ast.BoldNode{Content: []ast.Inline{&ast.TextNode{Value: "bold"}}},
		&ast.TextNode{Value: " "},
		&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "italic"}}},
	}}
	require.NoError(t, r.BeginDialogueLine(line))
	require.NoError(t, r.RenderText(line.Content[0].(*ast.TextNode)))
	require.NoError(t, r.BeginBold(line.Content[1].(*ast.BoldNode)))
	require.NoError(t, r.RenderText(line.Content[1].(*ast.BoldNode).Content[0].(*ast.TextNode)))
	require.NoError(t, r.EndBold(line.Content[1].(*ast.BoldNode)))
	require.NoError(t, r.RenderText(line.Content[2].(*ast.TextNode)))
	require.NoError(t, r.BeginItalic(line.Content[3].(*ast.ItalicNode)))
	require.NoError(t, r.RenderText(line.Content[3].(*ast.ItalicNode).Content[0].(*ast.TextNode)))
	require.NoError(t, r.EndItalic(line.Content[3].(*ast.ItalicNode)))
	require.NoError(t, r.EndDialogueLine(line))
	require.Len(t, r.activeDialogue.lines, 1)
	require.Equal(t, strings.Repeat("word ", 80)+"bold italic", dialogueRunsPlainText(r.activeDialogue.lines[0].runs))
	require.Contains(t, r.activeDialogue.lines[0].runs, dialogueTextRun{text: "bold", style: "B"})
	require.Contains(t, r.activeDialogue.lines[0].runs, dialogueTextRun{text: "italic", style: "I"})
	require.NoError(t, r.EndDialogue(dialogue))

	assert.Greater(t, r.pdf.PageNo(), 0)
}

func TestRenderBufferedDialogue_InlineDirectionNestedItalicPreservesItalicContext(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	dialogue := &ast.Dialogue{Character: "HAMLET"}
	require.NoError(t, r.BeginDialogue(dialogue))

	line := &ast.DialogueLine{Content: []ast.Inline{
		&ast.TextNode{Value: "To be "},
		&ast.InlineDirectionNode{Content: []ast.Inline{
			&ast.TextNode{Value: "not merely "},
			&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "appearing"}}},
			&ast.TextNode{Value: " so"},
		}},
		&ast.TextNode{Value: "."},
	}}
	require.NoError(t, r.BeginDialogueLine(line))
	require.NoError(t, r.RenderText(line.Content[0].(*ast.TextNode)))
	require.NoError(t, r.BeginInlineDirection(line.Content[1].(*ast.InlineDirectionNode)))
	require.NoError(t, r.RenderText(line.Content[1].(*ast.InlineDirectionNode).Content[0].(*ast.TextNode)))
	require.NoError(t, r.BeginItalic(line.Content[1].(*ast.InlineDirectionNode).Content[1].(*ast.ItalicNode)))
	require.NoError(t, r.RenderText(line.Content[1].(*ast.InlineDirectionNode).Content[1].(*ast.ItalicNode).Content[0].(*ast.TextNode)))
	require.NoError(t, r.EndItalic(line.Content[1].(*ast.InlineDirectionNode).Content[1].(*ast.ItalicNode)))
	require.NoError(t, r.RenderText(line.Content[1].(*ast.InlineDirectionNode).Content[2].(*ast.TextNode)))
	require.NoError(t, r.EndInlineDirection(line.Content[1].(*ast.InlineDirectionNode)))
	require.NoError(t, r.RenderText(line.Content[2].(*ast.TextNode)))
	require.NoError(t, r.EndDialogueLine(line))
	require.Len(t, r.activeDialogue.lines, 1)
	require.Equal(t, []dialogueTextRun{
		{text: "To be ", style: ""},
		{text: "(not merely appearing so)", style: "I"},
		{text: ".", style: ""},
	}, r.activeDialogue.lines[0].runs)
	require.NoError(t, r.EndDialogue(dialogue))
}
