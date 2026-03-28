package lsp

import (
	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
)

// Parser is the interface for parsing Downstage documents.
type Parser interface {
	Parse(content []byte) (*ast.Document, []*parser.ParseError)
}

// parserFunc adapts a standalone Parse function to the Parser interface.
var _ Parser = parserFunc(nil)

type parserFunc func([]byte) (*ast.Document, []*parser.ParseError)

func (f parserFunc) Parse(content []byte) (*ast.Document, []*parser.ParseError) {
	return f(content)
}
