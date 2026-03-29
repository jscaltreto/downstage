package lsp

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

const maxDocumentBytes = 1 << 20

type documentState struct {
	content     string
	doc         *ast.Document
	index       *documentIndex
	errors      []*parser.ParseError
	diagnostics []protocol.Diagnostic
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
	if diags, ok := validateDocumentInput(uri, content); !ok {
		dm.storeValidationFailure(uri, content, diags)
		return diags
	}

	doc, errs := dm.parser.Parse([]byte(content))
	index := newDocumentIndex(doc)
	diags := buildDiagnosticsWithIndex(doc, errs, index)
	state := &documentState{
		content:     content,
		doc:         doc,
		index:       index,
		errors:      errs,
		diagnostics: diags,
	}

	dm.mu.Lock()
	dm.docs[uri] = state
	dm.mu.Unlock()

	return diags
}

// Change updates a document's content, re-parses, and returns diagnostics.
func (dm *documentManager) Change(uri protocol.DocumentURI, content string) []protocol.Diagnostic {
	if diags, ok := validateDocumentInput(uri, content); !ok {
		dm.storeValidationFailure(uri, content, diags)
		return diags
	}

	doc, errs := dm.parser.Parse([]byte(content))
	index := newDocumentIndex(doc)
	diags := buildDiagnosticsWithIndex(doc, errs, index)
	state := &documentState{
		content:     content,
		doc:         doc,
		index:       index,
		errors:      errs,
		diagnostics: diags,
	}

	dm.mu.Lock()
	dm.docs[uri] = state
	dm.mu.Unlock()

	return diags
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

func (dm *documentManager) storeValidationFailure(
	uri protocol.DocumentURI,
	content string,
	diags []protocol.Diagnostic,
) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !isValidDocumentURI(uri) {
		delete(dm.docs, uri)
		return
	}

	dm.docs[uri] = &documentState{
		content:     content,
		doc:         nil,
		errors:      nil,
		diagnostics: diags,
	}
}

func validateDocumentInput(uri protocol.DocumentURI, content string) ([]protocol.Diagnostic, bool) {
	if !isValidDocumentURI(uri) {
		return []protocol.Diagnostic{newValidationDiagnostic("document URI must use the file:// scheme")}, false
	}
	if len(content) > maxDocumentBytes {
		return []protocol.Diagnostic{newValidationDiagnostic(
			fmt.Sprintf("document exceeds maximum size of %d bytes", maxDocumentBytes),
		)}, false
	}
	return nil, true
}

func isValidDocumentURI(uri protocol.DocumentURI) bool {
	parsed, err := url.Parse(string(uri))
	if err != nil {
		return false
	}
	return parsed.Scheme == "file" && parsed.Path != ""
}

func newValidationDiagnostic(message string) protocol.Diagnostic {
	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		},
		Severity: protocol.DiagnosticSeverityError,
		Source:   "downstage-lsp",
		Message:  message,
	}
}
