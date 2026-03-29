package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyDocument(t *testing.T) {
	doc, errs := Parse([]byte(""))
	require.Empty(t, errs)
	assert.NotNil(t, doc)
	assert.Nil(t, doc.TitlePage)
	assert.Empty(t, doc.Body)
}

func TestTitlePageOnly(t *testing.T) {
	input := `Title: Hamlet
Author: William Shakespeare
Draft: First`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.NotNil(t, doc.TitlePage)
	assert.Len(t, doc.TitlePage.Entries, 3)
	assert.Equal(t, "Title", doc.TitlePage.Entries[0].Key)
	assert.Equal(t, "Hamlet", doc.TitlePage.Entries[0].Value)
	assert.Equal(t, "Author", doc.TitlePage.Entries[1].Key)
	assert.Equal(t, "William Shakespeare", doc.TitlePage.Entries[1].Value)
}

func TestDramatisPersonae(t *testing.T) {
	input := `# Dramatis Personae
## The Royals
KING LEAR — King of Britain
CORDELIA — Youngest daughter
## The Earls
KENT — A loyal earl`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	dp := ast.FindDramatisPersonae(doc.Body)
	require.NotNil(t, dp, "expected Dramatis Personae section in body")
	assert.Len(t, dp.Groups, 2)
	assert.Equal(t, "The Royals", dp.Groups[0].Name)
	assert.Len(t, dp.Groups[0].Characters, 2)
	assert.Equal(t, "KING LEAR", dp.Groups[0].Characters[0].Name)
	assert.Equal(t, "King of Britain", dp.Groups[0].Characters[0].Description)
	assert.Equal(t, "The Earls", dp.Groups[1].Name)
	assert.Len(t, dp.Groups[1].Characters, 1)
}

func TestActsAndScenes(t *testing.T) {
	input := `# Act One

## Scene 1

HAMLET
To be or not to be.

## Scene 2

HORATIO
Good my lord.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	act, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok, "expected Section, got %T", doc.Body[0])
	assert.Equal(t, ast.SectionAct, act.Kind)

	// Count scene children
	var scenes []*ast.Section
	for _, child := range act.Children {
		if s, ok := child.(*ast.Section); ok && s.Kind == ast.SectionScene {
			scenes = append(scenes, s)
		}
	}
	assert.Len(t, scenes, 2)
	assert.Equal(t, "1", scenes[0].Number)
	assert.Equal(t, "2", scenes[1].Number)
}

func TestDialogue(t *testing.T) {
	input := `# Play

HAMLET
To be or not to be,
that is the question.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.NotEmpty(t, doc.Body)

	// Find the dialogue node
	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg, "expected to find Dialogue node")
	assert.Equal(t, "HAMLET", dlg.Character)
	assert.Len(t, dlg.Lines, 2)
}

func TestBodyOnlyDocument(t *testing.T) {
	input := "ALICE\nHello, world!"

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	assert.Nil(t, doc.TitlePage)
	require.Len(t, doc.Body, 1)

	dlg, ok := doc.Body[0].(*ast.Dialogue)
	require.True(t, ok)
	assert.Equal(t, "ALICE", dlg.Character)
}

func TestDialogueWithVerse(t *testing.T) {
	input := `# Play

HAMLET
  To be or not to be,
  that is the question.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	assert.Equal(t, "HAMLET", dlg.Character)
	for _, line := range dlg.Lines {
		assert.True(t, line.IsVerse, "expected verse line")
	}
}

func TestDialogueWithParenthetical(t *testing.T) {
	input := `# Play

HAMLET
(aside)
To be or not to be.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	assert.Equal(t, "(aside)", dlg.Parenthetical)
	assert.Equal(t, 3, dlg.ParentheticalRange().Start.Line)
	assert.Equal(t, 0, dlg.ParentheticalRange().Start.Column)
	assert.Equal(t, 7, dlg.ParentheticalRange().End.Column)
	assert.Len(t, dlg.Lines, 1)
}

func TestForcedCharacter(t *testing.T) {
	input := `# Play

@narrator
Once upon a time.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	assert.Equal(t, "narrator", dlg.Character)
}

func TestDualDialogue(t *testing.T) {
	input := `BRICK
Screw retirement.

STEEL ^
Screw retirement.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	dual, ok := doc.Body[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node")
	assert.Equal(t, "BRICK", dual.Left.Character)
	assert.Equal(t, "STEEL", dual.Right.Character)
	assert.Len(t, dual.Left.Lines, 1)
	assert.Len(t, dual.Right.Lines, 1)
}

func TestDualDialogueWithParenthetical(t *testing.T) {
	input := `BRICK
(angry)
Screw retirement.

STEEL ^
(calmly)
Screw retirement.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	dual, ok := doc.Body[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node")
	assert.Equal(t, "(angry)", dual.Left.Parenthetical)
	assert.Equal(t, "(calmly)", dual.Right.Parenthetical)
}

func TestDualDialogueForcedCharacter(t *testing.T) {
	input := `BRICK
Hello.

@narrator ^
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	dual, ok := doc.Body[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node")
	assert.Equal(t, "BRICK", dual.Left.Character)
	assert.Equal(t, "narrator", dual.Right.Character)
}

func TestDualDialogueInScene(t *testing.T) {
	input := `## ACT I

### SCENE 1

BRICK
Hello.

STEEL ^
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	// Navigate into act > scene > children
	act, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.NotEmpty(t, act.Children)

	scene, ok := act.Children[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, scene.Children, 1)

	dual, ok := scene.Children[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node in scene")
	assert.Equal(t, "BRICK", dual.Left.Character)
	assert.Equal(t, "STEEL", dual.Right.Character)
}

func TestDualDialogueWithoutPreviousDialogueBecomesStandalone(t *testing.T) {
	input := `# Play

## SCENE 1

> Thunder.

STEEL ^
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	section, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, section.Children, 1)

	scene, ok := section.Children[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, scene.Children, 2)

	_, ok = scene.Children[0].(*ast.StageDirection)
	require.True(t, ok)

	dialogue, ok := scene.Children[1].(*ast.Dialogue)
	require.True(t, ok, "expected standalone dialogue when no preceding dialogue exists")
	assert.Equal(t, "STEEL", dialogue.Character)
}

func TestDualDialogueDoesNotPairAcrossGenericSectionProse(t *testing.T) {
	input := `# Notes
BRICK
Hello.

This line stays prose.

STEEL ^
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	section, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, section.Children, 2)

	_, ok = section.Children[0].(*ast.Dialogue)
	require.True(t, ok)
	_, ok = section.Children[1].(*ast.Dialogue)
	require.True(t, ok, "expected second dialogue to remain standalone")
}

func TestInlineCharacterAlias(t *testing.T) {
	input := `# Dramatis Personae
JAMES/JIM — Her estranged son

# Play

JIM
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	dp := ast.FindDramatisPersonae(doc.Body)
	require.NotNil(t, dp)
	require.Len(t, dp.Characters, 1)
	assert.Equal(t, "JAMES", dp.Characters[0].Name)
	assert.Equal(t, []string{"JIM"}, dp.Characters[0].Aliases)
}

func TestStandaloneCharacterAlias(t *testing.T) {
	input := `# Dramatis Personae
JAMES — Her estranged son
[JAMES/JIM]`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	dp := ast.FindDramatisPersonae(doc.Body)
	require.NotNil(t, dp)
	require.Len(t, dp.Characters, 1)
	assert.Equal(t, "JAMES", dp.Characters[0].Name)
	assert.Equal(t, []string{"JIM"}, dp.Characters[0].Aliases)
}

func TestHeadingPrefixesRemainGeneric(t *testing.T) {
	input := `# Actor Notes
This is not an act.

# Scenery
This is not a scene.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 2)

	for i, node := range doc.Body {
		section, ok := node.(*ast.Section)
		require.True(t, ok, "node %d should be a section", i)
		assert.Equal(t, ast.SectionGeneric, section.Kind)
	}
}

func TestStandaloneStageDirection(t *testing.T) {
	input := `# Play

## ACT I

> The curtain rises.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var sd *ast.StageDirection
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if s, ok := c.(*ast.StageDirection); ok {
					sd = s
					break
				}
				if inner, ok := c.(*ast.Section); ok {
					for _, ic := range inner.Children {
						if s, ok := ic.(*ast.StageDirection); ok {
							sd = s
							break
						}
					}
				}
			}
		}
	}
	require.NotNil(t, sd, "expected StageDirection node")
}

func TestSong(t *testing.T) {
	input := `# Play

SONG 1 My Song
HAMLET
La la la.
SONG END`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var song *ast.Song
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if s, ok := c.(*ast.Song); ok {
					song = s
					break
				}
			}
		}
	}
	require.NotNil(t, song, "expected Song node")
	assert.Equal(t, "1", song.Number)
	assert.Equal(t, "My Song", song.Title)
}

func TestLineComment(t *testing.T) {
	input := `# Play

// This is a comment`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var comment *ast.Comment
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if cm, ok := c.(*ast.Comment); ok {
					comment = cm
					break
				}
			}
		}
	}
	require.NotNil(t, comment)
	assert.Equal(t, "This is a comment", comment.Text)
	assert.False(t, comment.Block)
}

func TestBlockComment(t *testing.T) {
	input := `# Play

/* This is
a block comment */`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var comment *ast.Comment
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if cm, ok := c.(*ast.Comment); ok {
					comment = cm
					break
				}
			}
		}
	}
	require.NotNil(t, comment)
	assert.True(t, comment.Block)
}

func TestPageBreak(t *testing.T) {
	input := `# Play

===`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var pb *ast.PageBreak
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if p, ok := c.(*ast.PageBreak); ok {
					pb = p
					break
				}
			}
		}
	}
	require.NotNil(t, pb, "expected PageBreak node")
}

func TestErrorRecovery(t *testing.T) {
	// Unterminated song should produce an error but not panic
	input := `# Play

SONG 1
HAMLET
La la la.`

	doc, errs := Parse([]byte(input))
	assert.NotNil(t, doc, "document should not be nil even with errors")
	assert.NotEmpty(t, errs, "expected at least one error for unterminated SONG")
}

func TestNoTitlePage(t *testing.T) {
	input := `# Act One

HAMLET
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	assert.Nil(t, doc.TitlePage)
	assert.NotEmpty(t, doc.Body)
}

func TestInlineFormatting(t *testing.T) {
	input := `# Play

HAMLET
This is **bold** and *italic* text.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	require.NotEmpty(t, dlg.Lines)

	// The line should have multiple inline nodes
	line := dlg.Lines[0]
	assert.True(t, len(line.Content) > 1, "expected multiple inline nodes for formatted text")

	// Find bold and italic nodes
	var hasBold, hasItalic bool
	for _, inline := range line.Content {
		switch inline.(type) {
		case *ast.BoldNode:
			hasBold = true
		case *ast.ItalicNode:
			hasItalic = true
		}
	}
	assert.True(t, hasBold, "expected bold node")
	assert.True(t, hasItalic, "expected italic node")
}

func TestVerseBlock(t *testing.T) {
	input := `# Play

  Roses are red,
  Violets are blue.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var vb *ast.VerseBlock
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if v, ok := c.(*ast.VerseBlock); ok {
					vb = v
					break
				}
			}
		}
	}
	require.NotNil(t, vb, "expected VerseBlock")
	assert.Len(t, vb.Lines, 2)
}

func TestCompleteDocument(t *testing.T) {
	input := `Title: A Test Play
Author: Test Author

# Dramatis Personae
## Main Characters
HAMLET — Prince of Denmark
HORATIO — Friend to Hamlet

# Act One

## Scene 1

(The stage is dark.)

HAMLET
(aside)
To be or not to be,
  that is the question.

// End of soliloquy

===`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.NotNil(t, doc.TitlePage)
	require.NotEmpty(t, doc.Body)

	assert.Equal(t, "A Test Play", doc.TitlePage.Entries[0].Value)

	dp := ast.FindDramatisPersonae(doc.Body)
	require.NotNil(t, dp, "expected Dramatis Personae in body")
	assert.Len(t, dp.Groups, 1)
	assert.Len(t, dp.Groups[0].Characters, 2)
}

func TestInlineRanges_ArePrecise(t *testing.T) {
	input := `# Play

HAMLET
This is **bold** and *italic* text.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	require.NotEmpty(t, dlg.Lines)

	line := dlg.Lines[0]

	var bold *ast.BoldNode
	var italic *ast.ItalicNode
	for _, inline := range line.Content {
		switch v := inline.(type) {
		case *ast.BoldNode:
			bold = v
		case *ast.ItalicNode:
			italic = v
		}
	}

	require.NotNil(t, bold)
	require.NotNil(t, italic)

	assert.Equal(t, 8, bold.Range.Start.Column)
	assert.Equal(t, 16, bold.Range.End.Column)
	assert.Equal(t, 21, italic.Range.Start.Column)
	assert.Equal(t, 29, italic.Range.End.Column)
}

func TestVerseInlineRanges_SkipIndentation(t *testing.T) {
	input := `# Play

HAMLET
  *Aside*`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	require.Len(t, dlg.Lines, 1)

	italic, ok := dlg.Lines[0].Content[0].(*ast.ItalicNode)
	require.True(t, ok)
	assert.Equal(t, 2, italic.Range.Start.Column)
	assert.Equal(t, 9, italic.Range.End.Column)
}

// Golden file tests
func TestGoldenFiles(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.ds")
	require.NoError(t, err)

	for _, dsFile := range matches {
		name := filepath.Base(dsFile)
		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile(dsFile)
			require.NoError(t, err)

			doc, errs := Parse(input)
			require.Empty(t, errs, "unexpected parse errors: %v", errs)
			require.NotNil(t, doc)

			goldenFile := dsFile[:len(dsFile)-3] + ".golden.json"
			actual, err := json.MarshalIndent(doc, "", "  ")
			require.NoError(t, err)

			if os.Getenv("UPDATE_GOLDEN") != "" {
				err = os.WriteFile(goldenFile, actual, 0o644)
				require.NoError(t, err)
				return
			}

			expected, err := os.ReadFile(goldenFile)
			require.NoError(t, err, "golden file %s not found; run with UPDATE_GOLDEN=1 to create", goldenFile)

			assert.JSONEq(t, string(expected), string(actual))
		})
	}
}

// findDialogue recursively searches body nodes for a Dialogue node.
func findDialogue(nodes []ast.Node, result **ast.Dialogue) {
	for _, n := range nodes {
		if d, ok := n.(*ast.Dialogue); ok {
			*result = d
			return
		}
		if sect, ok := n.(*ast.Section); ok {
			findDialogue(sect.Children, result)
			if *result != nil {
				return
			}
		}
		if song, ok := n.(*ast.Song); ok {
			findDialogue(song.Content, result)
			if *result != nil {
				return
			}
		}
	}
}
