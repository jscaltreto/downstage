package lsp

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestLimitedStreamRejectsOversizedMessageBeforeReadingBody(t *testing.T) {
	conn := &testReadWriteCloser{
		Reader: strings.NewReader("Content-Length: 1048577\r\n\r\n"),
		Writer: &bytes.Buffer{},
	}
	stream := newLimitedStream(conn, maxLSPMessageBytes)

	_, _, err := stream.Read(context.Background())

	if err == nil {
		t.Fatal("expected oversized message to be rejected")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type testReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (t *testReadWriteCloser) Close() error {
	return nil
}
