package impose_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/jscaltreto/downstage/internal/render/pdf/impose"
	"github.com/phpdave11/gofpdi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderCondensed(t *testing.T, body string, pageSize render.PageSize) []byte {
	t.Helper()
	doc, errs := parser.Parse([]byte(body))
	require.Empty(t, errs)

	cfg := render.DefaultConfig()
	cfg.Style = render.StyleCondensed
	cfg.PageSize = pageSize

	nr := pdf.NewCondensedRenderer(cfg)
	var buf bytes.Buffer
	require.NoError(t, render.Walk(nr, doc, &buf))
	return buf.Bytes()
}

func pdfPageCount(t *testing.T, data []byte) int {
	t.Helper()
	imp := gofpdi.NewImporter()
	var rs io.ReadSeeker = bytes.NewReader(data)
	imp.SetSourceStream(&rs)
	n := imp.GetNumPages()
	require.Greater(t, n, 0)
	return n
}

// buildFixture synthesizes a .ds source that produces approximately the
// requested number of condensed logical pages. Each cue takes roughly
// half a logical page given the default condensed layout, so we scale
// accordingly.
func buildFixture(pages int) string {
	var b strings.Builder
	b.WriteString("# Test Play\n\n## ACT I\n\n### SCENE 1\n\n")
	for i := 0; i < pages*20; i++ {
		fmt.Fprintf(&b, "ALICE\nLine number %d of dialogue that takes up some vertical space.\n\n", i)
	}
	return b.String()
}

func TestTwoUpPageCount(t *testing.T) {
	tests := []struct {
		name     string
		pages    int
		pageSize render.PageSize
		sheet    render.Dimensions
	}{
		{"letter 3 pages", 3, render.PageLetter, render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}},
		{"letter 5 pages", 5, render.PageLetter, render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}},
		{"letter 8 pages", 8, render.PageLetter, render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}},
		{"a4 5 pages", 5, render.PageA4, render.Dimensions{WidthMM: 210, HeightMM: 297}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := renderCondensed(t, buildFixture(tt.pages), tt.pageSize)
			n := pdfPageCount(t, src)

			var out bytes.Buffer
			require.NoError(t, impose.TwoUp(bytes.NewReader(src), tt.sheet, &out))
			assert.Equal(t, "%PDF-", string(out.Bytes()[:5]))

			expected := (n + 1) / 2
			assert.Equal(t, expected, pdfPageCount(t, out.Bytes()),
				"2-up should produce ceil(%d/2)=%d output PDF pages", n, expected)
		})
	}
}

func TestBookletPageCount(t *testing.T) {
	tests := []struct {
		name     string
		pages    int
		pageSize render.PageSize
		sheet    render.Dimensions
	}{
		{"letter 3 pages", 3, render.PageLetter, render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}},
		{"letter 5 pages", 5, render.PageLetter, render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}},
		{"letter 8 pages", 8, render.PageLetter, render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}},
		{"a4 5 pages", 5, render.PageA4, render.Dimensions{WidthMM: 210, HeightMM: 297}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := renderCondensed(t, buildFixture(tt.pages), tt.pageSize)
			n := pdfPageCount(t, src)

			var out bytes.Buffer
			require.NoError(t, impose.Booklet(bytes.NewReader(src), tt.sheet, 3.175, &out))
			assert.Equal(t, "%PDF-", string(out.Bytes()[:5]))

			// Padded page count reserves the back cover as blank, so we
			// round up n+1 (not n) to the next multiple of 4.
			expected := 2 * ((n + 4) / 4)
			assert.Equal(t, expected, pdfPageCount(t, out.Bytes()),
				"booklet should produce 2*ceil((%d+1)/4)=%d output PDF pages", n, expected)
		})
	}
}

// TestBookletReservesBlankBackCover checks the invariant that the
// back-cover slot (logical page N_padded, which lands on the front-left
// of sheet 0) is always a padding slot, so the back cover never carries
// content when the booklet is unfolded.
func TestBookletReservesBlankBackCover(t *testing.T) {
	sheet := render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}

	// Each fixture produces a different logical page count; across the
	// set we cover n % 4 ∈ {0, 1, 2, 3} so the multiple-of-4 case is
	// exercised even without knowing the exact page count up front.
	for _, pages := range []int{3, 5, 7, 8, 10} {
		t.Run(fmt.Sprintf("%d_pages", pages), func(t *testing.T) {
			src := renderCondensed(t, buildFixture(pages), render.PageLetter)
			n := pdfPageCount(t, src)

			var out bytes.Buffer
			require.NoError(t, impose.Booklet(bytes.NewReader(src), sheet, 3.175, &out))
			outputPages := pdfPageCount(t, out.Bytes())

			// Each physical sheet contributes 2 output PDF pages and
			// holds 4 logical pages. So the padded logical count is
			// 2 * outputPages.
			nPadded := 2 * outputPages
			assert.Greater(t, nPadded, n,
				"padded page count %d must exceed logical page count %d so the back cover is blank",
				nPadded, n)
			assert.Equal(t, 0, nPadded%4,
				"padded page count %d must be a multiple of 4", nPadded)
		})
	}
}

func TestBookletNoGutter(t *testing.T) {
	src := renderCondensed(t, buildFixture(5), render.PageLetter)
	var out bytes.Buffer
	require.NoError(t, impose.Booklet(bytes.NewReader(src), render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}, 0, &out))
	assert.Equal(t, "%PDF-", string(out.Bytes()[:5]))
}

func TestBookletRejectsNegativeGutter(t *testing.T) {
	src := renderCondensed(t, buildFixture(2), render.PageLetter)
	var out bytes.Buffer
	err := impose.Booklet(bytes.NewReader(src), render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}, -1, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative gutter")
}

func TestBookletRejectsGutterLargerThanSheet(t *testing.T) {
	src := renderCondensed(t, buildFixture(2), render.PageLetter)
	// Landscape Letter width = 279.4mm; a gutter at that value would
	// leave zero-width cells.
	sheet := render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}
	var out bytes.Buffer
	err := impose.Booklet(bytes.NewReader(src), sheet, 280, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large for sheet")
}

// TestTwoUpDoesNotBloatOutputSize guards against a regression where each
// gofpdi page import would re-emit every form XObject seen so far, yielding
// O(N²) duplicate templates in the output PDF. A 40-page condensed source
// previously produced an ~9 MB 2-up imposition that re-deduped back to
// ~270 KB through a roundtrip rewriter — assert that the imposed file is
// at most a small constant factor of the source size.
func TestTwoUpDoesNotBloatOutputSize(t *testing.T) {
	src := renderCondensed(t, buildFixture(40), render.PageLetter)
	sheet := render.Dimensions{WidthMM: 215.9, HeightMM: 279.4}

	var out bytes.Buffer
	require.NoError(t, impose.TwoUp(bytes.NewReader(src), sheet, &out))

	// The imposed PDF embeds each source page once as a form XObject plus
	// shared resources. 3× the source size is a generous ceiling; the
	// pre-fix output was ~36× larger.
	maxRatio := 3.0
	ratio := float64(out.Len()) / float64(len(src))
	assert.Less(t, ratio, maxRatio,
		"imposed PDF %d bytes is %.1f× source %d bytes (max %.1f×) — likely a return of the O(N²) gofpdi bridge duplication",
		out.Len(), ratio, len(src), maxRatio)
}

func TestImposeRejectsInvalidSheet(t *testing.T) {
	src := renderCondensed(t, buildFixture(2), render.PageLetter)
	badSheet := render.Dimensions{WidthMM: 0, HeightMM: 0}

	var out bytes.Buffer
	err := impose.TwoUp(bytes.NewReader(src), badSheet, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sheet")
}
