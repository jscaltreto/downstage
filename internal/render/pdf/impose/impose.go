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
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/phpdave11/gofpdi"
)

// TwoUp imposes two logical pages per landscape sheet.
func TwoUp(src io.ReadSeeker, sheet render.Dimensions, dst io.Writer) error {
	ctx, err := setup(src, sheet)
	if err != nil {
		return fmt.Errorf("impose.TwoUp: %w", err)
	}

	halfW := ctx.landscapeW / 2
	// Source condensed pages are halfW × landscapeH; target cells match.
	for i := 0; i < (ctx.pageCount+1)/2; i++ {
		ctx.out.AddPage()
		ctx.placeFitted(2*i+1, 0, 0, halfW, ctx.landscapeH, halfW, ctx.landscapeH)
		ctx.placeFitted(2*i+2, halfW, 0, halfW, ctx.landscapeH, halfW, ctx.landscapeH)
	}

	if err := ctx.finalize(dst); err != nil {
		return fmt.Errorf("impose.TwoUp: %w", err)
	}
	return nil
}

// Booklet imposes logical pages into duplex booklet order.
func Booklet(src io.ReadSeeker, sheet render.Dimensions, gutterMM float64, dst io.Writer) error {
	if gutterMM < 0 {
		return fmt.Errorf("impose.Booklet: negative gutter %.2fmm", gutterMM)
	}
	ctx, err := setup(src, sheet)
	if err != nil {
		return fmt.Errorf("impose.Booklet: %w", err)
	}

	// A gutter at or above the landscape width leaves zero-width cells.
	maxGutterMM := ctx.landscapeW
	if gutterMM >= maxGutterMM {
		return fmt.Errorf("impose.Booklet: gutter %.2fmm is too large for sheet (max %.2fmm)", gutterMM, maxGutterMM)
	}

	// +1 before rounding up to a multiple of 4 keeps the back-cover slot
	// blank even when pageCount is itself a multiple of 4.
	N := ctx.pageCount + 1
	if rem := N % 4; rem != 0 {
		N += 4 - rem
	}

	halfW := ctx.landscapeW / 2
	gutterHalf := gutterMM / 2
	leftW := halfW - gutterHalf
	rightX := halfW + gutterHalf
	rightW := halfW - gutterHalf

	// Source condensed pages are halfW × landscapeH. Placing them into a
	// narrower cell without this source size would stretch the content
	// (gofpdi fills the rectangle in both dimensions).
	sourceW, sourceH := halfW, ctx.landscapeH

	sheets := N / 4
	for m := range sheets {
		ctx.out.AddPage()
		ctx.placeFitted(N-2*m, 0, 0, leftW, ctx.landscapeH, sourceW, sourceH)
		ctx.placeFitted(2*m+1, rightX, 0, rightW, ctx.landscapeH, sourceW, sourceH)

		ctx.out.AddPage()
		ctx.placeFitted(2*m+2, 0, 0, leftW, ctx.landscapeH, sourceW, sourceH)
		ctx.placeFitted(N-2*m-1, rightX, 0, rightW, ctx.landscapeH, sourceW, sourceH)
	}

	if err := ctx.finalize(dst); err != nil {
		return fmt.Errorf("impose.Booklet: %w", err)
	}
	return nil
}

// imposeCtx threads the importer, output document, and source geometry
// through one imposition pass. Lifecycle: setup → placeFitted (any number
// of times) → finalize. finalize defers the gofpdi→fpdf object exchange
// to a single call after every placement is queued. The fpdf/contrib/gofpdi
// bridge does that exchange on each imported page instead, which causes
// PutFormXobjectsUnordered to emit every accumulated form XObject again
// each time — quadratic growth in the output file size.
type imposeCtx struct {
	imp        *gofpdi.Importer
	out        *fpdf.Fpdf
	pageCount  int
	landscapeW float64
	landscapeH float64
}

func setup(src io.ReadSeeker, sheet render.Dimensions) (*imposeCtx, error) {
	if sheet.WidthMM <= 0 || sheet.HeightMM <= 0 {
		return nil, fmt.Errorf("invalid sheet %.1fx%.1fmm", sheet.WidthMM, sheet.HeightMM)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind source: %w", err)
	}

	imp := gofpdi.NewImporter()
	rs := src
	imp.SetSourceStream(&rs)
	pageCount := imp.GetNumPages()
	if pageCount <= 0 {
		return nil, fmt.Errorf("unable to read source page count")
	}

	// fpdf's NewCustom expects Size in portrait and swaps internally when
	// OrientationStr is "L". The landscapeW/H we return are already swapped
	// for callers that compute cell geometry.
	out := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		Size:           fpdf.SizeType{Wd: sheet.WidthMM, Ht: sheet.HeightMM},
	})
	return &imposeCtx{
		imp:        imp,
		out:        out,
		pageCount:  pageCount,
		landscapeW: sheet.HeightMM,
		landscapeH: sheet.WidthMM,
	}, nil
}

// placeFitted imports logical page n (idempotent — gofpdi caches the
// template) and places it into the cell at (cellX, cellY) with size
// (cellW, cellH), scaling uniformly to preserve the source aspect ratio
// (sourceW × sourceH) and centering the result in the cell. This avoids
// horizontal squish when the gutter makes the cell narrower than the
// source page, at the cost of some top/bottom white space on the imposed
// sheet.
func (c *imposeCtx) placeFitted(n int, cellX, cellY, cellW, cellH, sourceW, sourceH float64) {
	if n < 1 || n > c.pageCount {
		return
	}
	w, h, dx, dy := fitUniform(sourceW, sourceH, cellW, cellH)
	tpl := c.imp.ImportPage(n, "/MediaBox")
	name, sx, sy, tx, ty := c.imp.UseTemplate(tpl, cellX+dx, cellY+dy, w, h)
	c.out.UseImportedTemplate(name, sx, sy, tx, ty)
}

// finalize emits all imported objects into the output document exactly
// once, then writes the PDF. See imposeCtx for why this batching is the
// load-bearing piece of the fix.
func (c *imposeCtx) finalize(dst io.Writer) error {
	tplObjIDs := c.imp.PutFormXobjectsUnordered()
	c.out.ImportTemplates(tplObjIDs)
	c.out.ImportObjects(c.imp.GetImportedObjectsUnordered())
	c.out.ImportObjPos(c.imp.GetImportedObjHashPos())
	if err := c.out.Output(dst); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
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
