package lsp

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type recordingConn struct {
	methods []string
	params  []interface{}
}

func (c *recordingConn) Call(context.Context, string, interface{}, interface{}) (jsonrpc2.ID, error) {
	return jsonrpc2.NewNumberID(0), nil
}

func (c *recordingConn) Notify(_ context.Context, method string, params interface{}) error {
	c.methods = append(c.methods, method)
	c.params = append(c.params, params)
	return nil
}

func (c *recordingConn) Go(context.Context, jsonrpc2.Handler) {}
func (c *recordingConn) Close() error                         { return nil }
func (c *recordingConn) Done() <-chan struct{}                { return nil }
func (c *recordingConn) Err() error                           { return nil }

type replyRecorder struct {
	result interface{}
	err    error
}

func (r *replyRecorder) replier(_ context.Context, result interface{}, err error) error {
	r.result = result
	r.err = err
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestHandler() (*handler, *recordingConn) {
	dm := newDocumentManager(parserFunc(parser.Parse))
	conn := &recordingConn{}
	h := newHandler(dm, testLogger())
	h.conn = conn
	h.cancel = func() {}
	return h, conn
}

func newCall(t *testing.T, id int, method string, params interface{}) jsonrpc2.Request {
	t.Helper()

	req, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(int32(id)), method, params)
	if err != nil {
		t.Fatalf("new call: %v", err)
	}
	return req
}

func newNotification(t *testing.T, method string, params interface{}) jsonrpc2.Request {
	t.Helper()

	req, err := jsonrpc2.NewNotification(method, params)
	if err != nil {
		t.Fatalf("new notification: %v", err)
	}
	return req
}

func TestHandleInitialize_AdvertisesFoldingRangeProvider(t *testing.T) {
	h, _ := newTestHandler()
	req := newCall(t, 1, protocol.MethodInitialize, protocol.InitializeParams{})

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

	rename, ok := got.Capabilities.RenameProvider.(protocol.RenameOptions)
	if !ok {
		t.Fatalf("expected rename provider to be RenameOptions, got %T", got.Capabilities.RenameProvider)
	}
	if !rename.PrepareProvider {
		t.Fatal("expected rename provider to advertise prepareProvider")
	}
}

func TestHandle_UnknownMethodReturnsMethodNotFound(t *testing.T) {
	h, _ := newTestHandler()
	req := newNotification(t, "$/downstage-unknown", map[string]string{"noop": "true"})
	var reply replyRecorder

	if err := h.Handle(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if reply.result != nil {
		t.Fatalf("expected nil result, got %#v", reply.result)
	}
	rpcErr, ok := reply.err.(*jsonrpc2.Error)
	if !ok {
		t.Fatalf("expected jsonrpc2 error, got %T", reply.err)
	}
	if rpcErr == nil {
		t.Fatal("expected method not found error")
	}
	if rpcErr.Code != jsonrpc2.MethodNotFound {
		t.Fatalf("expected method not found code, got %d", rpcErr.Code)
	}
}

func TestHandleDidOpenDidChangeAndDidCloseManageDocumentLifecycle(t *testing.T) {
	h, conn := newTestHandler()
	uri := protocol.DocumentURI("file:///test.ds")
	reply := &replyRecorder{}

	openReq := newNotification(t, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:  uri,
			Text: "# Play\n\nHAMLET\nHello.",
		},
	})
	if err := h.handleDidOpen(context.Background(), reply.replier, openReq); err != nil {
		t.Fatalf("handleDidOpen: %v", err)
	}

	doc := h.dm.Get(uri)
	if doc == nil {
		t.Fatal("expected open document to be tracked")
	}
	if doc.content != "# Play\n\nHAMLET\nHello." {
		t.Fatalf("unexpected open content: %q", doc.content)
	}
	if len(conn.methods) != 1 || conn.methods[0] != protocol.MethodTextDocumentPublishDiagnostics {
		t.Fatalf("expected one diagnostics publish, got %v", conn.methods)
	}

	changeReq := newNotification(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri}},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: "# Play\n\nHAMLET\nUpdated."},
		},
	})
	if err := h.handleDidChange(context.Background(), reply.replier, changeReq); err != nil {
		t.Fatalf("handleDidChange: %v", err)
	}

	doc = h.dm.Get(uri)
	if doc == nil {
		t.Fatal("expected changed document to remain tracked")
	}
	if doc.content != "# Play\n\nHAMLET\nUpdated." {
		t.Fatalf("unexpected changed content: %q", doc.content)
	}
	if len(conn.methods) != 2 {
		t.Fatalf("expected two diagnostics publishes after change, got %d", len(conn.methods))
	}

	closeReq := newNotification(t, protocol.MethodTextDocumentDidClose, protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err := h.handleDidClose(context.Background(), reply.replier, closeReq); err != nil {
		t.Fatalf("handleDidClose: %v", err)
	}

	if got := h.dm.Get(uri); got != nil {
		t.Fatal("expected close to remove the document")
	}
	if len(conn.methods) != 3 {
		t.Fatalf("expected three diagnostics publishes after close, got %d", len(conn.methods))
	}

	lastParams, ok := conn.params[2].(protocol.PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("unexpected diagnostics params type: %T", conn.params[2])
	}
	if len(lastParams.Diagnostics) != 0 {
		t.Fatalf("expected close to clear diagnostics, got %#v", lastParams.Diagnostics)
	}
}

func TestHandleSemanticTokensFull_ReturnsEmptyTokensWhenDocumentMissing(t *testing.T) {
	h, _ := newTestHandler()
	req := newCall(t, 2, protocol.MethodSemanticTokensFull, protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
	})
	var reply replyRecorder

	if err := h.handleSemanticTokensFull(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleSemanticTokensFull: %v", err)
	}

	tokens, ok := reply.result.(*protocol.SemanticTokens)
	if !ok {
		t.Fatalf("unexpected result type: %T", reply.result)
	}
	if len(tokens.Data) != 0 {
		t.Fatalf("expected no semantic tokens, got %#v", tokens.Data)
	}
}

func TestHandleDocumentRoutes_ReturnEmptyResultsWhenDocumentMissing(t *testing.T) {
	h, _ := newTestHandler()
	tests := []struct {
		name   string
		method string
		req    jsonrpc2.Request
		check  func(*testing.T, interface{})
	}{
		{
			name:   "document symbols",
			method: protocol.MethodTextDocumentDocumentSymbol,
			req: newCall(t, 3, protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
			}),
			check: func(t *testing.T, result interface{}) {
				t.Helper()
				symbols, ok := result.([]protocol.DocumentSymbol)
				if !ok {
					t.Fatalf("unexpected result type: %T", result)
				}
				if len(symbols) != 0 {
					t.Fatalf("expected no symbols, got %#v", symbols)
				}
			},
		},
		{
			name:   "hover",
			method: protocol.MethodTextDocumentHover,
			req: newCall(t, 4, protocol.MethodTextDocumentHover, protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
				},
			}),
			check: func(t *testing.T, result interface{}) {
				t.Helper()
				if result != nil {
					t.Fatalf("expected nil hover, got %#v", result)
				}
			},
		},
		{
			name:   "definition",
			method: protocol.MethodTextDocumentDefinition,
			req: newCall(t, 5, protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
				},
			}),
			check: func(t *testing.T, result interface{}) {
				t.Helper()
				if result != nil {
					t.Fatalf("expected nil definition, got %#v", result)
				}
			},
		},
		{
			name:   "completion",
			method: protocol.MethodTextDocumentCompletion,
			req: newCall(t, 6, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
				},
			}),
			check: func(t *testing.T, result interface{}) {
				t.Helper()
				list, ok := result.(*protocol.CompletionList)
				if !ok {
					t.Fatalf("unexpected result type: %T", result)
				}
				if len(list.Items) != 0 {
					t.Fatalf("expected no completion items, got %#v", list.Items)
				}
			},
		},
		{
			name:   "code action",
			method: protocol.MethodTextDocumentCodeAction,
			req: newCall(t, 7, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
			}),
			check: func(t *testing.T, result interface{}) {
				t.Helper()
				actions, ok := result.([]protocol.CodeAction)
				if !ok {
					t.Fatalf("unexpected result type: %T", result)
				}
				if len(actions) != 0 {
					t.Fatalf("expected no code actions, got %#v", actions)
				}
			},
		},
		{
			name:   "folding range",
			method: protocol.MethodTextDocumentFoldingRange,
			req: newCall(t, 8, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///missing.ds")},
				},
			}),
			check: func(t *testing.T, result interface{}) {
				t.Helper()
				ranges, ok := result.([]protocol.FoldingRange)
				if !ok {
					t.Fatalf("unexpected result type: %T", result)
				}
				if len(ranges) != 0 {
					t.Fatalf("expected no folding ranges, got %#v", ranges)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reply replyRecorder

			switch tt.method {
			case protocol.MethodTextDocumentDocumentSymbol:
				if err := h.handleDocumentSymbol(context.Background(), reply.replier, tt.req); err != nil {
					t.Fatalf("handleDocumentSymbol: %v", err)
				}
			case protocol.MethodTextDocumentHover:
				if err := h.handleHover(context.Background(), reply.replier, tt.req); err != nil {
					t.Fatalf("handleHover: %v", err)
				}
			case protocol.MethodTextDocumentDefinition:
				if err := h.handleDefinition(context.Background(), reply.replier, tt.req); err != nil {
					t.Fatalf("handleDefinition: %v", err)
				}
			case protocol.MethodTextDocumentCompletion:
				if err := h.handleCompletion(context.Background(), reply.replier, tt.req); err != nil {
					t.Fatalf("handleCompletion: %v", err)
				}
			case protocol.MethodTextDocumentCodeAction:
				if err := h.handleCodeAction(context.Background(), reply.replier, tt.req); err != nil {
					t.Fatalf("handleCodeAction: %v", err)
				}
			case protocol.MethodTextDocumentFoldingRange:
				if err := h.handleFoldingRange(context.Background(), reply.replier, tt.req); err != nil {
					t.Fatalf("handleFoldingRange: %v", err)
				}
			}

			tt.check(t, reply.result)
		})
	}
}

func TestHandleCompletion_BadParamsReturnEmptyCompletionList(t *testing.T) {
	h, _ := newTestHandler()
	req := newNotification(t, protocol.MethodTextDocumentCompletion, []string{"bad"})
	var reply replyRecorder

	if err := h.handleCompletion(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleCompletion: %v", err)
	}

	list, ok := reply.result.(*protocol.CompletionList)
	if !ok {
		t.Fatalf("unexpected result type: %T", reply.result)
	}
	if len(list.Items) != 0 {
		t.Fatalf("expected no completion items, got %#v", list.Items)
	}
}

func TestHandleFoldingRange_ReturnsComputedRanges(t *testing.T) {
	dm := newDocumentManager(parserFunc(parser.Parse))
	h := newHandler(dm, testLogger())
	uri := protocol.DocumentURI("file:///test.ds")
	content := `# Play
Title: Play
Author: Example

## ACT I

### SCENE 1

SONG: Ballad
Line one
SONG END`

	dm.Open(uri, content)

	req := newCall(t, 9, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		},
	})

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
	if got[0].StartLine != 1 || got[0].EndLine != 2 {
		t.Fatalf("unexpected metadata folding range: %+v", got[0])
	}
}

func TestPublishDiagnostics_NormalizesNilDiagnosticSlice(t *testing.T) {
	h, conn := newTestHandler()
	uri := protocol.DocumentURI("file:///test.ds")

	if err := h.publishDiagnostics(context.Background(), uri, nil); err != nil {
		t.Fatalf("publishDiagnostics: %v", err)
	}

	params, ok := conn.params[0].(protocol.PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("unexpected params type: %T", conn.params[0])
	}
	if params.Diagnostics == nil {
		t.Fatal("expected diagnostics slice to be normalized to empty")
	}
}

func TestHandleDispatchesInitializeThroughTopLevelHandle(t *testing.T) {
	h, _ := newTestHandler()
	req := newCall(t, 10, protocol.MethodInitialize, protocol.InitializeParams{})
	var reply replyRecorder

	if err := h.Handle(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	result, ok := reply.result.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", reply.result)
	}
	if result.ServerInfo == nil || result.ServerInfo.Name != "downstage-lsp" {
		t.Fatalf("unexpected server info: %#v", result.ServerInfo)
	}
}

func TestHandleDidOpen_BadParamsDoNotPublishDiagnostics(t *testing.T) {
	h, conn := newTestHandler()
	req := newNotification(t, protocol.MethodTextDocumentDidOpen, []string{"bad"})
	var reply replyRecorder

	if err := h.handleDidOpen(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleDidOpen: %v", err)
	}

	if reply.result != nil {
		t.Fatalf("expected nil result, got %#v", reply.result)
	}
	if len(conn.methods) != 0 {
		t.Fatalf("expected no diagnostics publish for invalid didOpen params, got %v", conn.methods)
	}
}

func TestHandleFoldingRange_BadParamsReturnEmptySlice(t *testing.T) {
	h, _ := newTestHandler()
	req := newNotification(t, protocol.MethodTextDocumentFoldingRange, []string{"bad"})
	var reply replyRecorder

	if err := h.handleFoldingRange(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleFoldingRange: %v", err)
	}

	ranges, ok := reply.result.([]protocol.FoldingRange)
	if !ok {
		t.Fatalf("unexpected result type: %T", reply.result)
	}
	if len(ranges) != 0 {
		t.Fatalf("expected no folding ranges, got %#v", ranges)
	}
}

func TestHandleDocumentSymbol_WithDocumentReturnsSymbols(t *testing.T) {
	h, _ := newTestHandler()
	uri := protocol.DocumentURI("file:///test.ds")
	h.dm.Open(uri, "# Play\n\n## ACT I")
	req := newCall(t, 11, protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	var reply replyRecorder

	if err := h.handleDocumentSymbol(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleDocumentSymbol: %v", err)
	}

	symbols, ok := reply.result.([]protocol.DocumentSymbol)
	if !ok {
		t.Fatalf("unexpected result type: %T", reply.result)
	}
	if len(symbols) == 0 {
		t.Fatal("expected symbols for populated document")
	}
}

func TestHandleDidChange_IgnoresEmptyContentChanges(t *testing.T) {
	h, conn := newTestHandler()
	uri := protocol.DocumentURI("file:///test.ds")
	h.dm.Open(uri, "# Play\n\nHAMLET\nHello.")
	req := newNotification(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument:   protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri}},
		ContentChanges: nil,
	})
	var reply replyRecorder

	if err := h.handleDidChange(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleDidChange: %v", err)
	}

	if len(conn.methods) != 0 {
		t.Fatalf("expected no diagnostics publish for empty change set, got %v", conn.methods)
	}
	if got := h.dm.Get(uri); got == nil || got.content != "# Play\n\nHAMLET\nHello." {
		t.Fatalf("expected document state to remain unchanged, got %#v", got)
	}
}

func TestHandleHover_WithDocumentButNoMatchReturnsNil(t *testing.T) {
	h, _ := newTestHandler()
	uri := protocol.DocumentURI("file:///test.ds")
	h.dm.Open(uri, "# Play\n\nHAMLET\nHello.")
	req := newCall(t, 12, protocol.MethodTextDocumentHover, protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	var reply replyRecorder

	if err := h.handleHover(context.Background(), reply.replier, req); err != nil {
		t.Fatalf("handleHover: %v", err)
	}
	if reply.result != nil {
		t.Fatalf("expected nil hover, got %#v", reply.result)
	}
}
