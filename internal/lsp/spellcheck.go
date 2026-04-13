package lsp

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

type SpellcheckContext struct {
	AllowWords    []string
	IgnoredRanges []protocol.Range
}

func ComputeSpellcheckContext(doc *ast.Document, _ []*parser.ParseError) SpellcheckContext {
	index := newDocumentIndex(doc)

	return SpellcheckContext{
		AllowWords:    spellAllowWords(index.documentCharacterNames),
		IgnoredRanges: ignoredSpellcheckRanges(doc),
	}
}

func spellAllowWords(names []string) []string {
	seen := make(map[string]struct{})
	words := make([]string, 0)

	for _, name := range names {
		for _, word := range strings.FieldsFunc(name, splitSpellAllowWord) {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}
			key := strings.ToUpper(word)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			words = append(words, word)
		}
	}

	return words
}

func splitSpellAllowWord(r rune) bool {
	switch {
	case r >= 'A' && r <= 'Z':
		return false
	case r >= 'a' && r <= 'z':
		return false
	case r == '\'' || r == '’':
		return false
	default:
		return true
	}
}

func ignoredSpellcheckRanges(doc *ast.Document) []protocol.Range {
	if doc == nil {
		return []protocol.Range{}
	}

	var tokens []rawToken
	for _, node := range doc.Body {
		tokens = append(tokens, extractTokens(node)...)
	}

	sortTokens(tokens)

	ranges := make([]protocol.Range, 0)
	for _, tok := range tokens {
		if tok.tokenType != tokenTypeNamespace &&
			tok.tokenType != tokenTypeType &&
			tok.tokenType != tokenTypeKeyword &&
			tok.tokenType != tokenTypeProperty {
			continue
		}
		ranges = append(ranges, protocol.Range{
			Start: protocol.Position{
				Line:      uint32(tok.line),
				Character: uint32(tok.startChar),
			},
			End: protocol.Position{
				Line:      uint32(tok.line),
				Character: uint32(tok.startChar + tok.length),
			},
		})
	}

	return ranges
}
