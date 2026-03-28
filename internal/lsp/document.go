package lsp

import (
	"sync"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

type documentState struct {
	content string
	doc     *ast.Document
	errors  []*parser.ParseError
}

func newDocumentManager(p Parser) *documentManager {
	return &documentManager{
		parser: p,
		docs:   make(map[protocol.DocumentURI]*documentState),
	}
}

type documentManager struct {
	parser Parser
	mu     sync.RWMutex
	docs   map[protocol.DocumentURI]*documentState
}

// Open stores a newly opened document, parses it, and returns diagnostics.
func (dm *documentManager) Open(uri protocol.DocumentURI, content string) []protocol.Diagnostic {
	doc, errs := dm.parser.Parse([]byte(content))
	state := &documentState{
		content: content,
		doc:     doc,
		errors:  errs,
	}

	dm.mu.Lock()
	dm.docs[uri] = state
	dm.mu.Unlock()

	return buildDiagnostics(doc, errs)
}

// Change updates a document's content, re-parses, and returns diagnostics.
func (dm *documentManager) Change(uri protocol.DocumentURI, content string) []protocol.Diagnostic {
	doc, errs := dm.parser.Parse([]byte(content))
	state := &documentState{
		content: content,
		doc:     doc,
		errors:  errs,
	}

	dm.mu.Lock()
	dm.docs[uri] = state
	dm.mu.Unlock()

	return buildDiagnostics(doc, errs)
}

// Close removes a document from the store.
func (dm *documentManager) Close(uri protocol.DocumentURI) {
	dm.mu.Lock()
	delete(dm.docs, uri)
	dm.mu.Unlock()
}

// Get retrieves the current state of a document. Returns nil if not found.
func (dm *documentManager) Get(uri protocol.DocumentURI) *documentState {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.docs[uri]
}
