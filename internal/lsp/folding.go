package lsp

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

func computeFoldingRanges(doc *ast.Document, _ []*parser.ParseError, content string) []protocol.FoldingRange {
	if doc == nil {
		return nil
	}

	lineCount := strings.Count(content, "\n") + 1
	var ranges []protocol.FoldingRange

	if doc.TitlePage != nil {
		if folding, ok := buildFoldingRange(doc.TitlePage.Range, lineCount, protocol.RegionFoldingRange); ok {
			ranges = append(ranges, folding)
		}
	}

	for _, node := range doc.Body {
		ranges = append(ranges, foldingRangesForNode(node, lineCount)...)
	}

	return ranges
}

func foldingRangesForNode(node ast.Node, lineCount int) []protocol.FoldingRange {
	var ranges []protocol.FoldingRange

	switch v := node.(type) {
	case *ast.Section:
		if folding, ok := buildFoldingRange(v.Range, lineCount, protocol.RegionFoldingRange); ok {
			ranges = append(ranges, folding)
		}
		for _, child := range v.Children {
			ranges = append(ranges, foldingRangesForNode(child, lineCount)...)
		}
	case *ast.Song:
		if folding, ok := buildFoldingRange(v.Range, lineCount, protocol.RegionFoldingRange); ok {
			ranges = append(ranges, folding)
		}
		for _, child := range v.Content {
			ranges = append(ranges, foldingRangesForNode(child, lineCount)...)
		}
	case *ast.Comment:
		if !v.Block {
			return nil
		}
		if folding, ok := buildFoldingRange(v.Range, lineCount, protocol.CommentFoldingRange); ok {
			ranges = append(ranges, folding)
		}
	}

	return ranges
}

func buildFoldingRange(r token.Range, lineCount int, kind protocol.FoldingRangeKind) (protocol.FoldingRange, bool) {
	startLine := r.Start.Line
	endLine := foldEndLine(r, lineCount)
	if startLine < 0 || endLine <= startLine {
		return protocol.FoldingRange{}, false
	}

	return protocol.FoldingRange{
		StartLine:      uint32(startLine),
		StartCharacter: uint32(r.Start.Column),
		EndLine:        uint32(endLine),
		Kind:           kind,
	}, true
}

func foldEndLine(r token.Range, lineCount int) int {
	endLine := r.End.Line
	if endLine >= lineCount {
		endLine = lineCount - 1
	}
	if r.End.Column == 0 && endLine > r.Start.Line {
		endLine--
	}
	return endLine
}
