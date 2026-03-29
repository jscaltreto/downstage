package lsp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func TestHandleInitialize_AdvertisesFoldingRangeProvider(t *testing.T) {
	h := newHandler(newDocumentManager(parserFunc(parser.Parse)), slog.Default())
	req, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), protocol.MethodInitialize, protocol.InitializeParams{})
	if err != nil {
		t.Fatalf("new call: %v", err)
	}

	var got protocol.InitializeResult
	reply := func(_ context.Context, result interface{}, err error) error {
		if err != nil {
			t.Fatalf("unexpected reply error: %v", err)
		}
		typed, ok := result.(protocol.InitializeResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		got = typed
		return nil
	}

	if err := h.handleInitialize(context.Background(), reply, req); err != nil {
		t.Fatalf("handleInitialize: %v", err)
	}

	if got.Capabilities.FoldingRangeProvider != true {
		t.Fatalf("expected folding range provider to be advertised, got %#v", got.Capabilities.FoldingRangeProvider)
	}
}

func TestHandleFoldingRange_ReturnsComputedRanges(t *testing.T) {
	dm := newDocumentManager(parserFunc(parser.Parse))
	h := newHandler(dm, slog.Default())
	uri := protocol.DocumentURI("file:///test.ds")
	content := `Title: Play
Author: Example

# Play

## ACT I

### SCENE 1

SONG: Ballad
Line one
SONG END`

	dm.Open(uri, content)

	req, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(2), protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		},
	})
	if err != nil {
		t.Fatalf("new call: %v", err)
	}

	var got []protocol.FoldingRange
	reply := func(_ context.Context, result interface{}, err error) error {
		if err != nil {
			t.Fatalf("unexpected reply error: %v", err)
		}
		typed, ok := result.([]protocol.FoldingRange)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		got = typed
		return nil
	}

	if err := h.handleFoldingRange(context.Background(), reply, req); err != nil {
		t.Fatalf("handleFoldingRange: %v", err)
	}

	if len(got) == 0 {
		t.Fatal("expected folding ranges")
	}
	if got[0].StartLine != 0 || got[0].EndLine != 1 {
		t.Fatalf("unexpected title page folding range: %+v", got[0])
	}
}
