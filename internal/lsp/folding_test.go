package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeFoldingRanges_NilDocument(t *testing.T) {
	ranges := computeFoldingRanges(nil, nil, "")
	if ranges != nil {
		t.Fatalf("expected nil ranges for nil doc, got %#v", ranges)
	}
}

func TestComputeFoldingRanges_TitlePageSectionsSongAndBlockComment(t *testing.T) {
	content := `# Play
Title: Play
Author: Example

## ACT I

### SCENE 1

SONG: Ballad
Line one
SONG END

/*
block
comment
*/`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	ranges := computeFoldingRanges(doc, errs, content)
	if len(ranges) != 6 {
		t.Fatalf("expected 6 folding ranges, got %d", len(ranges))
	}

	assertFoldingRange(t, ranges[0], 1, 2, protocol.RegionFoldingRange)
	assertFoldingRange(t, ranges[1], 0, 15, protocol.RegionFoldingRange)
	assertFoldingRange(t, ranges[2], 4, 15, protocol.RegionFoldingRange)
	assertFoldingRange(t, ranges[3], 6, 15, protocol.RegionFoldingRange)
	assertFoldingRange(t, ranges[4], 8, 10, protocol.RegionFoldingRange)
	assertFoldingRange(t, ranges[5], 12, 15, protocol.CommentFoldingRange)
}

func TestComputeFoldingRanges_SkipsSingleLineNodes(t *testing.T) {
	content := `# Play
### SCENE`

	doc, errs := parser.Parse([]byte(content))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	ranges := computeFoldingRanges(doc, errs, content)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 folding range, got %d", len(ranges))
	}

	assertFoldingRange(t, ranges[0], 0, 1, protocol.RegionFoldingRange)
}

func assertFoldingRange(
	t *testing.T,
	got protocol.FoldingRange,
	startLine int,
	endLine int,
	kind protocol.FoldingRangeKind,
) {
	t.Helper()

	if int(got.StartLine) != startLine || int(got.EndLine) != endLine {
		t.Fatalf("unexpected folding range lines: got %d-%d want %d-%d", got.StartLine, got.EndLine, startLine, endLine)
	}
	if got.Kind != kind {
		t.Fatalf("unexpected folding kind: got %q want %q", got.Kind, kind)
	}
}
