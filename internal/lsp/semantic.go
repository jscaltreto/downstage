package lsp

import (
	"cmp"
	"slices"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

// Semantic token type indices.
const (
	tokenTypeNamespace = 0 // headings (acts, scenes)
	tokenTypeType      = 1 // character names
	tokenTypeComment   = 2 // stage directions, comments
	tokenTypeKeyword   = 3 // song keywords
	tokenTypeProperty  = 4 // metadata keys (title page)
	tokenTypeString    = 5 // inline formatting (bold/italic)
)

// SemanticTokenTypeNames lists the token type names in index order.
// This is useful for consumers (e.g. WASM bridge) that need the legend
// without depending on LSP protocol types.
var SemanticTokenTypeNames = []string{
	"namespace", "type", "comment", "keyword", "property", "string",
}

// SemanticTokenTypesLegend returns the legend of token types registered with the client.
func SemanticTokenTypesLegend() []protocol.SemanticTokenTypes {
	return []protocol.SemanticTokenTypes{
		protocol.SemanticTokenNamespace, // 0
		protocol.SemanticTokenType,      // 1
		protocol.SemanticTokenComment,   // 2
		protocol.SemanticTokenKeyword,   // 3
		protocol.SemanticTokenProperty,  // 4
		protocol.SemanticTokenString,    // 5
	}
}

// ComputeSemanticTokens walks the AST and returns delta-encoded semantic tokens.
// Each token is 5 integers: [deltaLine, deltaStartChar, length, tokenType, tokenModifiers].
func ComputeSemanticTokens(doc *ast.Document, _ []*parser.ParseError) []uint32 {
	if doc == nil {
		return nil
	}

	var tokens []rawToken

	// Body nodes
	for _, n := range doc.Body {
		tokens = append(tokens, extractTokens(n)...)
	}

	return deltaEncode(tokens)
}

func extractTokens(n ast.Node) []rawToken {
	var tokens []rawToken

	switch v := n.(type) {
	case *ast.Section:
		if v.Metadata != nil {
			for _, entry := range v.Metadata.Entries {
				r := entry.Range
				tokens = append(tokens, rawToken{
					line:      r.Start.Line,
					startChar: r.Start.Column,
					length:    utf16Len(entry.Key),
					tokenType: tokenTypeProperty,
				})
			}
		}
		header := sectionHeaderText(v)
		if header != "" {
			startChar := v.Range.Start.Column
			if v.Level > 0 {
				startChar += v.Level + 1
			}
			tokens = append(tokens, rawToken{
				line:      v.Range.Start.Line,
				startChar: startChar,
				length:    utf16Len(header),
				tokenType: tokenTypeNamespace,
			})
		}
		for _, child := range v.Children {
			tokens = append(tokens, extractTokens(child)...)
		}

	case *ast.DualDialogue:
		tokens = append(tokens, extractTokens(v.Left)...)
		tokens = append(tokens, extractTokens(v.Right)...)

	case *ast.Dialogue:
		r := v.NameRange()
		name := v.Character
		if name != "" {
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeType,
			})
		}
		if v.Parenthetical != "" {
			r := v.ParentheticalRange()
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeComment,
			})
		}
		for _, line := range v.Lines {
			tokens = append(tokens, extractInlineTokens(line.Content)...)
		}

	case *ast.StageDirection:
		r := v.Range
		tokens = append(tokens, rawToken{
			line:      r.Start.Line,
			startChar: r.Start.Column,
			length:    r.End.Column - r.Start.Column,
			tokenType: tokenTypeComment,
		})

	case *ast.Callout:
		r := v.Range
		tokens = append(tokens, rawToken{
			line:      r.Start.Line,
			startChar: r.Start.Column,
			length:    r.End.Column - r.Start.Column,
			tokenType: tokenTypeComment,
		})

	case *ast.Song:
		header := "SONG"
		switch {
		case v.Number != "" && v.Title != "":
			header = "SONG " + v.Number + ": " + v.Title
		case v.Number != "":
			header = "SONG " + v.Number
		case v.Title != "":
			header = "SONG: " + v.Title
		}
		if header != "" {
			tokens = append(tokens, rawToken{
				line:      v.Range.Start.Line,
				startChar: v.Range.Start.Column,
				length:    utf16Len(header),
				tokenType: tokenTypeKeyword,
			})
		}
		for _, child := range v.Content {
			tokens = append(tokens, extractTokens(child)...)
		}

	case *ast.Comment:
		r := v.Range
		tokens = append(tokens, rawToken{
			line:      r.Start.Line,
			startChar: r.Start.Column,
			length:    r.End.Column - r.Start.Column,
			tokenType: tokenTypeComment,
		})
	}

	return tokens
}

func sectionHeaderText(section *ast.Section) string {
	switch section.Kind {
	case ast.SectionAct:
		return buildNumberedHeader("ACT", section.Number, section.Title)
	case ast.SectionScene:
		if strings.TrimSpace(section.Number) != "" {
			return buildNumberedHeader("SCENE", section.Number, section.Title)
		}
		if strings.TrimSpace(section.Title) == "" {
			return "SCENE"
		}
		return section.Title
	default:
		return section.Title
	}
}

func buildNumberedHeader(keyword, number, title string) string {
	number = strings.TrimSpace(number)
	title = strings.TrimSpace(title)

	switch {
	case number != "" && title != "":
		return keyword + " " + number + ": " + title
	case number != "":
		return keyword + " " + number
	case title != "":
		return keyword + ": " + title
	default:
		return keyword
	}
}

func extractInlineTokens(inlines []ast.Inline) []rawToken {
	var tokens []rawToken
	for _, inline := range inlines {
		switch v := inline.(type) {
		case *ast.BoldNode:
			r := v.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeString,
			})
		case *ast.ItalicNode:
			r := v.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeString,
			})
		case *ast.BoldItalicNode:
			r := v.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeString,
			})
		case *ast.UnderlineNode:
			r := v.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeString,
			})
		case *ast.StrikethroughNode:
			r := v.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeString,
			})
		case *ast.InlineDirectionNode:
			r := v.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    r.End.Column - r.Start.Column,
				tokenType: tokenTypeComment,
			})
		}
	}
	return tokens
}

type rawToken struct {
	line      int
	startChar int
	length    int
	tokenType int
}

// deltaEncode converts absolute-positioned tokens into LSP delta-encoded format.
func deltaEncode(tokens []rawToken) []uint32 {
	if len(tokens) == 0 {
		return nil
	}

	// Sort tokens by position (line, then column).
	sortTokens(tokens)

	data := make([]uint32, 0, len(tokens)*5)
	prevLine := 0
	prevChar := 0

	for _, t := range tokens {
		deltaLine := t.line - prevLine
		deltaStartChar := t.startChar
		if deltaLine == 0 {
			deltaStartChar = t.startChar - prevChar
		}

		data = append(data,
			uint32(deltaLine),
			uint32(deltaStartChar),
			uint32(t.length),
			uint32(t.tokenType),
			0, // no modifiers
		)

		prevLine = t.line
		prevChar = t.startChar
	}

	return data
}

// sortTokens sorts by line then column.
func sortTokens(tokens []rawToken) {
	slices.SortFunc(tokens, func(a, b rawToken) int {
		if c := cmp.Compare(a.line, b.line); c != 0 {
			return c
		}
		return cmp.Compare(a.startChar, b.startChar)
	})
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}
