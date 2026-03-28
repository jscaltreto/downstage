package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

// testParser is a minimal parser for testing that returns an empty document.
type testParser struct{}

func (tp *testParser) Parse(_ []byte) (*ast.Document, []*parser.ParseError) {
	return &ast.Document{}, nil
}

func TestDocumentManager_OpenAndGet(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	uri := protocol.DocumentURI("file:///test.ds")

	dm.Open(uri, "hello world")

	doc := dm.Get(uri)
	if doc == nil {
		t.Fatal("expected document to be stored after Open")
	}
	if doc.content != "hello world" {
		t.Errorf("expected content %q, got %q", "hello world", doc.content)
	}
	if doc.doc == nil {
		t.Error("expected parsed document to be set")
	}
}

func TestDocumentManager_Change(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	uri := protocol.DocumentURI("file:///test.ds")

	dm.Open(uri, "initial")
	dm.Change(uri, "updated")

	doc := dm.Get(uri)
	if doc == nil {
		t.Fatal("expected document to exist")
	}
	if doc.content != "updated" {
		t.Errorf("expected content %q, got %q", "updated", doc.content)
	}
}

func TestDocumentManager_Close(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	uri := protocol.DocumentURI("file:///test.ds")

	dm.Open(uri, "content")
	dm.Close(uri)

	doc := dm.Get(uri)
	if doc != nil {
		t.Error("expected document to be removed after Close")
	}
}

func TestDocumentManager_GetNotFound(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	doc := dm.Get(protocol.DocumentURI("file:///nonexistent.ds"))
	if doc != nil {
		t.Error("expected nil for unknown URI")
	}
}
