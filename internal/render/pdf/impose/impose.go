// Package impose composes condensed PDF logical pages onto landscape sheets
// by importing the source pages via gofpdi and placing them on new fpdf
// sheets. See issue #121.
//
// This is a second pass: the condensed renderer emits half-letter/A5 logical
// pages first, then this package imports those pages and arranges two per
// landscape sheet.
//
// Booklet ordering follows the standard saddle-stitch scheme. For N pages
// padded to a multiple of 4, physical sheet M (0-indexed) holds:
//
//	front left  = N - 2*M
//	front right = 2*M + 1
//	back left   = 2*M + 2
//	back right  = N - 2*M - 1
//
// Printing these in order double-sided, then folding the stack in half at
// the gutter, yields a readable booklet.
//
// Padding reserves the back-cover slot (front-left of sheet 0) for a blank
// page, so the back cover is never a content page when the booklet is
// unfolded. The padded page count N is the smallest multiple of 4 that is
// >= pageCount + 1, so N mod 4 == 0 inputs still get a full extra sheet of
// trailing blanks rather than having their last logical page land opposite
// the title on the outer sheet.
package impose

import (
	"fmt"
	"io"

	"github.com/go-pdf/fpdf"
	"github.com/go-pdf/fpdf/contrib/gofpdi"
	"github.com/jscaltreto/downstage/internal/render"
	realgofpdi "github.com/phpdave11/gofpdi"
)

// TwoUp imposes two logical pages per landscape sheet. sheet is the parent
// sheet size as returned by PageSize.SheetDimensions() (portrait orientation;
// the imposed output rotates to landscape). The two imposed cells each match
// the condensed logical page size (half the sheet's longer edge).
func TwoUp(src io.ReadSeeker, sheet render.Dimensions, dst io.Writer) error {
	imp, out, rs, pageCount, landscapeW, landscapeH, err := setup(src, sheet)
	if err != nil {
		return fmt.Errorf("impose.TwoUp: %w", err)
	}

	halfW := landscapeW / 2
	for i := 0; i < (pageCount+1)/2; i++ {
		out.AddPage()
		placePage(out, imp, &rs, 2*i+1, pageCount, 0, 0, halfW, landscapeH)
		placePage(out, imp, &rs, 2*i+2, pageCount, halfW, 0, halfW, landscapeH)
	}

	if err := out.Output(dst); err != nil {
		return fmt.Errorf("impose.TwoUp: write: %w", err)
	}
	return nil
}

// Booklet imposes logical pages into duplex booklet order. The input is
// padded to a multiple of 4 and each physical sheet receives four logical
// pages (two front, two back). gutterMM is the inner gap between the two
// pages on each output sheet.
func Booklet(src io.ReadSeeker, sheet render.Dimensions, gutterMM float64, dst io.Writer) error {
	if gutterMM < 0 {
		return fmt.Errorf("impose.Booklet: negative gutter %.2fmm", gutterMM)
	}
	imp, out, rs, pageCount, landscapeW, landscapeH, err := setup(src, sheet)
	if err != nil {
		return fmt.Errorf("impose.Booklet: %w", err)
	}

	// The gutter sits between two half-sheet cells on a landscape sheet. A
	// gutter at or above half the landscape width would leave zero-or-
	// negative-width cells with no room for content.
	maxGutterMM := landscapeW
	if gutterMM >= maxGutterMM {
		return fmt.Errorf("impose.Booklet: gutter %.2fmm is too large for sheet (max %.2fmm)", gutterMM, maxGutterMM)
	}

	// Pad logical page count up to a multiple of 4, reserving the back
	// cover (last imposed slot) for a blank so it never carries content
	// when the booklet unfolds beside the title page. Add one virtual
	// page before rounding so pageCount values that are already a
	// multiple of 4 pick up a fresh sheet of padding.
	N := pageCount + 1
	if rem := N % 4; rem != 0 {
		N += 4 - rem
	}

	halfW := landscapeW / 2
	gutterHalf := gutterMM / 2
	leftW := halfW - gutterHalf
	rightX := halfW + gutterHalf
	rightW := halfW - gutterHalf

	sheets := N / 4
	for m := 0; m < sheets; m++ {
		// Front
		out.AddPage()
		placePage(out, imp, &rs, N-2*m, pageCount, 0, 0, leftW, landscapeH)
		placePage(out, imp, &rs, 2*m+1, pageCount, rightX, 0, rightW, landscapeH)

		// Back
		out.AddPage()
		placePage(out, imp, &rs, 2*m+2, pageCount, 0, 0, leftW, landscapeH)
		placePage(out, imp, &rs, N-2*m-1, pageCount, rightX, 0, rightW, landscapeH)
	}

	if err := out.Output(dst); err != nil {
		return fmt.Errorf("impose.Booklet: write: %w", err)
	}
	return nil
}

// setup builds the landscape output PDF, the importer, and reads the source
// page count. The returned ReadSeeker is rewound and ready for
// ImportPageFromStream calls. landscapeW and landscapeH are the output
// sheet dimensions in millimeters (a rotation of the portrait sheet input).
func setup(src io.ReadSeeker, sheet render.Dimensions) (*gofpdi.Importer, *fpdf.Fpdf, io.ReadSeeker, int, float64, float64, error) {
	if sheet.WidthMM <= 0 || sheet.HeightMM <= 0 {
		return nil, nil, nil, 0, 0, 0, fmt.Errorf("invalid sheet %.1fx%.1fmm", sheet.WidthMM, sheet.HeightMM)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, nil, nil, 0, 0, 0, fmt.Errorf("rewind source: %w", err)
	}
	// Use a raw gofpdi importer to read the page count; the contrib
	// wrapper doesn't expose GetNumPages.
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

	// fpdf's NewCustom interprets Size in portrait and swaps when
	// OrientationStr is "L"; hand it portrait dims and let it rotate.
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

// placePage imports and places logical page `n` on the output sheet, or
// leaves the cell blank if n is a padding slot (n > realPages).
func placePage(out *fpdf.Fpdf, imp *gofpdi.Importer, rs *io.ReadSeeker, n, realPages int, x, y, w, h float64) {
	if n < 1 || n > realPages {
		return
	}
	tpl := imp.ImportPageFromStream(out, rs, n, "/MediaBox")
	imp.UseImportedTemplate(out, tpl, x, y, w, h)
}
