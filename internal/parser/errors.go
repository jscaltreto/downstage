package parser

import (
	"fmt"

	"github.com/jscaltreto/downstage/internal/token"
)

// ParseError represents a non-fatal parsing error with source location.
type ParseError struct {
	Message string
	Range   token.Range
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, col %d: %s", e.Range.Start.Line+1, e.Range.Start.Column+1, e.Message)
}
