package lsp

import (
	"strings"
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

func TestDocumentManager_OpenRejectsNonFileURI(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	uri := protocol.DocumentURI("untitled:buffer")

	diags := dm.Open(uri, "hello")

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Message != "document URI must use the file:// scheme" {
		t.Fatalf("unexpected diagnostic: %q", diags[0].Message)
	}
	if dm.Get(uri) != nil {
		t.Fatal("expected invalid URI to be rejected from the document store")
	}
}

func TestDocumentManager_OpenRejectsOversizedDocument(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	uri := protocol.DocumentURI("file:///large.ds")
	content := strings.Repeat("a", maxDocumentBytes+1)

	diags := dm.Open(uri, content)

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	expected := "document exceeds maximum size of 1048576 bytes"
	if diags[0].Message != expected {
		t.Fatalf("expected %q, got %q", expected, diags[0].Message)
	}

	doc := dm.Get(uri)
	if doc == nil {
		t.Fatal("expected oversized document to remain tracked with diagnostics")
	}
	if doc.doc != nil {
		t.Fatal("expected oversized document to skip parsing")
	}
}

func TestDocumentManager_ChangeReplacesStateWithValidationFailure(t *testing.T) {
	dm := newDocumentManager(&testParser{})
	uri := protocol.DocumentURI("file:///large.ds")

	dm.Open(uri, "initial")
	diags := dm.Change(uri, strings.Repeat("a", maxDocumentBytes+1))

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	doc := dm.Get(uri)
	if doc == nil {
		t.Fatal("expected document to remain tracked after invalid change")
	}
	if doc.doc != nil {
		t.Fatal("expected invalid change to clear the parsed document state")
	}
	if doc.content == "initial" {
		t.Fatal("expected stored content to be replaced by the invalid update")
	}
}
