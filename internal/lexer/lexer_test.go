package lexer

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tokenTypes(tokens []token.Token) []token.Type {
	types := make([]token.Type, len(tokens))
	for i, t := range tokens {
		types[i] = t.Type
	}
	return types
}

func TestBlankLines(t *testing.T) {
	tokens := Lex([]byte("# Title\n\n\n"))
	types := tokenTypes(tokens)
	// Trailing newline produces an extra blank line from bytes.Split
	assert.Equal(t, []token.Type{
		token.HeadingH1,
		token.Blank,
		token.Blank,
		token.Blank,
		token.EOF,
	}, types)
}

func TestHeadings(t *testing.T) {
	input := "# Act One\n## Scene One\n### Note"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Equal(t, []token.Type{
		token.HeadingH1,
		token.HeadingH2,
		token.HeadingH3,
		token.EOF,
	}, types)

	assert.Equal(t, "Act One", tokens[0].Literal)
	assert.Equal(t, "Scene One", tokens[1].Literal)
	assert.Equal(t, "Note", tokens[2].Literal)
}

func TestTitlePage(t *testing.T) {
	input := "Title: My Play\nAuthor: Jane Doe\nDraft: First"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Equal(t, []token.Type{
		token.TitleKey, token.TitleValue,
		token.TitleKey, token.TitleValue,
		token.TitleKey, token.TitleValue,
		token.EOF,
	}, types)

	assert.Equal(t, "Title", tokens[0].Literal)
	assert.Equal(t, "My Play", tokens[1].Literal)
	assert.Equal(t, "Author", tokens[2].Literal)
	assert.Equal(t, "Jane Doe", tokens[3].Literal)
}

func TestTitlePageKeyOnly(t *testing.T) {
	input := "Title:\n  My Play"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Equal(t, []token.Type{
		token.TitleKey,
		token.TitleValue,
		token.EOF,
	}, types)
	assert.Equal(t, "Title", tokens[0].Literal)
	assert.Equal(t, "My Play", tokens[1].Literal)
}

func TestContextSwitchTitleToBody(t *testing.T) {
	input := "Title: My Play\n# Act One\nSome text here"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Equal(t, []token.Type{
		token.TitleKey, token.TitleValue,
		token.HeadingH1,
		token.Text,
		token.EOF,
	}, types)
}

func TestBodyOnlyDocumentStartsInBody(t *testing.T) {
	input := "ALICE\nHello, world!"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Equal(t, []token.Type{
		token.CharacterName,
		token.Text,
		token.EOF,
	}, types)
}

func TestCharacterNameAllCaps(t *testing.T) {
	input := "# Play\n\nJOHN"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.CharacterName, tokens[2].Type)
	assert.Equal(t, "JOHN", tokens[2].Literal)
}

func TestCharacterNameWithPunctuation(t *testing.T) {
	input := "# Play\n\nMRS. O'BRIEN"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.CharacterName, tokens[2].Type)
	assert.Equal(t, "MRS. O'BRIEN", tokens[2].Literal)
}

func TestForcedCharacter(t *testing.T) {
	input := "# Play\n\n@narrator"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.ForcedCharacter, tokens[2].Type)
	assert.Equal(t, "@narrator", tokens[2].Literal)
}

func TestDualDialogueChar(t *testing.T) {
	input := "# Play\n\nSTEEL ^"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.DualDialogueChar, tokens[2].Type)
	assert.Equal(t, "STEEL", tokens[2].Literal)
}

func TestDualDialogueForcedChar(t *testing.T) {
	input := "# Play\n\n@narrator ^"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.DualDialogueChar, tokens[2].Type)
	assert.Equal(t, "@narrator", tokens[2].Literal)
}

func TestCharacterAlias(t *testing.T) {
	input := "# Play\n\n[JOHN/JACK]"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.CharacterAlias, tokens[2].Type)
	assert.Equal(t, "[JOHN/JACK]", tokens[2].Literal)
}

func TestStageDirection(t *testing.T) {
	input := "# Play\n\n> He exits stage left."
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.StageDirection, tokens[2].Type)
	assert.Equal(t, "He exits stage left.", tokens[2].Literal)
}

func TestCallout(t *testing.T) {
	input := "# Play\n\n>> Midwinter. The room has not been heated for days."
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 3)
	assert.Equal(t, token.Callout, tokens[2].Type)
	assert.Equal(t, "Midwinter. The room has not been heated for days.", tokens[2].Literal)
}

func TestLineComment(t *testing.T) {
	input := "# Play\n// this is a comment"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 2)
	assert.Equal(t, token.LineComment, tokens[1].Type)
}

func TestBlockComment(t *testing.T) {
	input := "# Play\n/* start\nsome text\n*/"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Contains(t, types, token.BlockCommentStart)
	assert.Contains(t, types, token.BlockCommentEnd)
}

func TestSingleLineBlockComment(t *testing.T) {
	input := "# Play\n/* inline comment */"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Contains(t, types, token.BlockCommentStart)
	assert.Contains(t, types, token.BlockCommentEnd)
}

func TestPageBreak(t *testing.T) {
	input := "# Play\n==="
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 2)
	assert.Equal(t, token.PageBreak, tokens[1].Type)
}

func TestSongStartEnd(t *testing.T) {
	input := "# Play\nSONG\nLyrics here\nSONG END"
	tokens := Lex([]byte(input))
	types := tokenTypes(tokens)

	assert.Contains(t, types, token.SongStart)
	assert.Contains(t, types, token.SongEnd)
}

func TestSongStartWithTitle(t *testing.T) {
	input := "# Play\nSONG 1 My Song"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 2)
	assert.Equal(t, token.SongStart, tokens[1].Type)
	assert.Equal(t, "SONG 1 My Song", tokens[1].Literal)
}

func TestVerse(t *testing.T) {
	input := "# Play\n  To be or not to be"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 2)
	assert.Equal(t, token.Verse, tokens[1].Type)
}

func TestForcedHeading(t *testing.T) {
	input := "# Play\n.EPILOGUE"
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 2)
	assert.Equal(t, token.ForcedHeading, tokens[1].Type)
	assert.Equal(t, ".EPILOGUE", tokens[1].Literal)
}

func TestDramatisPersonae(t *testing.T) {
	input := "# Dramatis Personae\n## The Court\nKING LEAR\nCORDELIA\n# Act One"
	tokens := Lex([]byte(input))

	// All headings are generic now — no special DP tokens. KING LEAR and
	// CORDELIA both tokenize as Text because neither has a blank line before
	// it (the strict cue rule). The DP parser accepts Text as a character
	// entry, so semantic behaviour in Dramatis Personae sections is
	// unchanged.
	assert.Equal(t, token.HeadingH1, tokens[0].Type)
	assert.Equal(t, "Dramatis Personae", tokens[0].Literal)
	assert.Equal(t, token.HeadingH2, tokens[1].Type)
	assert.Equal(t, "The Court", tokens[1].Literal)
	assert.Equal(t, token.Text, tokens[2].Type)
	assert.Equal(t, token.Text, tokens[3].Type)
	assert.Equal(t, token.HeadingH1, tokens[4].Type)
}

func TestCueRequiresBlankLineBefore(t *testing.T) {
	// Start of document is an implicit blank line.
	tokens := Lex([]byte("JIM\nHello\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, "JIM", tokens[0].Literal)
	assert.Equal(t, token.Text, tokens[1].Type)

	// Adjacent ALL-CAPS lines: the second is dialogue text, not a cue.
	tokens = Lex([]byte("JIM\nWHAT\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, "WHAT", tokens[1].Literal)

	// Parenthetical between cue and shouted dialogue: shouted line stays text.
	tokens = Lex([]byte("JIM\n(angrily)\nWHAT\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, token.Text, tokens[2].Type)

	// A comment with a blank line before it is transparent — the cue after
	// it is still valid.
	tokens = Lex([]byte("# Play\n\n// note: make jim meaner\nJIM\nI am angry\n"))
	assert.Equal(t, token.HeadingH1, tokens[0].Type)
	assert.Equal(t, token.Blank, tokens[1].Type)
	assert.Equal(t, token.LineComment, tokens[2].Type)
	assert.Equal(t, token.CharacterName, tokens[3].Type)
	assert.Equal(t, token.Text, tokens[4].Type)

	// Block comment with a blank line before it is transparent as well.
	tokens = Lex([]byte("# Play\n\n/*\nnote:\naddress comments\n*/\nJIM\nHello\n"))
	assert.Equal(t, token.HeadingH1, tokens[0].Type)
	assert.Equal(t, token.Blank, tokens[1].Type)
	assert.Equal(t, token.BlockCommentStart, tokens[2].Type)
	// body lines of the block comment emit Text, but they're invisible to the
	// cue-context tracker.
	for i := 3; i < 5; i++ {
		assert.Equal(t, token.Text, tokens[i].Type)
	}
	assert.Equal(t, token.BlockCommentEnd, tokens[5].Type)
	assert.Equal(t, token.CharacterName, tokens[6].Type)

	// A comment between dialogue body and an ALL-CAPS line does NOT reopen a
	// cue — the comment has no blank line before it.
	tokens = Lex([]byte("JIM\nHello.\n// inline note\nWHAT\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, token.LineComment, tokens[2].Type)
	assert.Equal(t, token.Text, tokens[3].Type)

	// Heading without a blank line is NOT a cue boundary under the strict
	// rule. ALICE becomes text (and the parser will render it as an implicit
	// stage direction).
	tokens = Lex([]byte("### SCENE 1\nALICE\nHello\n"))
	assert.Equal(t, token.HeadingH3, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, token.Text, tokens[2].Type)

	// Page break without a blank line is also NOT a cue boundary.
	tokens = Lex([]byte("===\n// note\nJIM\nHello\n"))
	assert.Equal(t, token.PageBreak, tokens[0].Type)
	assert.Equal(t, token.LineComment, tokens[1].Type)
	assert.Equal(t, token.Text, tokens[2].Type)
	assert.Equal(t, token.Text, tokens[3].Type)

	// Forced character always wins, even without a blank line.
	tokens = Lex([]byte("JIM\nHello.\n@JANE\nHi.\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, token.ForcedCharacter, tokens[2].Type)

	// Dual dialogue also needs a blank line before it.
	tokens = Lex([]byte("JIM\nHello.\nJANE ^\nHi.\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, token.Text, tokens[2].Type)

	tokens = Lex([]byte("JIM\nHello.\n\nJANE ^\nHi.\n"))
	assert.Equal(t, token.CharacterName, tokens[0].Type)
	assert.Equal(t, token.Text, tokens[1].Type)
	assert.Equal(t, token.Blank, tokens[2].Type)
	assert.Equal(t, token.DualDialogueChar, tokens[3].Type)
}

func TestPositionTracking(t *testing.T) {
	input := "# Hello\nWorld"
	tokens := Lex([]byte(input))

	// First token starts at line 0
	assert.Equal(t, 0, tokens[0].Range.Start.Line)
	// Second token starts at line 1
	assert.Equal(t, 1, tokens[1].Range.Start.Line)
}

func TestEmptyInput(t *testing.T) {
	tokens := Lex([]byte(""))
	types := tokenTypes(tokens)
	// Empty input produces a blank line + EOF
	assert.Equal(t, []token.Type{token.Blank, token.EOF}, types)
}

func TestTextLine(t *testing.T) {
	input := "# Play\nJust some regular text."
	tokens := Lex([]byte(input))

	require.True(t, len(tokens) >= 2)
	assert.Equal(t, token.Text, tokens[1].Type)
}

func TestIsCharacterName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"JOHN", true},
		{"MRS. O'BRIEN", true},
		{"LADY CAPULET", true},
		{"A", true},
		{"john", false},    // lowercase
		{"John", false},    // mixed case
		{"SONG", true},     // lexer handles SONG before char check
		{"SONG END", true}, // same
		{"123", false},     // digits
		{"ALL-GOOD", true}, // hyphen ok
		{"", false},        // empty
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isCharacterName(tt.input))
		})
	}
}

func TestFullDocumentLex(t *testing.T) {
	input := `Title: Hamlet
Author: Shakespeare

# Dramatis Personae
HAMLET — Prince of Denmark
HORATIO — Friend to Hamlet

# Act One
## Scene One

HAMLET
To be or not to be,
  that is the question.

> He pauses.

// a comment
===`

	tokens := Lex([]byte(input))
	require.NotEmpty(t, tokens)

	// Should end with EOF
	assert.Equal(t, token.EOF, tokens[len(tokens)-1].Type)

	// Should contain expected token types
	types := tokenTypes(tokens)
	assert.Contains(t, types, token.TitleKey)
	assert.Contains(t, types, token.TitleValue)
	assert.Contains(t, types, token.HeadingH1) // # Dramatis Personae is now a regular H1
	assert.Contains(t, types, token.HeadingH1)
	assert.Contains(t, types, token.HeadingH2)
	assert.Contains(t, types, token.CharacterName)
	assert.Contains(t, types, token.Verse)
	assert.Contains(t, types, token.StageDirection)
	assert.Contains(t, types, token.LineComment)
	assert.Contains(t, types, token.PageBreak)
}
