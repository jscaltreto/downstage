package lsp

import (
	"context"
	"log/slog"
	"os"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/jsonrpc2"
)

// Start runs the LSP server on stdio using the real parser.
func Start(ctx context.Context) error {
	return StartWithParser(ctx, parserFunc(parser.Parse))
}

// StartWithParser runs the LSP server on stdio with the given parser implementation.
func StartWithParser(ctx context.Context, p Parser) error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	dm := newDocumentManager(p)
	h := newHandler(dm, logger)

	stream := jsonrpc2.NewStream(newStdioReadWriteCloser())
	conn := jsonrpc2.NewConn(stream)

	h.conn = conn

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h.cancel = cancel

	conn.Go(ctx, h.Handle)

	<-conn.Done()
	return conn.Err()
}

// stdioReadWriteCloser wraps stdin/stdout as a single ReadWriteCloser.
func newStdioReadWriteCloser() *stdioReadWriteCloser {
	return &stdioReadWriteCloser{}
}

type stdioReadWriteCloser struct{}

func (s *stdioReadWriteCloser) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (s *stdioReadWriteCloser) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (s *stdioReadWriteCloser) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
