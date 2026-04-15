package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestV2TopLevelSectionMetadata(t *testing.T) {
	input := `# The First Play
Subtitle: A play in one act
Author: Your Name
Date: 2024
Draft: First

ALICE
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	section, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.NotNil(t, section.Metadata)
	require.Len(t, section.Metadata.Entries, 4)
	assert.Equal(t, "Subtitle", section.Metadata.Entries[0].Key)
	assert.Equal(t, "A play in one act", section.Metadata.Entries[0].Value)
	assert.Equal(t, "Author", section.Metadata.Entries[1].Key)
	assert.Equal(t, "Your Name", section.Metadata.Entries[1].Value)
}

func TestV2DocumentLevelMetadataProducesError(t *testing.T) {
	input := `Title: Old Format
Author: Someone

# Play`

	_, errs := Parse([]byte(input))
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "document-level metadata is a V1 pattern")
}

func TestV1TopLevelDramatisPersonaeProducesError(t *testing.T) {
	input := `# Dramatis Personae
ALICE - A curious young woman

# Play`

	_, errs := Parse([]byte(input))
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "top-level Dramatis Personae is a V1 pattern")
}

func TestV2BodyOnlyDocumentIsAccepted(t *testing.T) {
	input := `ALICE
Hello.`

	doc, errs := Parse([]byte(input))
	require.NotNil(t, doc)
	require.Empty(t, errs, "body-only documents without metadata are valid V2")

	require.Len(t, doc.Body, 1)
	dlg, ok := doc.Body[0].(*ast.Dialogue)
	require.True(t, ok)
	assert.Equal(t, "ALICE", dlg.Character)
}

func TestV2ScopedDramatisPersonae(t *testing.T) {
	input := `# The First Play
Author: Your Name

## Dramatis Personae
ALICE/A - A curious young woman
BOB - Her steadfast companion

## ACT I`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	play := doc.Body[0].(*ast.Section)
	dp := ast.FindDramatisPersonaeInSection(play)
	require.NotNil(t, dp)
	require.Len(t, dp.Characters, 2)
	assert.Equal(t, "ALICE", dp.Characters[0].Name)
	assert.Equal(t, []string{"A"}, dp.Characters[0].Aliases)
	assert.Equal(t, "A curious young woman", dp.Characters[0].Description)
}

func TestV2StandaloneAliasRejected(t *testing.T) {
	input := `# The First Play

## Dramatis Personae
ALICE - A curious young woman
[ALICE/A]
`

	doc, errs := Parse([]byte(input))
	require.NotNil(t, doc)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "standalone character alias syntax is not supported")
}

func TestV2UnicodeDashInDramatisPersonaeProducesError(t *testing.T) {
	input := `# Play

## Dramatis Personae
HAMLET — Prince of Denmark`

	doc, errs := Parse([]byte(input))
	require.NotNil(t, doc)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "must use ASCII ` - `")
}

func TestV1TitlePageOnlyProducesError(t *testing.T) {
	input := `Title: Hamlet
Author: William Shakespeare
Draft: First`

	_, errs := Parse([]byte(input))
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "document-level metadata is a V1 pattern")
}

func TestDramatisPersonae(t *testing.T) {
	input := `# King Lear

## Dramatis Personae
### The Royals
KING LEAR - King of Britain
CORDELIA - Youngest daughter
### The Earls
KENT - A loyal earl`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	play := doc.Body[0].(*ast.Section)
	dp := ast.FindDramatisPersonaeInSection(play)
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
	input := `# Play

## ACT I

### SCENE 1

HAMLET
To be or not to be.

### SCENE 2

HORATIO
Good my lord.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	play, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok, "expected Section, got %T", doc.Body[0])
	require.Len(t, play.Children, 1)

	act, ok := play.Children[0].(*ast.Section)
	require.True(t, ok, "expected Act section, got %T", play.Children[0])
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
	require.Len(t, dlg.ParentheticalInlines(), 1)
	assert.Equal(t, "aside", dlg.ParentheticalInlines()[0].(*ast.TextNode).Value)
	assert.Len(t, dlg.Lines, 1)
}

func TestDialogueWithFormattedParenthetical(t *testing.T) {
	input := `# Play

GUARD
(offstage, _exasperated_; **overlapping** NOTBOB)
Hold there.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	require.Len(t, dlg.ParentheticalInlines(), 5)
	assert.Equal(t, "offstage, ", dlg.ParentheticalInlines()[0].(*ast.TextNode).Value)
	_, ok := dlg.ParentheticalInlines()[1].(*ast.UnderlineNode)
	require.True(t, ok)
	assert.Equal(t, "; ", dlg.ParentheticalInlines()[2].(*ast.TextNode).Value)
	_, ok = dlg.ParentheticalInlines()[3].(*ast.BoldNode)
	require.True(t, ok)
	assert.Equal(t, " NOTBOB", dlg.ParentheticalInlines()[4].(*ast.TextNode).Value)
}

func TestDialogueInlineDirectionWithFormatting(t *testing.T) {
	input := `# Play

GIDEON
The state of our ship is strong.
(beat; _almost_ fighting the impulse to continue.)
By all measures.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	require.Len(t, dlg.Lines, 3)
	require.Len(t, dlg.Lines[1].Content, 1)

	dir, ok := dlg.Lines[1].Content[0].(*ast.InlineDirectionNode)
	require.True(t, ok)
	require.Len(t, dir.Content, 3)
	assert.Equal(t, "beat; ", dir.Content[0].(*ast.TextNode).Value)
	_, ok = dir.Content[1].(*ast.UnderlineNode)
	require.True(t, ok)
	assert.Equal(t, " fighting the impulse to continue.", dir.Content[2].(*ast.TextNode).Value)
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
	assert.True(t, dlg.Forced, "forced cue should record the `@` prefix on the Dialogue node")
}

func TestNaturalCharacterNotForced(t *testing.T) {
	input := `# Play

NARRATOR
Once upon a time.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	assert.False(t, dlg.Forced, "naturally-recognized cue should not be marked forced")
}

func TestDualDialogue(t *testing.T) {
	input := `# Play

BRICK
Screw retirement.

STEEL ^
Screw retirement.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	play, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, play.Children, 1)

	dual, ok := play.Children[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node")
	assert.Equal(t, "BRICK", dual.Left.Character)
	assert.Equal(t, "STEEL", dual.Right.Character)
	assert.Len(t, dual.Left.Lines, 1)
	assert.Len(t, dual.Right.Lines, 1)
}

func TestDualDialogueWithParenthetical(t *testing.T) {
	input := `# Play

BRICK
(angry)
Screw retirement.

STEEL ^
(calmly)
Screw retirement.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	play, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, play.Children, 1)

	dual, ok := play.Children[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node")
	assert.Equal(t, "(angry)", dual.Left.Parenthetical)
	assert.Equal(t, "(calmly)", dual.Right.Parenthetical)
}

func TestDualDialogueForcedCharacter(t *testing.T) {
	input := `# Play

BRICK
Hello.

@narrator ^
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	play, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, play.Children, 1)

	dual, ok := play.Children[0].(*ast.DualDialogue)
	require.True(t, ok, "expected DualDialogue node")
	assert.Equal(t, "BRICK", dual.Left.Character)
	assert.False(t, dual.Left.Forced)
	assert.Equal(t, "narrator", dual.Right.Character)
	assert.True(t, dual.Right.Forced, "`@` prefix on a dual-dialogue cue should mark it forced")
}

func TestDualDialogueInScene(t *testing.T) {
	input := `# Play

## ACT I

### SCENE 1

BRICK
Hello.

STEEL ^
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	// Navigate into act > scene > children
	play, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.NotEmpty(t, play.Children)

	act, ok := play.Children[0].(*ast.Section)
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

func TestNestedNotesSectionRemainsProse(t *testing.T) {
	input := `# My Compilation
Author: Me

## First Production
My Compilation was first produced in 2026 at the Faketown Fringe Festival in Anytown, US.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	section := doc.Body[0].(*ast.Section)
	require.Len(t, section.Children, 1)

	notes := section.Children[0].(*ast.Section)
	assert.Equal(t, ast.SectionGeneric, notes.Kind)
	require.Len(t, notes.Lines, 1)
	assert.Equal(t, "My Compilation was first produced in 2026 at the Faketown Fringe Festival in Anytown, US.", notes.Lines[0].Content[0].(*ast.TextNode).Value)
}

func TestInlineCharacterAlias(t *testing.T) {
	input := `# Play

## Dramatis Personae
JAMES/JIM - Her estranged son

## ACT I

JIM
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	play := doc.Body[0].(*ast.Section)
	dp := ast.FindDramatisPersonaeInSection(play)
	require.NotNil(t, dp)
	require.Len(t, dp.Characters, 1)
	assert.Equal(t, "JAMES", dp.Characters[0].Name)
	assert.Equal(t, []string{"JIM"}, dp.Characters[0].Aliases)
}

func TestStandaloneCharacterAliasProducesError(t *testing.T) {
	input := `# Play

## Dramatis Personae
JAMES - Her estranged son
[JAMES/JIM]`

	_, errs := Parse([]byte(input))
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "standalone character alias syntax is not supported")
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

func TestStandaloneCallout(t *testing.T) {
	input := `# Play

## ACT I

>> Midwinter. The room has not been heated for days.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var callout *ast.Callout
	for _, n := range doc.Body {
		if sect, ok := n.(*ast.Section); ok {
			for _, c := range sect.Children {
				if inner, ok := c.(*ast.Section); ok {
					for _, ic := range inner.Children {
						if co, ok := ic.(*ast.Callout); ok {
							callout = co
							break
						}
					}
				}
			}
		}
	}
	require.NotNil(t, callout, "expected Callout node")
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

func TestDemotedCueInLeafGenericSection(t *testing.T) {
	// Demoted ALL-CAPS lines still render as stage directions in leaf generic sections.
	input := "# Play\n\n## Notes\nALICE\n"
	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	play, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.NotEmpty(t, play.Children)
	notes, ok := play.Children[0].(*ast.Section)
	require.True(t, ok, "expected Notes section, got %T", play.Children[0])

	require.NotEmpty(t, notes.Children, "ALL-CAPS line should appear as a stage direction child")
	sd, ok := notes.Children[0].(*ast.StageDirection)
	require.True(t, ok, "expected *ast.StageDirection, got %T", notes.Children[0])
	require.Len(t, sd.Content, 1)
	text, ok := sd.Content[0].(*ast.TextNode)
	require.True(t, ok)
	assert.Equal(t, "ALICE", text.Value)
	assert.Empty(t, notes.Lines, "ALL-CAPS line must not become prose")
}

func TestCueCommentsAreTransparentInDialogue(t *testing.T) {
	// Comments stay transparent inside dialogue.
	cases := map[string]string{
		"line comment":  "# Play\n\nJIM\n// he pauses\nWHAT\n",
		"block comment": "# Play\n\nJIM\n/* he pauses */\nWHAT\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			doc, errs := Parse([]byte(input))
			require.Empty(t, errs)

			var dlg *ast.Dialogue
			findDialogue(doc.Body, &dlg)
			require.NotNil(t, dlg, "expected a single Dialogue block")
			assert.Equal(t, "JIM", dlg.Character)

			require.Len(t, dlg.Lines, 1, "WHAT should be dialogue text, not a new cue")
			require.Len(t, dlg.Lines[0].Content, 1)
			textNode, ok := dlg.Lines[0].Content[0].(*ast.TextNode)
			require.True(t, ok, "expected TextNode, got %T", dlg.Lines[0].Content[0])
			assert.Equal(t, "WHAT", textNode.Value)
		})
	}
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

func TestErrorRecovery_UnclosedBlockComment(t *testing.T) {
	input := `# Play

/*
comment that never ends

HAMLET
Still here.`

	doc, errs := Parse([]byte(input))
	assert.NotNil(t, doc, "document should not be nil even with errors")
	assert.NotEmpty(t, errs, "expected parse errors for unclosed block comment")
}

func TestErrorRecovery_CharacterAliasWithoutCharacterEntry(t *testing.T) {
	input := `# Play

## Dramatis Personae
[HAMLET/PRINCE]
HAMLET/PRINCE - Prince of Denmark`

	doc, errs := Parse([]byte(input))
	assert.NotNil(t, doc, "document should not be nil even with errors")
	assert.NotEmpty(t, errs, "expected parse errors for orphaned character alias")
	play := doc.Body[0].(*ast.Section)
	dp := ast.FindDramatisPersonaeInSection(play)
	require.NotNil(t, dp, "expected dramatis personae section to survive recovery")
	assert.Len(t, dp.Characters, 1, "expected parser to recover and keep subsequent character entries")
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
	input := `# A Test Play
Author: Test Author

## Dramatis Personae
### Main Characters
HAMLET - Prince of Denmark
HORATIO - Friend to Hamlet

## ACT I

### SCENE 1

(The stage is dark.)

HAMLET
(aside)
To be or not to be,
  that is the question.

// End of soliloquy

===`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.NotEmpty(t, doc.Body)

	play := doc.Body[0].(*ast.Section)
	require.NotNil(t, play.Metadata)
	assert.Equal(t, "A Test Play", play.Title)
	assert.Equal(t, "Test Author", play.Metadata.Entries[0].Value)

	dp := ast.FindDramatisPersonaeInSection(play)
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

func TestParseV1TitlePageProducesError(t *testing.T) {
	var builder strings.Builder
	for i := range 5 {
		builder.WriteString("Key")
		builder.WriteString(strings.Repeat("A", i))
		builder.WriteString(": Value\n")
	}

	_, errs := Parse([]byte(builder.String()))
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "document-level metadata is a V1 pattern")
}

func TestParseDialogueLineLimit(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("# Play\n\nHAMLET\n")
	for range maxDialogueLines + 1 {
		builder.WriteString("Line\n")
	}

	doc, errs := Parse([]byte(builder.String()))

	require.NotNil(t, doc)
	require.NotEmpty(t, errs)
	assert.Equal(t, "dialogue exceeds maximum line count", errs[0].Message)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	assert.Len(t, dlg.Lines, maxDialogueLines)
}

func TestInlineDelimiterLookaheadLimitTreatsMarkerAsText(t *testing.T) {
	input := "# Play\n\nHAMLET\n*" + strings.Repeat("a", maxInlineDelimiterLookahead+1) + "*"

	doc, errs := Parse([]byte(input))

	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg)
	require.Len(t, dlg.Lines, 1)

	_, ok := dlg.Lines[0].Content[0].(*ast.TextNode)
	require.True(t, ok, "expected unmatched marker to remain plain text")
}

func TestDialogueParagraphBreak(t *testing.T) {
	input := `# Play

HAMLET
To be or not to be,
that is the question.

Whether 'tis nobler in the mind.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	var dlg *ast.Dialogue
	findDialogue(doc.Body, &dlg)
	require.NotNil(t, dlg, "expected to find Dialogue node")
	assert.Equal(t, "HAMLET", dlg.Character)

	// Three content lines plus one paragraph break marker
	require.Len(t, dlg.Lines, 4)

	// First two lines have content
	assert.NotEmpty(t, dlg.Lines[0].Content, "line 0 should have content")
	assert.NotEmpty(t, dlg.Lines[1].Content, "line 1 should have content")

	// Third entry is the paragraph break marker (nil Content)
	assert.Empty(t, dlg.Lines[2].Content, "line 2 should be a paragraph break marker")

	// Fourth line has content
	assert.NotEmpty(t, dlg.Lines[3].Content, "line 3 should have content")
}

func TestStageDirectionContinuation(t *testing.T) {
	input := `# Play

## Scene 1

> First direction.
> Second direction.

> Third direction.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	// Find the scene's children
	act := doc.Body[0].(*ast.Section)
	scene := act.Children[0].(*ast.Section)
	require.Len(t, scene.Children, 3)

	sd1 := scene.Children[0].(*ast.StageDirection)
	sd2 := scene.Children[1].(*ast.StageDirection)
	sd3 := scene.Children[2].(*ast.StageDirection)

	assert.False(t, sd1.Continuation, "first direction is not a continuation")
	assert.True(t, sd2.Continuation, "second direction is adjacent — should be continuation")
	assert.False(t, sd3.Continuation, "third direction has blank line before — not a continuation")
}

func TestCalloutContinuation(t *testing.T) {
	input := `# Play

## Scene 1

>> First callout.
>> Second callout.

>> Third callout.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)

	act := doc.Body[0].(*ast.Section)
	scene := act.Children[0].(*ast.Section)
	require.Len(t, scene.Children, 3)

	c1 := scene.Children[0].(*ast.Callout)
	c2 := scene.Children[1].(*ast.Callout)
	c3 := scene.Children[2].(*ast.Callout)

	assert.False(t, c1.Continuation, "first callout is not a continuation")
	assert.True(t, c2.Continuation, "second callout is adjacent — should be continuation")
	assert.False(t, c3.Continuation, "third callout has blank line before — not a continuation")
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

func TestCommentBetweenHeadingAndMetadata(t *testing.T) {
	input := `# Play
// leading note
Author: Me

HAMLET
Hello.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	section, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.NotNil(t, section.Metadata)
	require.Len(t, section.Metadata.Entries, 1)
	assert.Equal(t, "Author", section.Metadata.Entries[0].Key)
	assert.Equal(t, "Me", section.Metadata.Entries[0].Value)
}

func TestStageDirectionSurvivesProseSection(t *testing.T) {
	input := `# Preface

> A quiet prelude.

Just a line of prose.`

	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	section, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)

	var foundStageDirection bool
	for _, child := range section.Children {
		if sd, ok := child.(*ast.StageDirection); ok {
			foundStageDirection = true
			assert.NotEmpty(t, sd.Content)
		}
	}
	assert.True(t, foundStageDirection, "expected `>` line to remain a StageDirection node")
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

func TestCharacterDescriptionInlines(t *testing.T) {
	input := `# The Play

## Dramatis Personae

HAMLET - Prince of **Denmark**, _melancholic_
`
	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)

	top, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.Len(t, top.Children, 1)
	section, ok := top.Children[0].(*ast.Section)
	require.True(t, ok)
	require.Equal(t, ast.SectionDramatisPersonae, section.Kind)
	require.Len(t, section.Characters, 1)

	ch := section.Characters[0]
	assert.Equal(t, "HAMLET", ch.Name)
	assert.Equal(t, "Prince of **Denmark**, _melancholic_", ch.Description)
	require.NotEmpty(t, ch.DescriptionInlines)

	var hasBold, hasUnderline bool
	for _, inline := range ch.DescriptionInlines {
		switch inline.(type) {
		case *ast.BoldNode:
			hasBold = true
		case *ast.UnderlineNode:
			hasUnderline = true
		}
	}
	assert.True(t, hasBold, "expected bold inline in description")
	assert.True(t, hasUnderline, "expected underline inline in description")
}

func TestTitlePageValueInlines(t *testing.T) {
	input := `# A Play
Subtitle: A tragedy in *five* acts
Author: Jane Doe
`
	doc, errs := Parse([]byte(input))
	require.Empty(t, errs)
	require.Len(t, doc.Body, 1)
	top, ok := doc.Body[0].(*ast.Section)
	require.True(t, ok)
	require.NotNil(t, top.Metadata)
	require.Len(t, top.Metadata.Entries, 2)

	var subtitle *ast.KeyValue
	for i := range top.Metadata.Entries {
		if strings.EqualFold(top.Metadata.Entries[i].Key, "subtitle") {
			subtitle = &top.Metadata.Entries[i]
		}
	}
	require.NotNil(t, subtitle)

	require.NotEmpty(t, subtitle.ValueInlines)
	var hasItalic bool
	for _, inline := range subtitle.ValueInlines {
		if _, ok := inline.(*ast.ItalicNode); ok {
			hasItalic = true
		}
	}
	assert.True(t, hasItalic, "expected italic inline in subtitle value")
}
