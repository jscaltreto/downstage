package pdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_EmptyDocument(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0, "PDF output should not be empty")
	assert.Equal(t, "%PDF-", string(buf.Bytes()[:5]), "output should be a valid PDF")
}

func TestRender_TitlePageOnly(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "My Play"},
				{Key: "Author", Value: "Test Author"},
				{Key: "Date", Value: "2025"},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
	assert.Equal(t, 1, r.pdf.PageNo())
}

func TestRender_TopLevelSectionMetadataAsTitlePage(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "My Play",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{
						{Key: "Author", Value: "Test Author"},
					},
				},
				Children: []ast.Node{
					&ast.Section{
						Kind:   ast.SectionAct,
						Level:  2,
						Number: "I",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
	assert.Equal(t, 2, r.pdf.PageNo())
}

func TestRender_CompilationTitlePagesStartOnFreshPages(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "First Play",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Author", Value: "One"}},
				},
				Children: []ast.Node{
					&ast.Section{Kind: ast.SectionAct, Level: 2, Number: "I"},
				},
			},
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Second Play",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Author", Value: "Two"}},
				},
				Children: []ast.Node{
					&ast.Section{Kind: ast.SectionAct, Level: 2, Number: "I"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
	assert.Equal(t, 4, r.pdf.PageNo())
}

func TestRender_DialogueWithFormatting(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:   ast.SectionAct,
				Number: "I",
				Children: []ast.Node{
					&ast.Section{
						Kind:   ast.SectionScene,
						Number: "1",
						Children: []ast.Node{
							&ast.Dialogue{
								Character: "HAMLET",
								Lines: []ast.DialogueLine{
									{
										Content: []ast.Inline{
											&ast.TextNode{Value: "To be, or "},
											&ast.BoldNode{Content: []ast.Inline{
												&ast.TextNode{Value: "not"},
											}},
											&ast.TextNode{Value: " to be."},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_PageBreak(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.StageDirection{
				Content: []ast.Inline{
					&ast.TextNode{Value: "Before break"},
				},
			},
			&ast.PageBreak{},
			&ast.StageDirection{
				Content: []ast.Inline{
					&ast.TextNode{Value: "After break"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_Callout(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Callout{
				Content: []ast.Inline{
					&ast.TextNode{Value: "Midwinter. The room has not been heated for days."},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_DramatisPersonae(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{Name: "HAMLET", Description: "Prince of Denmark"},
					{Name: "HORATIO", Description: "Friend to Hamlet"},
				},
				Groups: []ast.CharacterGroup{
					{
						Name: "Courtiers",
						Characters: []ast.Character{
							{Name: "ROSENCRANTZ", Description: "A courtier"},
							{Name: "GUILDENSTERN", Description: "A courtier"},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_Song(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Song{
				Number: "1",
				Title:  "The Wanderer's Lament",
				Content: []ast.Node{
					&ast.Dialogue{
						Character: "HAMLET",
						Lines: []ast.DialogueLine{
							{
								Content: []ast.Inline{
									&ast.TextNode{Value: "O wanderer, where goest thou?"},
								},
								IsVerse: true,
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_A4PageSize(t *testing.T) {
	cfg := render.DefaultConfig()
	cfg.PageSize = render.PageA4
	r := NewRenderer(cfg)
	doc := &ast.Document{}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_InlineFormatting(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "TEST",
				Lines: []ast.DialogueLine{
					{
						Content: []ast.Inline{
							&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "italic"}}},
							&ast.TextNode{Value: " "},
							&ast.BoldItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "bold italic"}}},
							&ast.TextNode{Value: " "},
							&ast.UnderlineNode{Content: []ast.Inline{&ast.TextNode{Value: "underline"}}},
							&ast.TextNode{Value: " "},
							&ast.StrikethroughNode{Content: []ast.Inline{&ast.TextNode{Value: "strike"}}},
							&ast.TextNode{Value: " "},
							&ast.InlineDirectionNode{Content: []ast.Inline{&ast.TextNode{Value: "aside"}}},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_VerseBlock(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.VerseBlock{
				Lines: []ast.VerseLine{
					{Content: []ast.Inline{&ast.TextNode{Value: "Line one"}}},
					{Content: []ast.Inline{&ast.TextNode{Value: "Line two"}}},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 0)
}

func TestRender_FullIntegration(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "The Test Play"},
				{Key: "Author", Value: "Jane Doe"},
			},
		},
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{Name: "ALICE", Description: "A protagonist"},
					{Name: "BOB", Description: "An antagonist"},
				},
			},
			&ast.Section{
				Kind:   ast.SectionAct,
				Number: "I",
				Children: []ast.Node{
					&ast.Section{
						Kind:   ast.SectionScene,
						Number: "1",
						Title:  "The Beginning",
						Children: []ast.Node{
							&ast.StageDirection{
								Content: []ast.Inline{
									&ast.TextNode{Value: "(A dimly lit room.)"},
								},
							},
							&ast.Dialogue{
								Character:     "ALICE",
								Parenthetical: "entering",
								Lines: []ast.DialogueLine{
									{Content: []ast.Inline{
										&ast.TextNode{Value: "Hello, "},
										&ast.BoldNode{Content: []ast.Inline{&ast.TextNode{Value: "world"}}},
										&ast.TextNode{Value: "!"},
									}},
								},
							},
							&ast.Dialogue{
								Character: "BOB",
								Lines: []ast.DialogueLine{
									{Content: []ast.Inline{
										&ast.TextNode{Value: "O, what a tale:"},
									}},
									{Content: []ast.Inline{
										&ast.TextNode{Value: "  Of woe and wonder,"},
									}, IsVerse: true},
									{Content: []ast.Inline{
										&ast.TextNode{Value: "  Told in the night."},
									}, IsVerse: true},
								},
							},
							&ast.PageBreak{Range: token.Range{}},
							&ast.Song{
								Number: "1",
								Title:  "Evening Song",
								Content: []ast.Node{
									&ast.Dialogue{
										Character: "ALICE",
										Lines: []ast.DialogueLine{
											{Content: []ast.Inline{
												&ast.TextNode{Value: "Sing me a song."},
											}, IsVerse: true},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	assert.True(t, buf.Len() > 500, "full integration PDF should have substantial content")
	assert.Equal(t, "%PDF-", string(buf.Bytes()[:5]))
}

func TestBeginDualDialogueFallsBackToSequentialWhenTooTall(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	line := ast.DialogueLine{Content: []ast.Inline{&ast.TextNode{Value: strings.Repeat("word ", 40)}}}
	lines := make([]ast.DialogueLine, 80)
	for i := range lines {
		lines[i] = line
	}

	dual := &ast.DualDialogue{
		Left:  &ast.Dialogue{Character: "ALICE", Lines: lines},
		Right: &ast.Dialogue{Character: "BOB", Lines: lines},
	}

	require.NoError(t, r.BeginDualDialogue(dual))
	assert.False(t, r.inDualDialogue, "expected oversized dual dialogue to fall back to sequential rendering")
}

func TestBeginCondensedDualDialogueFallsBackToSequentialWhenTooTall(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	line := ast.DialogueLine{Content: []ast.Inline{&ast.TextNode{Value: strings.Repeat("word ", 30)}}}
	lines := make([]ast.DialogueLine, 80)
	for i := range lines {
		lines[i] = line
	}

	dual := &ast.DualDialogue{
		Left:  &ast.Dialogue{Character: "ALICE", Lines: lines},
		Right: &ast.Dialogue{Character: "BOB", Lines: lines},
	}

	require.NoError(t, r.BeginDualDialogue(dual))
	assert.False(t, r.inDualDialogue, "expected oversized dual dialogue to fall back to sequential rendering")
}

func TestRender_DialogueParagraphBreak(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionAct,
				Title: "One",
				Children: []ast.Node{
					&ast.Dialogue{
						Character: "HAMLET",
						Lines: []ast.DialogueLine{
							{Content: []ast.Inline{&ast.TextNode{Value: "First line."}}},
							{}, // paragraph break marker
							{Content: []ast.Inline{&ast.TextNode{Value: "Second line."}}},
						},
					},
				},
			},
		},
	}

	t.Run("standard", func(t *testing.T) {
		r := NewRenderer(render.DefaultConfig())
		var buf bytes.Buffer
		err := render.Walk(r, doc, &buf)
		require.NoError(t, err)
		assert.True(t, buf.Len() > 0, "PDF output should not be empty")
	})

	t.Run("condensed", func(t *testing.T) {
		r := NewCondensedRenderer(render.DefaultConfig())
		var buf bytes.Buffer
		err := render.Walk(r, doc, &buf)
		require.NoError(t, err)
		assert.True(t, buf.Len() > 0, "PDF output should not be empty")
	})
}

func TestRender_StageDirectionContinuation(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionAct,
				Title: "One",
				Children: []ast.Node{
					&ast.StageDirection{
						Content: []ast.Inline{&ast.TextNode{Value: "First direction."}},
					},
					&ast.StageDirection{
						Content:      []ast.Inline{&ast.TextNode{Value: "Adjacent direction."}},
						Continuation: true,
					},
					&ast.StageDirection{
						Content: []ast.Inline{&ast.TextNode{Value: "Separated direction."}},
					},
				},
			},
		},
	}

	for _, tc := range []struct {
		name string
		r    render.NodeRenderer
	}{
		{"standard", NewRenderer(render.DefaultConfig())},
		{"condensed", NewCondensedRenderer(render.DefaultConfig())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := render.Walk(tc.r, doc, &buf)
			require.NoError(t, err)
			assert.True(t, buf.Len() > 0, "PDF output should not be empty")
		})
	}
}

func TestCondensedStageDirectionUsesTightLeadInSpacing(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	line := &ast.SectionLine{
		Content: []ast.Inline{&ast.TextNode{Value: "A line of text."}},
	}
	require.NoError(t, r.BeginSectionLine(line))
	require.NoError(t, r.RenderText(line.Content[0].(*ast.TextNode)))
	require.NoError(t, r.EndSectionLine(line))

	yBefore := r.pdf.GetY()
	require.NoError(t, r.BeginStageDirection(&ast.StageDirection{
		Content: []ast.Inline{&ast.TextNode{Value: "He crosses to the window."}},
	}))

	assert.InDelta(t, r.condensedSmallGap(), r.pdf.GetY()-yBefore, 0.01)
}

func TestStandardCalloutSetsIndentedLeftMargin(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	require.NoError(t, r.BeginCallout(&ast.Callout{
		Content: []ast.Inline{&ast.TextNode{Value: "Midwinter. The room has not been heated for days."}},
	}))

	left, _, _, _ := r.pdf.GetMargins()
	assert.Greater(t, left, r.marginL)

	require.NoError(t, r.EndCallout(&ast.Callout{}))
	left, _, _, _ = r.pdf.GetMargins()
	assert.InDelta(t, r.marginL, left, 0.01)
}

func TestStageDirectionNestedItalicPreservesOuterItalic(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	stageDirection := &ast.StageDirection{
		Content: []ast.Inline{
			&ast.TextNode{Value: "Not performing humanity, not approximating it--"},
			&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "inhabiting"}}},
			&ast.TextNode{Value: " it."},
		},
	}

	require.NoError(t, r.BeginStageDirection(stageDirection))
	require.Equal(t, "I", r.fontStyle)
	require.NoError(t, r.RenderText(stageDirection.Content[0].(*ast.TextNode)))
	require.NoError(t, r.BeginItalic(stageDirection.Content[1].(*ast.ItalicNode)))
	require.Equal(t, "I", r.fontStyle)
	require.NoError(t, r.RenderText(stageDirection.Content[1].(*ast.ItalicNode).Content[0].(*ast.TextNode)))
	require.NoError(t, r.EndItalic(stageDirection.Content[1].(*ast.ItalicNode)))
	assert.Equal(t, "I", r.fontStyle)
	require.NoError(t, r.RenderText(stageDirection.Content[2].(*ast.TextNode)))
	require.NoError(t, r.EndStageDirection(stageDirection))
	assert.Equal(t, "", r.fontStyle)
}

func TestDialogueParentheticalNestedFormattingPreservesOuterItalic(t *testing.T) {
	r := NewRenderer(render.DefaultConfig()).(*pdfRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	inlines := []ast.Inline{
		&ast.TextNode{Value: "offstage, "},
		&ast.UnderlineNode{Content: []ast.Inline{&ast.TextNode{Value: "exasperated"}}},
		&ast.TextNode{Value: "; "},
		&ast.BoldNode{Content: []ast.Inline{&ast.TextNode{Value: "overlapping"}}},
	}

	r.setStyle("I")
	r.pdf.Write(r.lineHeight, "(")
	require.NoError(t, r.renderInlineContent(inlines))
	assert.Equal(t, "I", r.fontStyle)
	r.pdf.Write(r.lineHeight, ")")
	r.setStyle("")
	assert.Equal(t, "", r.fontStyle)
}

func TestCondensedCalloutUsesParagraphGapAfterPreviousCallout(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	require.NoError(t, r.BeginCallout(&ast.Callout{
		Content: []ast.Inline{&ast.TextNode{Value: "First callout."}},
	}))
	require.NoError(t, r.EndCallout(&ast.Callout{}))

	yBefore := r.pdf.GetY()
	require.NoError(t, r.BeginCallout(&ast.Callout{
		Content: []ast.Inline{&ast.TextNode{Value: "Second callout."}},
	}))

	assert.InDelta(t, r.condensedSmallGap(), r.pdf.GetY()-yBefore, 0.01)
}

func TestCondensedDialogueLongParentheticalWrapsDuringRendering(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	inlines := []ast.Inline{
		&ast.TextNode{Value: "fewer voices than before; but the ones who say it, mean it and keep saying it long after the room should have gone quiet"},
	}
	runs, err := r.captureInlineRuns(inlines, "I")
	require.NoError(t, err)
	runs = append([]dialogueTextRun{{text: "(", style: "I"}}, runs...)
	runs = append(runs, dialogueTextRun{text: ")", style: "I"})

	startX := r.measureTextWidth("B", "ALL.") + r.measureTextWidth("", "  ")
	yBefore := r.pdf.GetY()
	xAfter := r.renderWrappedStyledRuns(startX, runs, r.bodyW)

	assert.Greater(t, r.pdf.GetY(), yBefore)
	assert.LessOrEqual(t, xAfter, r.bodyW)
}

func TestCondensedDialogueExactReportedParentheticalWraps(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	dialogue := &ast.Dialogue{
		Character:     "ALL",
		Parenthetical: "(fewer voices than before; but the ones who say it, mean it)",
		Lines: []ast.DialogueLine{
			{Content: []ast.Inline{&ast.TextNode{Value: "Thank Bob."}}},
		},
	}

	yBefore := r.pdf.GetY()
	require.NoError(t, r.BeginDialogue(dialogue))
	require.NoError(t, r.BeginDialogueLine(&dialogue.Lines[0]))
	require.NoError(t, r.RenderText(dialogue.Lines[0].Content[0].(*ast.TextNode)))
	require.NoError(t, r.EndDialogueLine(&dialogue.Lines[0]))
	require.NoError(t, r.EndDialogue(dialogue))

	assert.Greater(t, r.pdf.GetY(), yBefore+r.lineHeight)
}

func TestCondensedPrepareDialogueLinesOnlyNarrowsFirstVisualLine(t *testing.T) {
	r := NewCondensedRenderer(render.DefaultConfig()).(*condensedRenderer)
	require.NoError(t, r.BeginDocument(&ast.Document{}, &bytes.Buffer{}))

	text := "As I record this, we are celebrating the very first in what I hope will become an enduring tradition: Ascension Day. Today marks twenty-five years since our departure from Earth, and it's hard to believe that an entire generation of children born here aboard Nemus Dianae will begin their adulthood, never having known the world I was born into."
	lines := []bufferedDialogueLine{{runs: []dialogueTextRun{{text: text, style: ""}}}}

	prepared := r.prepareDialogueLines(lines, 45)
	require.Len(t, prepared, 1)

	narrowAllLines := r.pdf.SplitText(text, 45)
	assert.Less(t, len(prepared[0].wrappedText), len(narrowAllLines))
}
