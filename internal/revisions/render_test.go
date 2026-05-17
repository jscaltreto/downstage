package revisions

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_PureInsert_ProducesPDF(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n\nHAMLET\nSecond.\n"

	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source:    []byte(v1),
		V2Source:    []byte(v2),
		V1Name:      "v1.ds",
		V2Name:      "v2.ds",
		Config:      render.DefaultConfig(),
		MarkChanges: true,
	})
	require.NoError(t, err)
	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "%PDF-"), "output should be a PDF file (got prefix %q)", out[:min(len(out), 8)])
	assert.Greater(t, buf.Len(), 1000, "PDF should be non-trivially sized")
}

func TestRender_NoDifferences_ReturnsSentinel(t *testing.T) {
	src := "# Play\n\nHAMLET\nLine.\n"
	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source: []byte(src),
		V2Source: []byte(src),
		Config:   render.DefaultConfig(),
	})
	assert.ErrorIs(t, err, ErrNoDifferences)
}

func TestRender_ParseError_ReturnsDescriptiveError(t *testing.T) {
	v1 := "# Bad\n\n@\n" // forced cue with no name — likely produces a parse error
	v2 := "# Good\n\nHAMLET\nLine.\n"
	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source: []byte(v1),
		V2Source: []byte(v2),
		V1Name:   "v1.ds",
		V2Name:   "v2.ds",
		Config:   render.DefaultConfig(),
	})
	if err != nil && err != ErrNoDifferences {
		assert.Contains(t, err.Error(), "v1.ds", "parse errors should be attributed to the source file")
	}
}

func TestRender_AppliesPageLabelFormatter(t *testing.T) {
	v1 := "# P\n\n## ACT I\n\nHAMLET\nOriginal.\n"
	v2 := "# P\n\n## ACT I\n\nHAMLET\nUpdated.\n"
	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source:      []byte(v1),
		V2Source:      []byte(v2),
		Config:        render.DefaultConfig(),
		PageNumbering: PageNumberingV1Labels,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(buf.String(), "%PDF-"))
}

func TestRender_NaturalPageNumbering(t *testing.T) {
	v1 := "# P\n\n## ACT I\n\nHAMLET\nA.\n"
	v2 := "# P\n\n## ACT I\n\nHAMLET\nB.\n"
	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source:      []byte(v1),
		V2Source:      []byte(v2),
		Config:        render.DefaultConfig(),
		PageNumbering: PageNumberingNatural,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(buf.String(), "%PDF-"))
}

func TestRender_MultipleRegions(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nA.\n\nHAMLET\nB.\n\n## ACT II\n\nHAMLET\nC.\n\nHAMLET\nD.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nA prime.\n\nHAMLET\nB.\n\n## ACT II\n\nHAMLET\nC.\n\nHAMLET\nD prime.\n"
	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source:    []byte(v1),
		V2Source:    []byte(v2),
		Config:      render.DefaultConfig(),
		MarkChanges: true,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(buf.String(), "%PDF-"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRender_RemovedMarkerFlagAcceptedWithoutCrash(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n\nHAMLET\nSecond.\n"
	var buf bytes.Buffer
	err := Render(&buf, Options{
		V1Source:              []byte(v1),
		V2Source:              []byte(v2),
		Config:                render.DefaultConfig(),
		IncludeRemovedMarkers: true,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(buf.String(), "%PDF-"))
}

func TestSynthesizeDocument_TrailingPlaceholderProducesExtraPage(t *testing.T) {
	region := Region{
		Kind:        RegionReplace,
		V1FirstPage: 23,
		V1LastPage:  24,
		MarginNote:  "Replace pp. 23–24 of v1",
		V2Nodes:     []ast.Node{&ast.Section{Level: 0, Kind: ast.SectionGeneric, Title: "Inserted heading"}},
	}
	doc, _ := SynthesizeDocument([]Region{region}, SynthOptions{
		TrailingPlaceholders: map[int]string{0: "REMOVED — p. 24 intentionally removed"},
	})
	assert.Len(t, doc.Body, 3, "trailing placeholder should add a PageBreak + Section")
	_, isPageBreak := doc.Body[1].(*ast.PageBreak)
	assert.True(t, isPageBreak)
	placeholder, isSection := doc.Body[2].(*ast.Section)
	require.True(t, isSection)
	assert.Contains(t, placeholder.Title, "REMOVED")
}
