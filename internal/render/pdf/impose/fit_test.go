package impose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFitUniform_NoScalingWhenCellMatchesSource(t *testing.T) {
	w, h, dx, dy := fitUniform(139.7, 215.9, 139.7, 215.9)
	assert.InDelta(t, 139.7, w, 0.001)
	assert.InDelta(t, 215.9, h, 0.001)
	assert.InDelta(t, 0, dx, 0.001)
	assert.InDelta(t, 0, dy, 0.001)
}

func TestFitUniform_FitsToWidthWhenCellIsNarrower(t *testing.T) {
	// Half-letter source (139.7 × 215.9) into a cell narrowed by a 10 mm gutter.
	// Uniform scale factor = 134.7 / 139.7 ≈ 0.964.
	w, h, dx, dy := fitUniform(139.7, 215.9, 134.7, 215.9)
	expectedH := 215.9 * (134.7 / 139.7)
	assert.InDelta(t, 134.7, w, 0.001, "scaled width matches cell width")
	assert.InDelta(t, expectedH, h, 0.001, "height scales by the same factor as width")
	assert.InDelta(t, 0, dx, 0.001, "no horizontal offset — full-width fit")
	assert.InDelta(t, (215.9-expectedH)/2, dy, 0.001, "vertical offset centers the scaled content")
	// The source aspect (139.7/215.9) must survive scaling.
	assert.InDelta(t, 139.7/215.9, w/h, 0.0001, "aspect ratio preserved")
}

func TestFitUniform_FitsToHeightWhenCellIsShorter(t *testing.T) {
	// Source taller than cell relative to their widths.
	w, h, dx, dy := fitUniform(100, 200, 120, 150)
	// source aspect = 100/200 = 0.5; cell aspect = 120/150 = 0.8; cell wider → fit to height.
	assert.InDelta(t, 150, h, 0.001)
	assert.InDelta(t, 75, w, 0.001) // 150 * 0.5
	assert.InDelta(t, (120.0-75.0)/2, dx, 0.001)
	assert.InDelta(t, 0, dy, 0.001)
}

func TestFitUniform_DegenerateInputsReturnCell(t *testing.T) {
	w, h, dx, dy := fitUniform(0, 0, 100, 100)
	assert.Equal(t, 100.0, w)
	assert.Equal(t, 100.0, h)
	assert.Equal(t, 0.0, dx)
	assert.Equal(t, 0.0, dy)
}
