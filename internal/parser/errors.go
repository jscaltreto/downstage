package parser

import (
	"fmt"

	"github.com/jscaltreto/downstage/internal/token"
)

// ParseError represents a non-fatal parsing error with source location.
// Code is an optional stable identifier for downstream surfaces (LSP,
// quick fixes) that need to match on the error beyond its message text.
type ParseError struct {
	Message string
	Range   token.Range
	Code    string
}

// Parse error codes that downstream tooling keys off.
const (
	ErrCodeDPUnicodeDash     = "dp-unicode-dash"
	ErrCodeDPStandaloneAlias = "dp-standalone-alias"
)

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", e.Range.Start.Line+1, e.Range.Start.Column+1, e.Message)
}
