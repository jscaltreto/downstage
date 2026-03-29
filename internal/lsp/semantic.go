package lsp

import (
	"unicode/utf16"

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

// semanticTokenTypesLegend returns the legend of token types registered with the client.
func semanticTokenTypesLegend() []protocol.SemanticTokenTypes {
	return []protocol.SemanticTokenTypes{
		protocol.SemanticTokenNamespace, // 0
		protocol.SemanticTokenType,      // 1
		protocol.SemanticTokenComment,   // 2
		protocol.SemanticTokenKeyword,   // 3
		protocol.SemanticTokenProperty,  // 4
		protocol.SemanticTokenString,    // 5
	}
}

// computeSemanticTokens walks the AST and returns delta-encoded semantic tokens.
// Each token is 5 integers: [deltaLine, deltaStartChar, length, tokenType, tokenModifiers].
func computeSemanticTokens(doc *ast.Document, _ []*parser.ParseError) []uint32 {
	if doc == nil {
		return nil
	}

	var tokens []rawToken

	// Title page entries: key is property, value spans the line.
	if tp := doc.TitlePage; tp != nil {
		for _, entry := range tp.Entries {
			r := entry.Range
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    utf16Len(entry.Key),
				tokenType: tokenTypeProperty,
			})
		}
	}

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
		r := v.Range
		title := v.Title
		if title != "" {
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    utf16Len(title),
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

	case *ast.Song:
		r := v.Range
		title := v.Title
		if title == "" && v.Number != "" {
			title = "Song " + v.Number
		}
		if title != "" {
			tokens = append(tokens, rawToken{
				line:      r.Start.Line,
				startChar: r.Start.Column,
				length:    utf16Len(title),
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

// sortTokens sorts by line then column using insertion sort (stable, simple, fast for small N).
func sortTokens(tokens []rawToken) {
	for i := 1; i < len(tokens); i++ {
		key := tokens[i]
		j := i - 1
		for j >= 0 && (tokens[j].line > key.line || (tokens[j].line == key.line && tokens[j].startChar > key.startChar)) {
			tokens[j+1] = tokens[j]
			j--
		}
		tokens[j+1] = key
	}
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
