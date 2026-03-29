package parser

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/token"
)

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Message: "unexpected token",
		Range: token.Range{
			Start: token.Position{Line: 2, Column: 4},
		},
	}

	if got := err.Error(); got != "line 3, col 5: unexpected token" {
		t.Fatalf("unexpected error string: %q", got)
	}
}
