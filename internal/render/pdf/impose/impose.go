// Package impose composes condensed PDF logical pages onto landscape sheets.
//
// Booklet ordering follows the standard saddle-stitch scheme. For N pages
// padded to a multiple of 4, physical sheet M (0-indexed) holds:
//
//	front left  = N - 2*M
//	front right = 2*M + 1
//	back left   = 2*M + 2
//	back right  = N - 2*M - 1
//
// N is padded to the next multiple of 4 that is >= pageCount + 1, so
// multiple-of-4 inputs still pick up a full extra sheet of trailing blanks
// and the back-cover slot never carries content.
package impose

import (
	"fmt"
	"io"

	"github.com/go-pdf/fpdf"
	"github.com/go-pdf/fpdf/contrib/gofpdi"
	"github.com/jscaltreto/downstage/internal/render"
	realgofpdi "github.com/phpdave11/gofpdi"
)

// TwoUp imposes two logical pages per landscape sheet.
func TwoUp(src io.ReadSeeker, sheet render.Dimensions, dst io.Writer) error {
	imp, out, rs, pageCount, landscapeW, landscapeH, err := setup(src, sheet)
	if err != nil {
		return fmt.Errorf("impose.TwoUp: %w", err)
	}

	halfW := landscapeW / 2
	// Source condensed pages are halfW × landscapeH; target cells match.
	for i := 0; i < (pageCount+1)/2; i++ {
		out.AddPage()
		placeFitted(out, imp, &rs, 2*i+1, pageCount, 0, 0, halfW, landscapeH, halfW, landscapeH)
		placeFitted(out, imp, &rs, 2*i+2, pageCount, halfW, 0, halfW, landscapeH, halfW, landscapeH)
	}

	if err := out.Output(dst); err != nil {
		return fmt.Errorf("impose.TwoUp: write: %w", err)
	}
	return nil
}

// Booklet imposes logical pages into duplex booklet order.
func Booklet(src io.ReadSeeker, sheet render.Dimensions, gutterMM float64, dst io.Writer) error {
	if gutterMM < 0 {
		return fmt.Errorf("impose.Booklet: negative gutter %.2fmm", gutterMM)
	}
	imp, out, rs, pageCount, landscapeW, landscapeH, err := setup(src, sheet)
	if err != nil {
		return fmt.Errorf("impose.Booklet: %w", err)
	}

	// A gutter at or above the landscape width leaves zero-width cells.
	maxGutterMM := landscapeW
	if gutterMM >= maxGutterMM {
		return fmt.Errorf("impose.Booklet: gutter %.2fmm is too large for sheet (max %.2fmm)", gutterMM, maxGutterMM)
	}

	// +1 before rounding up to a multiple of 4 keeps the back-cover slot
	// blank even when pageCount is itself a multiple of 4.
	N := pageCount + 1
	if rem := N % 4; rem != 0 {
		N += 4 - rem
	}

	halfW := landscapeW / 2
	gutterHalf := gutterMM / 2
	leftW := halfW - gutterHalf
	rightX := halfW + gutterHalf
	rightW := halfW - gutterHalf

	// Source condensed pages are halfW × landscapeH. Placing them into a
	// narrower cell without this source size would stretch the content
	// (gofpdi fills the rectangle in both dimensions).
	sourceW, sourceH := halfW, landscapeH

	sheets := N / 4
	for m := 0; m < sheets; m++ {
		out.AddPage()
		placeFitted(out, imp, &rs, N-2*m, pageCount, 0, 0, leftW, landscapeH, sourceW, sourceH)
		placeFitted(out, imp, &rs, 2*m+1, pageCount, rightX, 0, rightW, landscapeH, sourceW, sourceH)

		out.AddPage()
		placeFitted(out, imp, &rs, 2*m+2, pageCount, 0, 0, leftW, landscapeH, sourceW, sourceH)
		placeFitted(out, imp, &rs, N-2*m-1, pageCount, rightX, 0, rightW, landscapeH, sourceW, sourceH)
	}

	if err := out.Output(dst); err != nil {
		return fmt.Errorf("impose.Booklet: write: %w", err)
	}
	return nil
}

func setup(src io.ReadSeeker, sheet render.Dimensions) (*gofpdi.Importer, *fpdf.Fpdf, io.ReadSeeker, int, float64, float64, error) {
	if sheet.WidthMM <= 0 || sheet.HeightMM <= 0 {
		return nil, nil, nil, 0, 0, 0, fmt.Errorf("invalid sheet %.1fx%.1fmm", sheet.WidthMM, sheet.HeightMM)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, nil, nil, 0, 0, 0, fmt.Errorf("rewind source: %w", err)
	}
	// Raw gofpdi for GetNumPages — the fpdf/contrib wrapper doesn't
	// re-export it, so we use a throwaway importer for the page count
	// and a fresh one below for the actual imports.
	probe := realgofpdi.NewImporter()
	rs := src
	probe.SetSourceStream(&rs)
	pageCount := probe.GetNumPages()
	if pageCount <= 0 {
		return nil, nil, nil, 0, 0, 0, fmt.Errorf("unable to read source page count")
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, nil, nil, 0, 0, 0, fmt.Errorf("rewind after count: %w", err)
	}

	// fpdf's NewCustom expects Size in portrait and swaps internally when
	// OrientationStr is "L". The landscapeW/H we return are already swapped
	// for callers that compute cell geometry.
	landscapeW := sheet.HeightMM
	landscapeH := sheet.WidthMM
	imp := gofpdi.NewImporter()
	out := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		Size:           fpdf.SizeType{Wd: sheet.WidthMM, Ht: sheet.HeightMM},
	})
	return imp, out, rs, pageCount, landscapeW, landscapeH, nil
}

// placeFitted imports logical page n and places it into the target cell
// at (cellX, cellY) with size (cellW, cellH), scaling uniformly to
// preserve the source aspect ratio (sourceW × sourceH) and centering the
// result in the cell. This avoids horizontal squish when the gutter makes
// the cell narrower than the source page, at the cost of some top/bottom
// white space on the imposed sheet.
func placeFitted(out *fpdf.Fpdf, imp *gofpdi.Importer, rs *io.ReadSeeker, n, realPages int, cellX, cellY, cellW, cellH, sourceW, sourceH float64) {
	if n < 1 || n > realPages {
		return
	}
	w, h, dx, dy := fitUniform(sourceW, sourceH, cellW, cellH)
	tpl := imp.ImportPageFromStream(out, rs, n, "/MediaBox")
	imp.UseImportedTemplate(out, tpl, cellX+dx, cellY+dy, w, h)
}

// fitUniform returns the largest w × h that fits inside cellW × cellH while
// preserving sourceW:sourceH, along with the (dx, dy) offset to center the
// fitted box within the cell.
func fitUniform(sourceW, sourceH, cellW, cellH float64) (w, h, dx, dy float64) {
	if sourceW <= 0 || sourceH <= 0 || cellW <= 0 || cellH <= 0 {
		return cellW, cellH, 0, 0
	}
	sourceAspect := sourceW / sourceH
	cellAspect := cellW / cellH
	if cellAspect <= sourceAspect {
		// Cell is taller relative to source → fit to width, pillar-center vertically.
		w = cellW
		h = cellW / sourceAspect
	} else {
		// Cell is wider relative to source → fit to height, letter-center horizontally.
		h = cellH
		w = cellH * sourceAspect
	}
	dx = (cellW - w) / 2
	dy = (cellH - h) / 2
	return
}
