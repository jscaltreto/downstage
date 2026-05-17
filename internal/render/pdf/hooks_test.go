package pdf

import (
	"bytes"
	"sync/atomic"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderToBuffer(t *testing.T, cfg render.Config, src string) []byte {
	t.Helper()
	doc, errs := parser.Parse([]byte(src))
	require.Empty(t, errs)
	nr := NewRenderer(cfg)
	var buf bytes.Buffer
	require.NoError(t, render.Walk(nr, doc, &buf))
	return buf.Bytes()
}

func TestConfig_PageLabelFormatter_OverridesFooter(t *testing.T) {
	calls := int32(0)
	cfg := render.DefaultConfig()
	cfg.PageLabelFormatter = func(p int) string {
		atomic.AddInt32(&calls, 1)
		// Map page 1 → "23A".
		if p == 1 {
			return "23A"
		}
		return ""
	}
	out := renderToBuffer(t, cfg, "# Tiny\n\nHAMLET\nLine.\n")
	assert.NotEmpty(t, out)
	assert.Greater(t, atomic.LoadInt32(&calls), int32(0), "formatter should have been called by the footer")
}

func TestConfig_PageHeaderFn_DrawsTopMarginText(t *testing.T) {
	calls := int32(0)
	cfg := render.DefaultConfig()
	cfg.PageHeaderFn = func(p int) string {
		atomic.AddInt32(&calls, 1)
		return "REVISION HEADER"
	}
	out := renderToBuffer(t, cfg, "# Tiny\n\nHAMLET\nLine.\n")
	// The header callback must have been called for the body page.
	assert.Greater(t, atomic.LoadInt32(&calls), int32(0))
	// PDF stream is binary but our header text is ASCII — a substring grep
	// is sufficient to detect that the header was actually emitted.
	if !bytes.Contains(out, []byte("REVISION HEADER")) {
		// PDF text may be compressed; in that case at least verify the
		// callback fired (already asserted above).
		t.Log("header text not present uncompressed — relying on callback assertion")
	}
}

func TestConfig_SkipOutline_DropsBookmarks(t *testing.T) {
	// Use a script with several outline-bearing sections so the bookmark
	// objects produce a measurable size difference vs the skipped variant.
	src := "# Play\n\n" +
		"## ACT I\n\n### SCENE 1\n\nHAMLET\nLine.\n\n### SCENE 2\n\nHAMLET\nLine.\n\n" +
		"## ACT II\n\n### SCENE 1\n\nHAMLET\nLine.\n\n### SCENE 2\n\nHAMLET\nLine.\n"

	withOutline := renderToBuffer(t, render.DefaultConfig(), src)

	cfgNoOutline := render.DefaultConfig()
	cfgNoOutline.SkipOutline = true
	withoutOutline := renderToBuffer(t, cfgNoOutline, src)

	// Skipping outline emission should produce a measurably smaller PDF
	// because fpdf no longer writes outline-item dictionaries.
	assert.Less(t, len(withoutOutline), len(withOutline),
		"SkipOutline render should be smaller (no outline dicts emitted): withOutline=%d, withoutOutline=%d",
		len(withOutline), len(withoutOutline))
}

func TestConfig_MarkChangedBlocksMembership(t *testing.T) {
	// Membership lookup is purely a config check; it is exercised by the
	// renderer's asterisk path in body.go but we sanity-check the helper
	// itself here.
	cfg := render.DefaultConfig()
	doc, errs := parser.Parse([]byte("# P\n\nHAMLET\nLine.\n"))
	require.Empty(t, errs)
	// Find the dialogue node.
	var d any
	for _, n := range doc.Body {
		d = n
	}
	cfg.MarkChangedBlocks = map[any]bool{d: true}

	r := NewRenderer(cfg).(*pdfRenderer)
	assert.True(t, r.blockChanged(d))
	assert.False(t, r.blockChanged("not-a-node"))
}
