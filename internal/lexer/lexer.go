package lexer

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/jscaltreto/downstage/internal/token"
)

// context tracks which section of the document the lexer is processing.
type context int

const (
	ctxTitlePage context = iota
	ctxBody
)

// Lex tokenizes the input bytes into a slice of tokens.
func Lex(input []byte) []token.Token {
	l := &lexer{
		input:      input,
		lines:      bytes.Split(input, []byte("\n")),
		ctx:        ctxTitlePage,
		offset:     0,
		tokens:     make([]token.Token, 0, 256),
		inBlock:    false,
		cueAllowed: true,
	}
	l.lex()
	return l.tokens
}

type lexer struct {
	input   []byte
	lines   [][]byte
	ctx     context
	offset  int // byte offset of the current line start
	tokens  []token.Token
	inBlock bool // inside a block comment
	// cueAllowed reports whether the current line may start a cue.
	cueAllowed bool
}

func (l *lexer) lex() {
	for lineNum, rawLine := range l.lines {
		line := string(rawLine)
		lineLen := len(rawLine)

		// Handle block comment continuation/end
		if l.inBlock {
			trimmed := strings.TrimSpace(line)
			if trimmed == "*/" {
				l.emit(token.BlockCommentEnd, line, line, lineNum, 0, lineLen)
				l.inBlock = false
			} else if idx := strings.Index(line, "*/"); idx >= 0 {
				// Block comment ends mid-line; emit the whole line as comment end
				l.emit(token.BlockCommentEnd, line, line, lineNum, 0, lineLen)
				l.inBlock = false
			} else {
				l.emit(token.Text, line, line, lineNum, 0, lineLen)
			}
			l.offset += lineLen + 1
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Blank line
		if trimmed == "" {
			l.emit(token.Blank, "", line, lineNum, 0, lineLen)
			l.offset += lineLen + 1
			continue
		}

		// Line comment
		if strings.HasPrefix(trimmed, "//") {
			l.emit(token.LineComment, trimmed, line, lineNum, 0, lineLen)
			l.offset += lineLen + 1
			continue
		}

		// Block comment start
		if strings.HasPrefix(trimmed, "/*") {
			if strings.Contains(trimmed, "*/") {
				// Single-line block comment
				l.emit(token.BlockCommentStart, trimmed, line, lineNum, 0, lineLen)
				l.emit(token.BlockCommentEnd, "", line, lineNum, 0, lineLen)
			} else {
				l.emit(token.BlockCommentStart, trimmed, line, lineNum, 0, lineLen)
				l.inBlock = true
			}
			l.offset += lineLen + 1
			continue
		}

		// Page break
		if trimmed == "===" {
			l.ctx = ctxBody
			l.emit(token.PageBreak, trimmed, line, lineNum, 0, lineLen)
			l.offset += lineLen + 1
			continue
		}

		// Headings (always switch to body context)
		if strings.HasPrefix(trimmed, "###") && len(trimmed) > 3 && trimmed[3] == ' ' {
			l.ctx = ctxBody
			content := strings.TrimSpace(trimmed[4:])
			l.emit(token.HeadingH3, content, line, lineNum, 0, lineLen)
			l.offset += lineLen + 1
			continue
		}

		if strings.HasPrefix(trimmed, "##") && !strings.HasPrefix(trimmed, "###") && len(trimmed) > 2 && trimmed[2] == ' ' {
			l.ctx = ctxBody
			content := strings.TrimSpace(trimmed[3:])
			l.emit(token.HeadingH2, content, line, lineNum, 0, lineLen)
			l.offset += lineLen + 1
			continue
		}

		if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "##") && len(trimmed) > 1 && trimmed[1] == ' ' {
			l.ctx = ctxBody
			content := strings.TrimSpace(trimmed[2:])
			l.emit(token.HeadingH1, content, line, lineNum, 0, lineLen)
			l.offset += lineLen + 1
			continue
		}

		// Title page context
		if l.ctx == ctxTitlePage {
			if idx := strings.Index(line, ":"); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				value := strings.TrimSpace(line[idx+1:])
				l.emit(token.TitleKey, key, line, lineNum, 0, idx+1)
				if value != "" {
					l.emit(token.TitleValue, value, line, lineNum, idx+1, lineLen)
				}
			} else if isTitleContinuation(line) {
				l.emit(token.TitleValue, trimmed, line, lineNum, 0, lineLen)
			} else {
				l.ctx = ctxBody
				l.classifyBodyLine(line, trimmed, lineNum, lineLen)
			}
			l.offset += lineLen + 1
			continue
		}

		// Body context
		l.classifyBodyLine(line, trimmed, lineNum, lineLen)
		l.offset += lineLen + 1
	}

	// EOF
	finalOffset := len(l.input)
	finalLine := len(l.lines)
	l.tokens = append(l.tokens, token.Token{
		Type:    token.EOF,
		Literal: "",
		Range: token.Range{
			Start: token.Position{Line: finalLine, Column: 0, Offset: finalOffset},
			End:   token.Position{Line: finalLine, Column: 0, Offset: finalOffset},
		},
	})
}

func (l *lexer) classifyBodyLine(line, trimmed string, lineNum, lineLen int) {
	// Dual dialogue: character name or forced character ending with ^.
	if strings.HasSuffix(trimmed, " ^") || strings.HasSuffix(trimmed, "\t^") {
		name := strings.TrimSpace(trimmed[:len(trimmed)-2])
		if strings.HasPrefix(name, "@") && len(name) > 1 {
			l.emit(token.DualDialogueChar, name, line, lineNum, 0, len(name))
			return
		}
		if isCharacterName(name) && l.cueAllowed {
			l.emit(token.DualDialogueChar, name, line, lineNum, 0, len(name))
			return
		}
	}

	// Forced character: @TEXT.
	if strings.HasPrefix(trimmed, "@") && len(trimmed) > 1 {
		l.emit(token.ForcedCharacter, trimmed, line, lineNum, 0, lineLen)
		return
	}

	// Forced heading: .TEXT where TEXT starts with uppercase
	if strings.HasPrefix(trimmed, ".") && len(trimmed) > 1 && unicode.IsUpper(rune(trimmed[1])) {
		l.emit(token.ForcedHeading, trimmed, line, lineNum, 0, lineLen)
		return
	}

	// Character alias: [NAME/ALIAS]
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") && strings.Contains(trimmed, "/") {
		l.emit(token.CharacterAlias, trimmed, line, lineNum, 0, lineLen)
		return
	}

	// SONG END
	if trimmed == "SONG END" {
		l.emit(token.SongEnd, trimmed, line, lineNum, 0, lineLen)
		return
	}

	// SONG (start)
	if trimmed == "SONG" || strings.HasPrefix(trimmed, "SONG ") || strings.HasPrefix(trimmed, "SONG:") {
		l.emit(token.SongStart, trimmed, line, lineNum, 0, lineLen)
		return
	}

	// Callout: line starting with >>
	if strings.HasPrefix(trimmed, ">>") {
		content := strings.TrimSpace(trimmed[2:])
		l.emit(token.Callout, content, line, lineNum, 0, lineLen)
		return
	}

	// Stage direction: line starting with >
	if strings.HasPrefix(trimmed, ">") {
		content := strings.TrimSpace(trimmed[1:])
		l.emit(token.StageDirection, content, line, lineNum, 0, lineLen)
		return
	}

	// Verse: line starting with 2+ spaces (check raw line, not trimmed)
	if len(line) >= 2 && line[0] == ' ' && line[1] == ' ' {
		l.emit(token.Verse, line, line, lineNum, 0, lineLen)
		return
	}

	// ALL CAPS character name: requires a blank line or document start.
	if isCharacterName(trimmed) && l.cueAllowed {
		l.emit(token.CharacterName, trimmed, line, lineNum, 0, lineLen)
		return
	}

	// Default: text
	l.emit(token.Text, line, line, lineNum, 0, lineLen)
}

func (l *lexer) emit(typ token.Type, literal, sourceLine string, line, colStart, colEnd int) {
	startColumn := utf16Column(sourceLine, colStart)
	endColumn := utf16Column(sourceLine, colEnd)
	l.tokens = append(l.tokens, token.Token{
		Type:    typ,
		Literal: literal,
		Range: token.Range{
			Start: token.Position{Line: line, Column: startColumn, Offset: l.offset + colStart},
			End:   token.Position{Line: line, Column: endColumn, Offset: l.offset + colEnd},
		},
	})
	// Comments do not change cue context.
	switch typ {
	case token.LineComment, token.BlockCommentStart, token.BlockCommentEnd:
		return
	}
	if l.inBlock {
		return
	}
	l.cueAllowed = typ == token.Blank
}

// IsCharacterName reports whether s looks like an ALL CAPS character name.
func IsCharacterName(s string) bool { return isCharacterName(s) }

// isCharacterName returns true if s looks like an ALL CAPS character name.
// Must be 1+ characters, contain at least one letter, and consist only of
// uppercase letters, digits, spaces, periods, commas, hyphens, and apostrophes.
func isCharacterName(s string) bool {
	if len(s) < 1 {
		return false
	}
	hasLetter := false
	for _, r := range s {
		if unicode.IsUpper(r) {
			hasLetter = true
			continue
		}
		if unicode.IsDigit(r) || r == ' ' || r == '.' || r == ',' || r == '-' || r == '\'' || r == '/' {
			continue
		}
		return false
	}
	return hasLetter
}

func isTitleContinuation(line string) bool {
	if line == "" {
		return false
	}
	return line[0] == ' ' || line[0] == '\t'
}

func utf16Column(s string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(s) {
		return token.UTF16Len(s)
	}
	return token.UTF16Len(s[:byteOffset])
}
