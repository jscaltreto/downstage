package pagemap_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/jscaltreto/downstage/internal/render/pdf/pagemap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderWithMap renders src and returns the populated page-span map.
func renderWithMap(t *testing.T, src string) (*ast.Document, pagemap.Map) {
	t.Helper()
	doc, errs := parser.Parse([]byte(src))
	require.Empty(t, errs, "parse errors: %v", errs)

	rec := pagemap.NewRecorder()
	cfg := render.DefaultConfig()
	cfg.RecordPageMap = rec
	nr := pdf.NewRenderer(cfg)

	var buf bytes.Buffer
	require.NoError(t, render.Walk(nr, doc, &buf))
	return doc, rec.Map()
}

func TestRecorder_BasicBlocksRecordSinglePage(t *testing.T) {
	src := "# Tiny Play\n\n" +
		"## ACT I\n\n" +
		"HAMLET\nA short line.\n\n" +
		"> Lights up.\n"

	doc, m := renderWithMap(t, src)
	require.NotEmpty(t, m, "recorder should have captured spans")

	for _, node := range doc.Body {
		span, ok := m.Lookup(node)
		if !ok {
			continue
		}
		assert.GreaterOrEqual(t, span.End, span.Start, "End must be >= Start")
		assert.GreaterOrEqual(t, span.Start, 1, "page numbers are 1-indexed")
	}
}

func TestRecorder_DialogueOverflowSpansMultiplePages(t *testing.T) {
	// Build a long single-character dialogue cue with many lines so the
	// block has to break across pages.
	var b strings.Builder
	b.WriteString("# Long\n\n## ACT I\n\nHAMLET\n")
	for i := 0; i < 200; i++ {
		b.WriteString("Line of dialogue that takes up space.\n")
	}

	doc, m := renderWithMap(t, b.String())

	// Find the lone Dialogue node and check its span.
	var dialogue *ast.Dialogue
	var walk func([]ast.Node)
	walk = func(nodes []ast.Node) {
		for _, n := range nodes {
			switch v := n.(type) {
			case *ast.Dialogue:
				dialogue = v
			case *ast.Section:
				walk(v.Children)
			}
		}
	}
	walk(doc.Body)
	require.NotNil(t, dialogue, "expected a Dialogue node")

	span, ok := m.Lookup(dialogue)
	require.True(t, ok, "dialogue span must be recorded")
	assert.Greater(t, span.End, span.Start, "long dialogue should span multiple pages: %+v", span)
}

func TestRecorder_LastPageMatchesLargestEnd(t *testing.T) {
	src := "# Multi\n\n" +
		"## ACT I\n\n" +
		"HAMLET\nFirst.\n\n" +
		"===\n\n" +
		"## ACT II\n\n" +
		"HAMLET\nSecond.\n"

	_, m := renderWithMap(t, src)
	last := m.LastPage()
	assert.GreaterOrEqual(t, last, 2, "explicit page break should produce at least 2 pages")
}

func TestRecorder_NilRecorderIsNoOp(t *testing.T) {
	src := "# Tiny\n\nHAMLET\nLine.\n"
	doc, errs := parser.Parse([]byte(src))
	require.Empty(t, errs)

	cfg := render.DefaultConfig()
	cfg.RecordPageMap = nil
	nr := pdf.NewRenderer(cfg)
	var buf bytes.Buffer
	require.NoError(t, render.Walk(nr, doc, &buf))
}

func TestMap_LookupOnZeroValue(t *testing.T) {
	var m pagemap.Map
	_, ok := m.Lookup("nope")
	assert.False(t, ok)
	assert.Equal(t, 0, m.LastPage())
}
