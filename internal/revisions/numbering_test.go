package revisions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLetterSuffix(t *testing.T) {
	cases := map[int]string{
		1:  "A",
		2:  "B",
		25: "Y",
		26: "Z",
		27: "AA",
		28: "AB",
		52: "AZ",
		53: "BA",
	}
	for n, want := range cases {
		assert.Equal(t, want, letterSuffix(n), "letterSuffix(%d)", n)
	}
}

func TestLabels_PureInsert(t *testing.T) {
	rp := RegionPaging{
		Region:    Region{Kind: RegionInsert, V1FirstPage: 22, V1LastPage: 22},
		PageCount: 3,
	}
	got := Labels([]RegionPaging{rp}, LabelOptions{})
	assert.Equal(t, []string{"22A", "22B", "22C"}, got)
}

func TestLabels_ReplaceFits(t *testing.T) {
	rp := RegionPaging{
		Region:    Region{Kind: RegionReplace, V1FirstPage: 23, V1LastPage: 25},
		PageCount: 3,
	}
	got := Labels([]RegionPaging{rp}, LabelOptions{})
	assert.Equal(t, []string{"23", "24", "25"}, got)
}

func TestLabels_ReplaceOverflowsWithLetterSuffixes(t *testing.T) {
	rp := RegionPaging{
		Region:    Region{Kind: RegionReplace, V1FirstPage: 23, V1LastPage: 24},
		PageCount: 5,
	}
	got := Labels([]RegionPaging{rp}, LabelOptions{})
	assert.Equal(t, []string{"23", "23A", "23B", "23C", "23D"}, got)
}

func TestLabels_ReplaceShorterWithRemovedMarker(t *testing.T) {
	rp := RegionPaging{
		Region:    Region{Kind: RegionReplace, V1FirstPage: 30, V1LastPage: 33},
		PageCount: 2,
	}
	got := Labels([]RegionPaging{rp}, LabelOptions{IncludeRemovedMarkers: true})
	// 30, 31 are the replacements; 32 is the REMOVED placeholder slot.
	assert.Equal(t, []string{"30", "31", "32"}, got)
}

func TestLabels_ReplaceShorterWithoutRemovedMarker(t *testing.T) {
	rp := RegionPaging{
		Region:    Region{Kind: RegionReplace, V1FirstPage: 30, V1LastPage: 33},
		PageCount: 2,
	}
	got := Labels([]RegionPaging{rp}, LabelOptions{})
	assert.Equal(t, []string{"30", "31"}, got)
}

func TestLabels_DeletePlaceholderLabeledWithFirstPage(t *testing.T) {
	rp := RegionPaging{
		Region:    Region{Kind: RegionDelete, V1FirstPage: 40, V1LastPage: 42},
		PageCount: 1,
	}
	got := Labels([]RegionPaging{rp}, LabelOptions{})
	assert.Equal(t, []string{"40"}, got)
}

func TestLabels_MixedPlanOrdersByRegion(t *testing.T) {
	plan := []RegionPaging{
		{Region: Region{Kind: RegionInsert, V1FirstPage: 5, V1LastPage: 5}, PageCount: 2},
		{Region: Region{Kind: RegionReplace, V1FirstPage: 10, V1LastPage: 11}, PageCount: 2},
		{Region: Region{Kind: RegionDelete, V1FirstPage: 20, V1LastPage: 22}, PageCount: 1},
	}
	got := Labels(plan, LabelOptions{})
	assert.Equal(t, []string{"5A", "5B", "10", "11", "20"}, got)
}

func TestFormatter_FallsBackToNaturalNumberOutOfRange(t *testing.T) {
	f := Formatter([]string{"23A", "23B"})
	assert.Equal(t, "23A", f(1))
	assert.Equal(t, "23B", f(2))
	assert.Equal(t, "3", f(3), "out-of-range should fall back to natural page number")
	assert.Equal(t, "0", f(0), "non-positive should fall back to natural page number")
}
