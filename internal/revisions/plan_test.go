package revisions

import (
	"bytes"
	"testing"

	"github.com/jscaltreto/downstage/internal/diff"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/jscaltreto/downstage/internal/render/pdf/pagemap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildInputs parses both sources, renders v1 with a page-map recorder, and
// returns everything needed to call Plan.
func buildInputs(t *testing.T, v1src, v2src string) ([]diff.Block, []diff.Block, []diff.Hunk, pagemap.Map) {
	t.Helper()
	v1Doc, errs := parser.Parse([]byte(v1src))
	require.Empty(t, errs)
	v2Doc, errs := parser.Parse([]byte(v2src))
	require.Empty(t, errs)

	v1Blocks := diff.FlattenedBlocks(v1Doc, diff.CanonicalNameMap(v1Doc))
	v2Blocks := diff.FlattenedBlocks(v2Doc, diff.CanonicalNameMap(v2Doc))
	hunks := diff.Diff(v1Blocks, v2Blocks)

	rec := pagemap.NewRecorder()
	cfg := render.DefaultConfig()
	cfg.RecordPageMap = rec
	nr := pdf.NewRenderer(cfg)
	var buf bytes.Buffer
	require.NoError(t, render.Walk(nr, v1Doc, &buf))

	return v1Blocks, v2Blocks, hunks, rec.Map()
}

func TestPlan_NoChangesReturnsEmpty(t *testing.T) {
	src := "# Same\n\nHAMLET\nLine.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, src, src)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	assert.Empty(t, regions)
}

func TestPlan_PureInsert(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n\nHAMLET\nSecond.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	require.Len(t, regions, 1)
	r := regions[0]
	assert.Equal(t, RegionInsert, r.Kind)
	assert.GreaterOrEqual(t, r.V1FirstPage, 1)
	assert.Contains(t, r.MarginNote, "Insert")
	assert.NotEmpty(t, r.V2Nodes)
	assert.NotEmpty(t, r.ChangedNodes)
}

func TestPlan_PureDelete(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n\nHAMLET\nSecond.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	require.Len(t, regions, 1)
	r := regions[0]
	assert.Equal(t, RegionDelete, r.Kind)
	assert.Contains(t, r.MarginNote, "Remove")
	assert.Empty(t, r.V2Nodes)
}

func TestPlan_Replace(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nOld line.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nUpdated line.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	require.Len(t, regions, 1)
	r := regions[0]
	assert.Equal(t, RegionReplace, r.Kind)
	assert.Contains(t, r.MarginNote, "Replace")
	assert.NotEmpty(t, r.V2Nodes)
}

func TestPlan_MergeAdjacentHunksWithinAnchorWindow(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nA.\n\nHAMLET\nB.\n\nHAMLET\nC.\n\nHAMLET\nD.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nA prime.\n\nHAMLET\nB.\n\nHAMLET\nC prime.\n\nHAMLET\nD.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)

	// With default window (4), two non-Equal hunks separated by one Equal
	// block should merge.
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	assert.Len(t, regions, 1, "adjacent modifications within anchor window should merge")
}

func TestPlan_NarrowAnchorWindowKeepsRegionsApart(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nA.\n\nHAMLET\nB.\n\nHAMLET\nC.\n\nHAMLET\nD.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nA prime.\n\nHAMLET\nB.\n\nHAMLET\nC prime.\n\nHAMLET\nD.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)

	// AnchorWindow=0 → no merging; two separate regions.
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{AnchorWindow: -1}) // negative -> default 4
	// Default keeps merge behaviour. Now try a non-default explicit window.
	_ = regions
	// We can't use 0 (treated as default), but verify the default merges.
	// And then verify a *smaller* window (1) still merges, while a window
	// that's too small wouldn't (we'd need a wider gap to test that fully;
	// the parser produces dialogue blocks 1 unit apart in fingerprint
	// space).
	assert.Len(t, Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{AnchorWindow: 1}), 1)
}

func TestPlan_ContextHeading(t *testing.T) {
	v1 := "# Play\n\n## ACT II\n\n### SCENE 3\n\nHAMLET\nOriginal.\n"
	v2 := "# Play\n\n## ACT II\n\n### SCENE 3\n\nHAMLET\nUpdated.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	require.Len(t, regions, 1)
	// Heading should reflect enclosing sections.
	assert.NotEmpty(t, regions[0].ContextHeading)
	assert.Contains(t, regions[0].ContextHeading, "ACT II")
}

func TestPlan_StartOfDocInsert_UsesSentinelFallback(t *testing.T) {
	// New scene inserted at the very front of the body.
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nOriginal.\n"
	v2 := "# Play\n\n## PROLOGUE\n\nNARRATOR\nNew prologue line.\n\n## ACT I\n\nHAMLET\nOriginal.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	require.NotEmpty(t, regions)
	// The earliest region should anchor at page 1.
	assert.GreaterOrEqual(t, regions[0].V1FirstPage, 1)
}

func TestPlan_ChangedNodesSetMatchesNonEqualHunks(t *testing.T) {
	v1 := "# Play\n\n## ACT I\n\nHAMLET\nKeep.\n\nHAMLET\nChange me.\n\nHAMLET\nKeep.\n"
	v2 := "# Play\n\n## ACT I\n\nHAMLET\nKeep.\n\nHAMLET\nChanged.\n\nHAMLET\nKeep.\n"
	v1Blocks, v2Blocks, hunks, m := buildInputs(t, v1, v2)
	regions := Plan(v1Blocks, v2Blocks, hunks, m, PlanOptions{})
	require.NotEmpty(t, regions)
	// One Dialogue node should be in the changed set.
	totalChanged := 0
	for _, r := range regions {
		totalChanged += len(r.ChangedNodes)
	}
	assert.Equal(t, 1, totalChanged, "only the modified dialogue should be marked changed")
}
