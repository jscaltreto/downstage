package lsp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/jsonrpc2"
)

const maxLSPMessageBytes = 1 << 20

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

	stream := newLimitedStream(newStdioReadWriteCloser(), maxLSPMessageBytes)
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
	return errors.Join(os.Stdin.Close(), os.Stdout.Close())
}

func newLimitedStream(conn io.ReadWriteCloser, maxContentLength int64) jsonrpc2.Stream {
	return &limitedStream{
		conn:             conn,
		in:               bufio.NewReader(conn),
		maxContentLength: maxContentLength,
	}
}

type limitedStream struct {
	conn             io.ReadWriteCloser
	in               *bufio.Reader
	maxContentLength int64
}

func (s *limitedStream) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	var total int64
	var length int64
	for {
		line, err := s.in.ReadString('\n')
		total += int64(len(line))
		if err != nil {
			return nil, total, fmt.Errorf("failed reading header line: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		colon := strings.IndexRune(line, ':')
		if colon < 0 {
			return nil, total, fmt.Errorf("invalid header line %q", line)
		}

		name, value := line[:colon], strings.TrimSpace(line[colon+1:])
		switch name {
		case jsonrpc2.HdrContentLength:
			if length, err = strconv.ParseInt(value, 10, 32); err != nil {
				return nil, total, fmt.Errorf("failed parsing %s: %v: %w", jsonrpc2.HdrContentLength, value, err)
			}
			if length <= 0 {
				return nil, total, fmt.Errorf("invalid %s: %v", jsonrpc2.HdrContentLength, length)
			}
			if length > s.maxContentLength {
				return nil, total, fmt.Errorf("content length %d exceeds maximum of %d bytes", length, s.maxContentLength)
			}
		}
	}

	if length == 0 {
		return nil, total, fmt.Errorf("missing %s header", jsonrpc2.HdrContentLength)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(s.in, data); err != nil {
		return nil, total, fmt.Errorf("read full of data: %w", err)
	}

	total += length
	msg, err := jsonrpc2.DecodeMessage(data)
	return msg, total, err
}

func (s *limitedStream) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	return jsonrpc2.NewStream(s.conn).Write(ctx, msg)
}

func (s *limitedStream) Close() error {
	return s.conn.Close()
}
