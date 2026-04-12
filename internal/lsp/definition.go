package lsp

import (
	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

// computeDefinition returns the location of a character's entry in the dramatis personae
// when the cursor is on a character name in dialogue or song.
func computeDefinition(doc *ast.Document, _ []*parser.ParseError, uri protocol.DocumentURI, pos protocol.Position) *protocol.Location {
	if doc == nil {
		return nil
	}

	charName := findCharacterAtPosition(doc, pos)
	if charName == "" {
		return nil
	}

	dp := scopedDramatisPersonae(doc, int(pos.Line))
	if dp == nil {
		return nil
	}

	ch, _ := findCharacterByName(dp, charName)
	if ch == nil {
		return nil
	}

	r := toLSPRange(ch.Range)
	return &protocol.Location{
		URI:   uri,
		Range: r,
	}
}
