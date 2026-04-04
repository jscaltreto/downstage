package token

import "fmt"

// Position tracks location in source.
type Position struct {
	Line   int // 0-based
	Column int // 0-based
	Offset int // byte offset
}

// Range tracks start/end position.
type Range struct {
	Start Position
	End   Position
}

// Type represents the kind of token.
type Type int

const (
	// Structure
	EOF Type = iota
	Blank
	HeadingH1
	HeadingH2
	HeadingH3

	// Title page
	TitleKey
	TitleValue

	// Characters
	CharacterName
	CharacterAlias

	// Content
	Text
	Dialogue
	StageDirection
	Callout
	Verse

	// Songs
	SongStart
	SongEnd

	// Formatting (inline)
	Bold
	Italic
	BoldItalic
	Underline
	Strikethrough

	// Comments
	LineComment
	BlockCommentStart
	BlockCommentEnd

	// Misc
	PageBreak
	ForcedHeading
	ForcedCharacter
	DualDialogueChar
)

var typeNames = map[Type]string{
	EOF:               "EOF",
	Blank:             "Blank",
	HeadingH1:         "HeadingH1",
	HeadingH2:         "HeadingH2",
	HeadingH3:         "HeadingH3",
	TitleKey:          "TitleKey",
	TitleValue:        "TitleValue",
	CharacterName:     "CharacterName",
	CharacterAlias:    "CharacterAlias",
	Text:              "Text",
	Dialogue:          "Dialogue",
	StageDirection:    "StageDirection",
	Callout:           "Callout",
	Verse:             "Verse",
	SongStart:         "SongStart",
	SongEnd:           "SongEnd",
	Bold:              "Bold",
	Italic:            "Italic",
	BoldItalic:        "BoldItalic",
	Underline:         "Underline",
	Strikethrough:     "Strikethrough",
	LineComment:       "LineComment",
	BlockCommentStart: "BlockCommentStart",
	BlockCommentEnd:   "BlockCommentEnd",
	PageBreak:         "PageBreak",
	ForcedHeading:     "ForcedHeading",
	ForcedCharacter:   "ForcedCharacter",
	DualDialogueChar:  "DualDialogueChar",
}

// String returns a human-readable name for the token type.
func (t Type) String() string {
	if name, ok := typeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", int(t))
}

// Token represents a lexed token with its type, literal value, and source range.
type Token struct {
	Type    Type
	Literal string
	Range   Range
}
