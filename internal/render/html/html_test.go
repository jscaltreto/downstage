package html

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

func renderHTML(t *testing.T, doc *ast.Document, style ...render.Style) string {
	t.Helper()
	cfg := render.DefaultConfig()
	if len(style) > 0 {
		cfg.Style = style[0]
	}
	r := NewRenderer(cfg)
	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	return buf.String()
}

func TestRender_EmptyDocument(t *testing.T) {
	out := renderHTML(t, &ast.Document{})

	assert.True(t, strings.HasPrefix(out, "<!DOCTYPE html>"))
	assert.Contains(t, out, "<html lang=\"en\">")
	assert.Contains(t, out, "<meta charset=\"utf-8\">")
	assert.Contains(t, out, "<title>Untitled</title>")
	assert.Contains(t, out, "<div class=\"downstage-document\">")
	assert.Contains(t, out, "</div>\n</body>\n</html>")
	assert.Contains(t, out, "<style>")
	assert.Contains(t, out, "<meta name=\"generator\" content=\"Downstage\">")
}

func TestRender_TitlePage(t *testing.T) {
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "My Play"},
				{Key: "Subtitle", Value: "A Drama"},
				{Key: "Author", Value: "Jane Doe"},
				{Key: "Date", Value: "2025"},
				{Key: "Draft", Value: "First"},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<title>My Play</title>")
	assert.Contains(t, out, "<meta name=\"author\" content=\"Jane Doe\">")
	assert.Contains(t, out, "<header class=\"downstage-title-page\">")
	assert.Contains(t, out, "<h1>My Play</h1>")
	assert.Contains(t, out, "<p class=\"subtitle\">A Drama</p>")
	assert.Contains(t, out, "<p class=\"author\">Jane Doe</p>")
	assert.Contains(t, out, "<dt>Date</dt>")
	assert.Contains(t, out, "<dd>2025</dd>")
	assert.Contains(t, out, "<dt>Draft</dt>")
	assert.Contains(t, out, "<dd>First</dd>")
	assert.Contains(t, out, "</header>")
}

func TestRender_TitlePageDuplicateTitleSubtitleUsesLastWins(t *testing.T) {
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "First Title"},
				{Key: "Title", Value: "Final Title"},
				{Key: "Subtitle", Value: "First Subtitle"},
				{Key: "Subtitle", Value: "Final Subtitle"},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<h1>Final Title</h1>")
	assert.NotContains(t, out, "<h1>First Title</h1>")
	assert.Contains(t, out, "<p class=\"subtitle\">Final Subtitle</p>")
	assert.NotContains(t, out, "<p class=\"subtitle\">First Subtitle</p>")
	assert.Contains(t, out, "<title>Final Title</title>")
}

func TestRender_TitlePageSupportsMultipleAuthors(t *testing.T) {
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "My Play"},
				{Key: "Author", Value: "Jane Doe"},
				{Key: "Author", Value: "John Smith"},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"author\">Jane Doe</p>")
	assert.Contains(t, out, "<p class=\"author\">John Smith</p>")
}

func TestRender_TopLevelSectionMetadataAsTitlePage(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "My Play",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{
						{Key: "Author", Value: "Jane Doe"},
						{Key: "Subtitle", Value: "A Drama"},
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

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<title>My Play</title>")
	assert.Contains(t, out, "<header class=\"downstage-title-page\">")
	assert.Contains(t, out, "<h1>My Play</h1>")
	assert.Contains(t, out, "<p class=\"subtitle\">A Drama</p>")
	assert.Contains(t, out, "<p class=\"author\">Jane Doe</p>")
}

func TestRender_TopLevelSectionWithoutMetadataStillUsesTitlePage(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "My Play",
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

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<title>My Play</title>")
	assert.Contains(t, out, "<header class=\"downstage-title-page\">")
	assert.Contains(t, out, "<h1>My Play</h1>")
}

func TestRender_CompilationSubplaysUseInlineHeaders(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "My Compilation",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Author", Value: "Editor"}},
				},
			},
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "First Play",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Author", Value: "Jane Doe"}},
				},
				Children: []ast.Node{
					&ast.Section{
						Kind:  ast.SectionDramatisPersonae,
						Level: 2,
						Characters: []ast.Character{
							{Name: "ALICE"},
						},
					},
					&ast.Section{Kind: ast.SectionAct, Level: 2, Number: "I"},
				},
			},
		},
	}

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<header class=\"downstage-title-page\">")
	assert.Contains(t, out, "<section class=\"downstage-subplay\">")
	assert.Contains(t, out, "<header class=\"downstage-subplay-header\">")
	assert.Contains(t, out, "<h1>First Play</h1>")
	assert.Contains(t, out, "<p class=\"downstage-subplay-author\">Jane Doe</p>")
	assert.Contains(t, out, "downstage-dramatis-personae downstage-dramatis-personae-inline")
	assert.NotContains(t, out, "<title>First Play</title>")
}

func TestRender_SubplayHeaderPrefersMetadataTitle(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Collection",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Author", Value: "Editor"}},
				},
			},
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Hamlet draft",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{
						{Key: "Title", Value: "Hamlet"},
						{Key: "Author", Value: "Shakespeare"},
					},
				},
				Children: []ast.Node{
					&ast.Section{Kind: ast.SectionAct, Level: 2, Number: "I"},
				},
			},
		},
	}

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<h1>Hamlet</h1>")
	assert.NotContains(t, out, "<h1>Hamlet draft</h1>")
}

func TestRender_SinglePlayTitleMetadataSuppressesHeadingDuplication(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Hamlet draft",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{
						{Key: "Title", Value: "Hamlet"},
					},
				},
				Children: []ast.Node{
					&ast.Section{Kind: ast.SectionAct, Level: 2, Number: "I"},
				},
			},
		},
	}

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<h1>Hamlet</h1>")
	// Heading text must not leak into the body as a second h1.
	assert.Equal(t, 1, strings.Count(out, "<h1>"), "expected only the title-page h1")
}

func TestRender_SubplayRendersSubtitle(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Collection",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Author", Value: "Editor"}},
				},
			},
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Act of Kindness",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{
						{Key: "Subtitle", Value: "A short tragedy"},
						{Key: "Author", Value: "Jane Doe"},
					},
				},
				Children: []ast.Node{
					&ast.Section{Kind: ast.SectionScene, Level: 2, Number: "1"},
				},
			},
		},
	}

	out := renderHTML(t, doc)
	assert.Contains(t, out, `<p class="downstage-subplay-subtitle">A short tragedy</p>`)
}

func TestRender_CompilationSubplaySupportsMultipleAuthors(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "Compilation",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{{Key: "Editor", Value: "Person"}},
				},
			},
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 1,
				Title: "First Play",
				Metadata: &ast.TitlePage{
					Entries: []ast.KeyValue{
						{Key: "Author", Value: "Jane Doe"},
						{Key: "Author", Value: "John Smith"},
					},
				},
				Children: []ast.Node{
					&ast.Section{Kind: ast.SectionAct, Level: 2, Number: "I"},
				},
			},
		},
	}

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<p class=\"downstage-subplay-author\">Jane Doe</p>")
	assert.Contains(t, out, "<p class=\"downstage-subplay-author\">John Smith</p>")
}

func TestRender_DramatisPersonaeIncludesAliases(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionDramatisPersonae,
				Level: 2,
				Characters: []ast.Character{
					{Name: "HAMLET", Aliases: []string{"HAM"}, Description: "Prince of Denmark"},
				},
			},
		},
	}

	out := renderHTML(t, doc)
	assert.Contains(t, out, "<dt>HAMLET/HAM</dt>")
	assert.Contains(t, out, "<dd>Prince of Denmark</dd>")
}

func TestRender_DramatisPersonaeEmDashSeparatorPreservesTrailingSpace(t *testing.T) {
	// The CSS escape "\2014 " swallows a single trailing space. The fix uses
	// a non-breaking space after the em-dash; guard that in the stylesheet
	// so a regression in css.go fails loudly.
	out := renderHTML(t, &ast.Document{})
	assert.Contains(t, out, `content: " \2014\00a0"`)
}

func TestRender_ForcedPageBreakHasVisibleRule(t *testing.T) {
	out := renderHTML(t, &ast.Document{})
	assert.Contains(t, out, ".downstage-page-break")
	assert.Contains(t, out, "border-top: 1px dashed var(--downstage-break-color)")
}

func TestRender_CompilationBreaksMatchPDF(t *testing.T) {
	// When a title-page-only section precedes a subplay, only one visible
	// rule should appear — the title page's bottom border. The subplay's
	// top border rule is suppressed via `.downstage-title-page + .downstage-subplay`.
	// When the subplay contains an inline DP followed by a scene, the scene
	// itself draws the break (matching PDF's deferred page advance into the
	// first body element of the subplay).
	out := renderHTML(t, &ast.Document{})
	assert.Contains(t, out, ".downstage-title-page + .downstage-subplay")
	assert.Contains(t, out, ".downstage-subplay > .downstage-act:first-of-type")
	assert.Contains(t, out, ".downstage-subplay > .downstage-scene:first-of-type")
}

func TestRender_DialogueWithFormatting(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character:     "HAMLET",
				Parenthetical: "aside",
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
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<div class=\"downstage-dialogue\">")
	assert.Contains(t, out, "<p class=\"downstage-character\">HAMLET</p>")
	assert.Contains(t, out, "<p class=\"downstage-parenthetical\">(aside)</p>")
	assert.Contains(t, out, "<p class=\"downstage-line\">")
	assert.Contains(t, out, "To be, or <strong>not</strong> to be.")
	assert.Contains(t, out, "</div>")
}

func TestRender_DialogueParentheticalWithFormatting(t *testing.T) {
	parenthetical := []ast.Inline{
		&ast.TextNode{Value: "offstage, "},
		&ast.UnderlineNode{Content: []ast.Inline{&ast.TextNode{Value: "exasperated"}}},
		&ast.TextNode{Value: "; "},
		&ast.BoldNode{Content: []ast.Inline{&ast.TextNode{Value: "overlapping"}}},
		&ast.TextNode{Value: " NOTBOB"},
	}
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character:     "GUARD",
				Parenthetical: "(offstage, _exasperated_; **overlapping** NOTBOB)",
				Lines: []ast.DialogueLine{
					{Content: []ast.Inline{&ast.TextNode{Value: "Hold there."}}},
				},
			},
		},
	}
	doc.Body[0].(*ast.Dialogue).SetParentheticalInlines(parenthetical)
	out := renderHTML(t, doc)

	assert.Contains(t, out, `<p class="downstage-parenthetical">(offstage, <u>exasperated</u>; <strong>overlapping</strong> NOTBOB)</p>`)
}

func TestRender_InlineDirectionWithFormatting(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "GIDEON",
				Lines: []ast.DialogueLine{
					{Content: []ast.Inline{
						&ast.InlineDirectionNode{Content: []ast.Inline{
							&ast.TextNode{Value: "beat; "},
							&ast.UnderlineNode{Content: []ast.Inline{&ast.TextNode{Value: "almost"}}},
							&ast.TextNode{Value: " fighting the impulse to continue."},
						}},
					}},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, `<span class="downstage-inline-direction">(beat; <u>almost</u> fighting the impulse to continue.)</span>`)
}

func TestRender_DualDialogue(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.DualDialogue{
				Left: &ast.Dialogue{
					Character: "ALICE",
					Lines: []ast.DialogueLine{
						{Content: []ast.Inline{&ast.TextNode{Value: "Left side."}}},
					},
				},
				Right: &ast.Dialogue{
					Character: "BOB",
					Lines: []ast.DialogueLine{
						{Content: []ast.Inline{&ast.TextNode{Value: "Right side."}}},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<div class=\"downstage-dual-dialogue\">")
	assert.Contains(t, out, "ALICE")
	assert.Contains(t, out, "BOB")
	assert.Contains(t, out, "Left side.")
	assert.Contains(t, out, "Right side.")

	dualBlock := extractBlock(t, out, "<div class=\"downstage-dual-dialogue\">", "</div>")
	assert.Equal(t, 2, strings.Count(dualBlock, "<div class=\"downstage-dialogue\">"))
	assert.Less(t, strings.Index(dualBlock, "ALICE"), strings.Index(dualBlock, "BOB"))
}

func TestRender_StageDirection(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.StageDirection{
				Content: []ast.Inline{
					&ast.TextNode{Value: "The lights dim slowly."},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"downstage-stage-direction\">The lights dim slowly.</p>")
}

func TestRender_StageDirectionWithNestedItalic(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.StageDirection{
				Content: []ast.Inline{
					&ast.TextNode{Value: "Not performing humanity, not approximating it--"},
					&ast.ItalicNode{Content: []ast.Inline{
						&ast.TextNode{Value: "inhabiting"},
					}},
					&ast.TextNode{Value: " it."},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"downstage-stage-direction\">Not performing humanity, not approximating it--<em>inhabiting</em> it.</p>")
}

func TestRender_Callout(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Callout{
				Content: []ast.Inline{
					&ast.TextNode{Value: "Midwinter. The room has not been heated for days."},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"downstage-callout\">Midwinter. The room has not been heated for days.</p>")
}

func TestRender_Song(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Song{
				Number: "1",
				Title:  "Evening Song",
				Content: []ast.Node{
					&ast.Dialogue{
						Character: "HAMLET",
						Lines: []ast.DialogueLine{
							{Content: []ast.Inline{&ast.TextNode{Value: "Sing me a song."}}},
						},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<div class=\"downstage-song\">")
	assert.Contains(t, out, "<h4>SONG 1: Evening Song</h4>")
	assert.Contains(t, out, "<p class=\"downstage-song-end\">SONG END</p>")
	assert.Contains(t, out, "Sing me a song.")
}

func TestRender_DramatisPersonae(t *testing.T) {
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
						},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<section class=\"downstage-dramatis-personae\">")
	assert.Contains(t, out, "<h2>DRAMATIS PERSONAE</h2>")
	assert.Contains(t, out, "<dt>HAMLET</dt>")
	assert.Contains(t, out, "<dd>Prince of Denmark</dd>")
	assert.Contains(t, out, "<dt>HORATIO</dt>")
	assert.Contains(t, out, "<p class=\"downstage-character-group-name\">Courtiers</p>")
	assert.Contains(t, out, "<dt>ROSENCRANTZ</dt>")
}

func TestRender_DramatisPersonaeInlineStyling(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind: ast.SectionDramatisPersonae,
				Characters: []ast.Character{
					{
						Name:        "GHOST",
						Description: "A *spectre* in _ragged_ attire",
						DescriptionInlines: []ast.Inline{
							&ast.TextNode{Value: "A "},
							&ast.BoldNode{Content: []ast.Inline{&ast.TextNode{Value: "spectre"}}},
							&ast.TextNode{Value: " in "},
							&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "ragged"}}},
							&ast.TextNode{Value: " attire"},
						},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<dd>A <strong>spectre</strong> in <em>ragged</em> attire</dd>")
	assert.NotContains(t, out, "*spectre*")
	assert.NotContains(t, out, "_ragged_")
}

func TestRender_TitlePageSubtitleInlineStyling(t *testing.T) {
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "A Play"},
				{
					Key:   "Subtitle",
					Value: "A tragedy in _five_ acts",
					ValueInlines: []ast.Inline{
						&ast.TextNode{Value: "A tragedy in "},
						&ast.ItalicNode{Content: []ast.Inline{&ast.TextNode{Value: "five"}}},
						&ast.TextNode{Value: " acts"},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"subtitle\">A tragedy in <em>five</em> acts</p>")
	assert.NotContains(t, out, "_five_")
}

func TestRender_PageBreak(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.StageDirection{Content: []ast.Inline{&ast.TextNode{Value: "Before."}}},
			&ast.PageBreak{Range: token.Range{}},
			&ast.StageDirection{Content: []ast.Inline{&ast.TextNode{Value: "After."}}},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<hr class=\"downstage-page-break\">")
	assert.Contains(t, out, "Before.")
	assert.Contains(t, out, "After.")
}

func TestRender_Sections(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:   ast.SectionAct,
				Number: "I",
				Title:  "The Beginning",
				Children: []ast.Node{
					&ast.Section{
						Kind:   ast.SectionScene,
						Number: "1",
						Title:  "Morning",
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<section class=\"downstage-act\">")
	assert.Contains(t, out, "<h2>ACT I: THE BEGINNING</h2>")
	assert.Contains(t, out, "<section class=\"downstage-scene\">")
	assert.Contains(t, out, "<h3>SCENE 1: MORNING</h3>")
}

func TestRender_InlineDirection(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "TEST",
				Lines: []ast.DialogueLine{
					{
						Content: []ast.Inline{
							&ast.TextNode{Value: "Hello "},
							&ast.InlineDirectionNode{Content: []ast.Inline{
								&ast.TextNode{Value: "turning to audience"},
							}},
							&ast.TextNode{Value: " goodbye."},
						},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<span class=\"downstage-inline-direction\">(turning to audience)</span>")
}

func TestRender_InlineDirectionNested(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "TEST",
				Lines: []ast.DialogueLine{
					{
						Content: []ast.Inline{
							&ast.InlineDirectionNode{Content: []ast.Inline{
								&ast.TextNode{Value: "outer "},
								&ast.InlineDirectionNode{Content: []ast.Inline{
									&ast.TextNode{Value: "inner"},
								}},
							}},
						},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	// Only the outermost direction should have parentheses
	assert.Contains(t, out, "(outer inner)")
	// Should not have double parens
	assert.NotContains(t, out, "((")
	assert.NotContains(t, out, "))")
}

func TestRender_VerseBlock(t *testing.T) {
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
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<div class=\"downstage-verse-block\">")
	assert.Contains(t, out, "<p class=\"downstage-verse-line\">Line one</p>")
	assert.Contains(t, out, "<p class=\"downstage-verse-line\">Line two</p>")
	assert.Contains(t, out, "</div>")
}

func TestRender_FullIntegration(t *testing.T) {
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
									&ast.TextNode{Value: "A dimly lit room."},
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
	out := renderHTML(t, doc)

	assert.True(t, len(out) > 500, "full integration HTML should have substantial content")
	assert.True(t, strings.HasPrefix(out, "<!DOCTYPE html>"))

	// Title page
	assert.Contains(t, out, "<h1>The Test Play</h1>")

	// DP
	assert.Contains(t, out, "DRAMATIS PERSONAE")
	assert.Contains(t, out, "<dt>ALICE</dt>")

	// Act/Scene
	assert.Contains(t, out, "ACT I")
	assert.Contains(t, out, "SCENE 1")

	// Dialogue
	assert.Contains(t, out, "ALICE")
	assert.Contains(t, out, "(entering)")
	assert.Contains(t, out, "<strong>world</strong>")

	// Song
	assert.Contains(t, out, "SONG 1: Evening Song")
	assert.Contains(t, out, "SONG END")

	// Page break
	assert.Contains(t, out, "downstage-page-break")

	// Stage direction
	assert.Contains(t, out, "A dimly lit room.")
}

func TestRender_CondensedStyle(t *testing.T) {
	doc := &ast.Document{}
	out := renderHTML(t, doc, render.StyleCondensed)

	// Condensed should use serif font stack
	assert.Contains(t, out, "Georgia")
	assert.Contains(t, out, "10pt")
	// Should NOT contain monospace-specific standard styles
	assert.NotContains(t, out, "Courier New")
}

func TestRender_TitlePageDedup(t *testing.T) {
	section := &ast.Section{
		Kind:  ast.SectionGeneric,
		Level: 1,
		Title: "My Play",
	}
	section.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "Opening note."}}})

	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "My Play"},
			},
		},
		Body: []ast.Node{
			section,
		},
	}
	out := renderHTML(t, doc)

	// The h1 in the title page should exist
	assert.Contains(t, out, "<h1>My Play</h1>")
	// But there should be no generic section with the same title
	assert.NotContains(t, out, "<section class=\"downstage-section\">")
	assert.Contains(t, out, "<p>Opening note. </p>")
}

func TestRender_HTMLEscaping(t *testing.T) {
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "A <Script> & \"Play\""},
			},
		},
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "O'BRIEN",
				Lines: []ast.DialogueLine{
					{Content: []ast.Inline{
						&ast.TextNode{Value: "Use <html> & \"quotes\" safely."},
					}},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "A &lt;Script&gt; &amp; &#34;Play&#34;")
	assert.Contains(t, out, "O&#39;BRIEN")
	assert.Contains(t, out, "Use &lt;html&gt; &amp; &#34;quotes&#34; safely.")
	// Must not contain unescaped HTML
	assert.NotContains(t, out, "<Script>")
}

func TestRender_InlineFormatting(t *testing.T) {
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
						},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<em>italic</em>")
	assert.Contains(t, out, "<strong><em>bold italic</em></strong>")
	assert.Contains(t, out, "<u>underline</u>")
	assert.Contains(t, out, "<del>strike</del>")
}

func TestRender_ForcedHeading(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionGeneric,
				Level: 0,
				Title: "The Next Evening",
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"downstage-forced-heading\"><strong>The Next Evening</strong></p>")
	// Forced heading should not open a section
	assert.NotContains(t, out, "<section class=\"downstage-section\">")
}

func TestRender_DialogueVerseLine(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "HAMLET",
				Lines: []ast.DialogueLine{
					{Content: []ast.Inline{&ast.TextNode{Value: "Prose line."}}, IsVerse: false},
					{Content: []ast.Inline{&ast.TextNode{Value: "Verse line."}}, IsVerse: true},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p class=\"downstage-line\">Prose line.</p>")
	assert.Contains(t, out, "<p class=\"downstage-line downstage-verse\">Verse line.</p>")
}

func TestRender_SongWithoutNumber(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Song{
				Title: "A Little Song",
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<h4>SONG: A Little Song</h4>")
}

func TestRender_ActWithoutNumber(t *testing.T) {
	// Act without number but with title page should omit the duplicate wrapper
	// while preserving its child content.
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Entries: []ast.KeyValue{
				{Key: "Title", Value: "Test"},
			},
		},
		Body: []ast.Node{
			&ast.Section{
				Kind:  ast.SectionAct,
				Title: "Test",
				Children: []ast.Node{
					&ast.StageDirection{
						Content: []ast.Inline{&ast.TextNode{Value: "Still render me."}},
					},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	// Unnumbered act wrapper should be omitted
	assert.NotContains(t, out, "<section class=\"downstage-act\">")
	assert.Contains(t, out, "Still render me.")
}

func TestRender_SectionLineParagraphs(t *testing.T) {
	// Section lines should group consecutive non-blank lines into <p> elements
	// and blank lines should separate paragraphs.
	sec := &ast.Section{
		Kind:  ast.SectionGeneric,
		Level: 1,
		Title: "Notes",
	}
	sec.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "First line."}}})
	sec.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "Second line."}}})
	sec.AppendLine(ast.SectionLine{Content: nil}) // blank line = paragraph break
	sec.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "Third line."}}})

	doc := &ast.Document{Body: []ast.Node{sec}}
	out := renderHTML(t, doc)

	// First two lines should be in one paragraph with reflow space
	assert.Contains(t, out, "<p>First line. Second line. </p>")
	// Blank line should separate paragraphs
	assert.Contains(t, out, "<div class=\"downstage-section-break\"></div>")
	// Third line in its own paragraph
	assert.Contains(t, out, "<p>Third line. </p>")
	// No nested <p> tags
	assert.NotContains(t, out, "<p><p>")
	assert.NotContains(t, out, "<p>First line. <p>")
}

func TestRender_SectionLineParagraphsCloseBeforeBlocks(t *testing.T) {
	sec := &ast.Section{
		Kind:  ast.SectionGeneric,
		Level: 1,
		Title: "Notes",
	}
	sec.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "First line."}}})
	sec.AppendChild(&ast.Dialogue{
		Character: "ALICE",
		Lines: []ast.DialogueLine{
			{Content: []ast.Inline{&ast.TextNode{Value: "Hello."}}},
		},
	})
	sec.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "Last line."}}})

	doc := &ast.Document{Body: []ast.Node{sec}}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p>First line. </p>\n<div class=\"downstage-dialogue\">")
	assert.Contains(t, out, "</div>\n<p>Last line. </p>")
	assert.NotContains(t, out, "<p>First line. <div class=\"downstage-dialogue\">")
}

func TestRender_ReusedRendererResetsState(t *testing.T) {
	r := NewRenderer(render.DefaultConfig())

	firstDoc := &ast.Document{
		Body: []ast.Node{
			&ast.StageDirection{Content: []ast.Inline{&ast.TextNode{Value: "__doc_one__"}}},
		},
	}
	secondDoc := &ast.Document{
		Body: []ast.Node{
			&ast.StageDirection{Content: []ast.Inline{&ast.TextNode{Value: "__doc_two__"}}},
		},
	}

	var first bytes.Buffer
	require.NoError(t, render.Walk(r, firstDoc, &first))
	assert.Contains(t, first.String(), "__doc_one__")

	var second bytes.Buffer
	require.NoError(t, render.Walk(r, secondDoc, &second))
	assert.Contains(t, second.String(), "__doc_two__")
	assert.NotContains(t, second.String(), "__doc_one__")
	assert.Equal(t, 1, strings.Count(second.String(), "<!DOCTYPE html>"))
}

func TestRender_SectionLineParagraphClosedAtEndSection(t *testing.T) {
	// A paragraph that's open when EndSection is called should be closed
	sec := &ast.Section{
		Kind:  ast.SectionGeneric,
		Level: 1,
		Title: "Notes",
	}
	sec.AppendLine(ast.SectionLine{Content: []ast.Inline{&ast.TextNode{Value: "Only line."}}})

	doc := &ast.Document{Body: []ast.Node{sec}}
	out := renderHTML(t, doc)

	assert.Contains(t, out, "<p>Only line. </p>")
	assert.Contains(t, out, "</section>")
}

func renderHTMLWithAnchors(t *testing.T, doc *ast.Document) string {
	t.Helper()
	cfg := render.DefaultConfig()
	cfg.SourceAnchors = true
	r := NewRenderer(cfg)
	var buf bytes.Buffer
	err := render.Walk(r, doc, &buf)
	require.NoError(t, err)
	return buf.String()
}

func TestRender_SourceAnchors(t *testing.T) {
	doc := &ast.Document{
		TitlePage: &ast.TitlePage{
			Range:   token.Range{Start: token.Position{Line: 0}},
			Entries: []ast.KeyValue{{Key: "Title", Value: "Test"}},
		},
		Body: []ast.Node{
			&ast.Section{
				Kind:   ast.SectionAct,
				Number: "I",
				Range:  token.Range{Start: token.Position{Line: 5}},
				Children: []ast.Node{
					&ast.Section{
						Kind:   ast.SectionScene,
						Number: "1",
						Range:  token.Range{Start: token.Position{Line: 7}},
						Children: []ast.Node{
							&ast.StageDirection{
								Range:   token.Range{Start: token.Position{Line: 9}},
								Content: []ast.Inline{&ast.TextNode{Value: "Lights up."}},
							},
							&ast.Dialogue{
								Character: "ALICE",
								Range:     token.Range{Start: token.Position{Line: 11}},
								Lines: []ast.DialogueLine{
									{Content: []ast.Inline{&ast.TextNode{Value: "Hello."}}},
								},
							},
						},
					},
				},
			},
		},
	}
	out := renderHTMLWithAnchors(t, doc)

	assert.Contains(t, out, `data-source-line="1"`)  // title page (line 0 -> 1)
	assert.Contains(t, out, `data-source-line="6"`)  // act (line 5 -> 6)
	assert.Contains(t, out, `data-source-line="8"`)  // scene (line 7 -> 8)
	assert.Contains(t, out, `data-source-line="10"`) // stage direction (line 9 -> 10)
	assert.Contains(t, out, `data-source-line="12"`) // dialogue (line 11 -> 12)
}

func TestRender_SourceAnchorsOff(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "ALICE",
				Range:     token.Range{Start: token.Position{Line: 5}},
				Lines: []ast.DialogueLine{
					{Content: []ast.Inline{&ast.TextNode{Value: "Hello."}}},
				},
			},
		},
	}
	out := renderHTML(t, doc)
	assert.NotContains(t, out, "data-source-line")
}

func TestRender_DialogueParagraphBreak(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Dialogue{
				Character: "HAMLET",
				Lines: []ast.DialogueLine{
					{Content: []ast.Inline{&ast.TextNode{Value: "To be or not to be."}}},
					{}, // paragraph break marker
					{Content: []ast.Inline{&ast.TextNode{Value: "That is the question."}}},
				},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, `<div class="downstage-dialogue-break"></div>`)
	assert.Contains(t, out, "To be or not to be.")
	assert.Contains(t, out, "That is the question.")
}

func TestRender_StageDirectionContinuation(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
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
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, `class="downstage-stage-direction downstage-continuation"`)
	assert.Contains(t, out, "Adjacent direction.")
	// Non-continuation directions should not have the continuation class
	assert.Contains(t, out, "Separated direction.")
	// CSS rules present
	assert.Contains(t, out, ".downstage-stage-direction.downstage-continuation")
}

func TestRender_CalloutContinuation(t *testing.T) {
	doc := &ast.Document{
		Body: []ast.Node{
			&ast.Callout{
				Content: []ast.Inline{&ast.TextNode{Value: "First callout."}},
			},
			&ast.Callout{
				Content:      []ast.Inline{&ast.TextNode{Value: "Adjacent callout."}},
				Continuation: true,
			},
			&ast.Callout{
				Content: []ast.Inline{&ast.TextNode{Value: "Separated callout."}},
			},
		},
	}
	out := renderHTML(t, doc)

	assert.Contains(t, out, `class="downstage-callout downstage-continuation"`)
	assert.Contains(t, out, "Adjacent callout.")
	assert.Contains(t, out, ".downstage-callout.downstage-continuation")
}

func extractBlock(t *testing.T, out, prefix, suffix string) string {
	t.Helper()
	start := strings.Index(out, prefix)
	require.NotEqual(t, -1, start)
	rest := out[start:]
	depth := 1
	for i := len(prefix); i < len(rest); {
		switch {
		case strings.HasPrefix(rest[i:], "<div"):
			depth++
			i += len("<div")
		case strings.HasPrefix(rest[i:], suffix):
			depth--
			i += len(suffix)
			if depth == 0 {
				return rest[:i]
			}
		default:
			i++
		}
	}
	t.Fatalf("unterminated block for prefix %q", prefix)
	return ""
}
