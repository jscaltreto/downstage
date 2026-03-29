package lsp

import (
	"context"
	"encoding/json"
	"log/slog"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func newHandler(dm *documentManager, logger *slog.Logger) *handler {
	return &handler{
		dm:     dm,
		logger: logger,
	}
}

type handler struct {
	conn   jsonrpc2.Conn
	dm     *documentManager
	logger *slog.Logger
	cancel context.CancelFunc
}

func (h *handler) Handle(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	method := req.Method()
	h.logger.Info("received request", slog.String("method", method))

	switch method {
	case protocol.MethodInitialize:
		return h.handleInitialize(ctx, reply, req)
	case protocol.MethodInitialized:
		return reply(ctx, nil, nil)
	case protocol.MethodShutdown:
		return h.handleShutdown(ctx, reply)
	case protocol.MethodExit:
		h.cancel()
		return reply(ctx, nil, nil)
	case protocol.MethodTextDocumentDidOpen:
		return h.handleDidOpen(ctx, reply, req)
	case protocol.MethodTextDocumentDidChange:
		return h.handleDidChange(ctx, reply, req)
	case protocol.MethodTextDocumentDidClose:
		return h.handleDidClose(ctx, reply, req)
	case protocol.MethodSemanticTokensFull:
		return h.handleSemanticTokensFull(ctx, reply, req)
	case protocol.MethodTextDocumentDocumentSymbol:
		return h.handleDocumentSymbol(ctx, reply, req)
	case protocol.MethodTextDocumentHover:
		return h.handleHover(ctx, reply, req)
	case protocol.MethodTextDocumentDefinition:
		return h.handleDefinition(ctx, reply, req)
	case protocol.MethodTextDocumentCompletion:
		return h.handleCompletion(ctx, reply, req)
	default:
		return reply(ctx, nil, jsonrpc2.NewError(jsonrpc2.MethodNotFound, "method not found: "+method))
	}
}

// semanticTokensCapability is the shape the LSP client expects for semanticTokensProvider.
// The protocol library's SemanticTokensOptions is incomplete, so we define our own.
type semanticTokensCapability struct {
	Full   bool                          `json:"full"`
	Legend protocol.SemanticTokensLegend `json:"legend"`
}

func (h *handler) handleInitialize(ctx context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
	result := protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
			},
			SemanticTokensProvider: semanticTokensCapability{
				Full: true,
				Legend: protocol.SemanticTokensLegend{
					TokenTypes:     semanticTokenTypesLegend(),
					TokenModifiers: []protocol.SemanticTokenModifiers{},
				},
			},
			DocumentSymbolProvider: true,
			HoverProvider:          true,
			DefinitionProvider:     true,
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"@", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"},
			},
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "downstage-lsp",
			Version: "0.1.0",
		},
	}
	return reply(ctx, result, nil)
}

func (h *handler) handleShutdown(ctx context.Context, reply jsonrpc2.Replier) error {
	return reply(ctx, nil, nil)
}

func (h *handler) handleDidOpen(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		h.logger.Error("failed to unmarshal didOpen params", slog.String("error", err.Error()))
		return reply(ctx, nil, nil)
	}

	diags := h.dm.Open(params.TextDocument.URI, params.TextDocument.Text)
	return h.publishDiagnostics(ctx, params.TextDocument.URI, diags)
}

func (h *handler) handleDidChange(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		h.logger.Error("failed to unmarshal didChange params", slog.String("error", err.Error()))
		return reply(ctx, nil, nil)
	}

	if len(params.ContentChanges) == 0 {
		return reply(ctx, nil, nil)
	}

	// Full sync: last content change is the full document
	content := params.ContentChanges[len(params.ContentChanges)-1].Text
	diags := h.dm.Change(params.TextDocument.URI, content)
	return h.publishDiagnostics(ctx, params.TextDocument.URI, diags)
}

func (h *handler) handleDidClose(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		h.logger.Error("failed to unmarshal didClose params", slog.String("error", err.Error()))
		return reply(ctx, nil, nil)
	}

	h.dm.Close(params.TextDocument.URI)

	// Clear diagnostics on close
	return h.publishDiagnostics(ctx, params.TextDocument.URI, []protocol.Diagnostic{})
}

func (h *handler) handleSemanticTokensFull(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.SemanticTokensParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, nil)
	}

	doc := h.dm.Get(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, &protocol.SemanticTokens{}, nil)
	}

	tokens := computeSemanticTokens(doc.doc, doc.errors)
	return reply(ctx, &protocol.SemanticTokens{Data: tokens}, nil)
}

func (h *handler) handleDocumentSymbol(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DocumentSymbolParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, nil)
	}

	doc := h.dm.Get(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, []protocol.DocumentSymbol{}, nil)
	}

	symbols := computeDocumentSymbols(doc.doc, doc.errors)
	return reply(ctx, symbols, nil)
}

func (h *handler) handleHover(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.HoverParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, nil)
	}

	doc := h.dm.Get(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, nil, nil)
	}

	result := computeHover(doc.doc, doc.errors, params.Position)
	if result == nil {
		return reply(ctx, nil, nil)
	}
	return reply(ctx, result, nil)
}

func (h *handler) handleDefinition(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DefinitionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, nil)
	}

	doc := h.dm.Get(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, nil, nil)
	}

	loc := computeDefinition(doc.doc, doc.errors, params.TextDocument.URI, params.Position)
	if loc == nil {
		return reply(ctx, nil, nil)
	}
	return reply(ctx, loc, nil)
}

func (h *handler) handleCompletion(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.CompletionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, emptyCompletionList(), nil)
	}

	doc := h.dm.Get(params.TextDocument.URI)
	if doc == nil {
		return reply(ctx, emptyCompletionList(), nil)
	}

	result := computeCompletion(doc.doc, doc.errors, doc.content, params.Position)
	return reply(ctx, result, nil)
}

func (h *handler) publishDiagnostics(ctx context.Context, docURI protocol.DocumentURI, diags []protocol.Diagnostic) error {
	if diags == nil {
		diags = []protocol.Diagnostic{}
	}

	params := protocol.PublishDiagnosticsParams{
		URI:         docURI,
		Diagnostics: diags,
	}
	return h.conn.Notify(ctx, protocol.MethodTextDocumentPublishDiagnostics, params)
}
