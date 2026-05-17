package revisions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/diff"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/jscaltreto/downstage/internal/render/pdf/pagemap"
)

// Options configures the revision-pages run.
type Options struct {
	// V1Source is the prior version's .ds source.
	V1Source []byte
	// V2Source is the new version's .ds source.
	V2Source []byte
	// V1Name / V2Name are display names used in diagnostics; the file path
	// or "<stdin>" is appropriate.
	V1Name string
	V2Name string

	// Config is the renderer config; PageSize / Style / margins all flow
	// through to the revisions render.
	Config render.Config

	// AnchorWindow merges adjacent non-Equal hunks separated by ≤ N Equal
	// blocks into a single region. 0 means use the package default (4).
	AnchorWindow int

	// MarkChanges turns on right-margin asterisks on changed blocks.
	MarkChanges bool

	// IncludeRemovedMarkers, when true, leaves room for a REMOVED placeholder
	// page at the trailing edge of an under-replaced region.
	IncludeRemovedMarkers bool

	// PageNumbering controls how revision pages are labeled.
	PageNumbering PageNumberingMode
}

// PageNumberingMode chooses how revision pages are labeled.
type PageNumberingMode int

const (
	// PageNumberingV1Labels (default): pages keep v1 numbers with A/B/C
	// suffixes per the Hollywood revision-pages convention.
	PageNumberingV1Labels PageNumberingMode = iota
	// PageNumberingNatural: pages are numbered 1, 2, 3, … like a normal
	// render. Useful when v1 is no longer relevant or for testing.
	PageNumberingNatural
	// PageNumberingNone: footer is suppressed entirely.
	PageNumberingNone
)

// ErrNoDifferences is returned when v1 and v2 contain no diffable changes.
var ErrNoDifferences = errors.New("no differences detected between v1 and v2")

// Render orchestrates the full revision-pages workflow and writes the
// resulting PDF to w. The Options.Config style and page size flow through;
// unrelated fields on Config are overwritten by this function.
func Render(w io.Writer, opts Options) error {
	v1Doc, err := parse(opts.V1Source, opts.V1Name)
	if err != nil {
		return err
	}
	v2Doc, err := parse(opts.V2Source, opts.V2Name)
	if err != nil {
		return err
	}

	v1Blocks := diff.FlattenedBlocks(v1Doc, diff.CanonicalNameMap(v1Doc))
	v2Blocks := diff.FlattenedBlocks(v2Doc, diff.CanonicalNameMap(v2Doc))

	v1Pages, err := renderPageMap(v1Doc, opts.Config)
	if err != nil {
		return fmt.Errorf("rendering v1 for page map: %w", err)
	}

	hunks := diff.Diff(v1Blocks, v2Blocks)
	planOpts := PlanOptions{AnchorWindow: opts.AnchorWindow}
	regions := Plan(v1Blocks, v2Blocks, hunks, v1Pages, planOpts)

	if len(regions) == 0 {
		return ErrNoDifferences
	}

	synthDoc, anchors := SynthesizeDocument(regions, SynthOptions{})

	paging, err := measureRegionPaging(synthDoc, anchors, opts.Config)
	if err != nil {
		return fmt.Errorf("measuring revision pagination: %w", err)
	}
	for i := range paging {
		paging[i].Region = regions[i]
	}

	trailingPlaceholders := map[int]string{}
	if opts.IncludeRemovedMarkers {
		for i, rp := range paging {
			r := rp.Region
			if r.Kind != RegionReplace {
				continue
			}
			span := r.V1LastPage - r.V1FirstPage + 1
			if span <= rp.PageCount {
				continue
			}
			leftoverStart := r.V1FirstPage + rp.PageCount
			leftoverEnd := r.V1LastPage
			var note string
			if leftoverStart == leftoverEnd {
				note = fmt.Sprintf("REMOVED — p. %d intentionally removed in this revision", leftoverStart)
			} else {
				note = fmt.Sprintf("REMOVED — pp. %d–%d intentionally removed in this revision", leftoverStart, leftoverEnd)
			}
			trailingPlaceholders[i] = note
			paging[i].PageCount++
		}
		if len(trailingPlaceholders) > 0 {
			synthDoc, _ = SynthesizeDocument(regions, SynthOptions{TrailingPlaceholders: trailingPlaceholders})
		}
	}

	changed := map[any]bool{}
	for _, r := range regions {
		for k := range r.ChangedNodes {
			changed[k] = true
		}
	}

	labelOpts := LabelOptions{IncludeRemovedMarkers: opts.IncludeRemovedMarkers}
	labels := Labels(paging, labelOpts)
	pageToRegion := buildPageRegionIndex(paging)

	cfg := opts.Config
	cfg.SkipOutline = true
	if opts.MarkChanges {
		cfg.MarkChangedBlocks = changed
	}

	switch opts.PageNumbering {
	case PageNumberingNone:
		cfg.PageLabelFormatter = func(int) string { return "" }
	case PageNumberingNatural:
		cfg.PageLabelFormatter = nil
	default:
		cfg.PageLabelFormatter = Formatter(labels)
	}

	cfg.PageHeaderFn = func(internalPage int) string {
		idx := internalPage - 1
		if idx < 0 || idx >= len(pageToRegion) {
			return ""
		}
		r := regions[pageToRegion[idx]]
		parts := []string{r.MarginNote}
		if r.ContextHeading != "" {
			parts = append(parts, r.ContextHeading)
		}
		return strings.Join(parts, "  •  ")
	}

	nr := pdf.NewRenderer(cfg)
	if err := render.Walk(nr, synthDoc, w); err != nil {
		return fmt.Errorf("rendering revisions: %w", err)
	}
	return nil
}

func parse(src []byte, name string) (*ast.Document, error) {
	doc, errs := parser.Parse(src)
	if len(errs) > 0 {
		base := name
		if base == "" {
			base = "<input>"
		} else {
			base = filepath.Base(name)
		}
		first := errs[0]
		return nil, fmt.Errorf("%s:%d:%d: %s",
			base,
			first.Range.Start.Line+1,
			first.Range.Start.Column+1,
			first.Message,
		)
	}
	return doc, nil
}

func renderPageMap(doc *ast.Document, cfg render.Config) (pagemap.Map, error) {
	rec := pagemap.NewRecorder()
	cfg = revisionCaptureConfig(cfg)
	cfg.RecordPageMap = rec
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	nr := pdf.NewRenderer(cfg)
	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return nil, err
	}
	return rec.Map(), nil
}

// measureRegionPaging runs the synthetic doc through the renderer once to
// capture the page span of each region's anchor node. Returns one
// RegionPaging per region in input order. Region fields other than
// PageCount are zero-valued; the caller fills them from the original plan.
func measureRegionPaging(synthDoc *ast.Document, anchors []ast.Node, cfg render.Config) ([]RegionPaging, error) {
	rec := pagemap.NewRecorder()
	cfg = revisionCaptureConfig(cfg)
	cfg.RecordPageMap = rec
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	nr := pdf.NewRenderer(cfg)
	var buf bytes.Buffer
	if err := render.Walk(nr, synthDoc, &buf); err != nil {
		return nil, err
	}
	m := rec.Map()
	lastPage := m.LastPage()
	out := make([]RegionPaging, len(anchors))
	for i, a := range anchors {
		span, ok := m.Lookup(a)
		startPage := 1
		if ok {
			startPage = span.Start
		}
		next := lastPage + 1
		if i+1 < len(anchors) {
			if nspan, ok := m.Lookup(anchors[i+1]); ok {
				next = nspan.Start
			}
		}
		count := next - startPage
		if count < 1 {
			count = 1
		}
		out[i].PageCount = count
	}
	return out, nil
}

func revisionCaptureConfig(cfg render.Config) render.Config {
	cfg.PageLabelFormatter = nil
	cfg.PageHeaderFn = nil
	cfg.MarkChangedBlocks = nil
	cfg.SkipOutline = true
	return cfg
}

// buildPageRegionIndex returns a slice of length total-internal-pages where
// each entry is the region index that owns that internal page.
func buildPageRegionIndex(paging []RegionPaging) []int {
	total := 0
	for _, rp := range paging {
		total += rp.PageCount
	}
	out := make([]int, total)
	idx := 0
	for i, rp := range paging {
		for j := 0; j < rp.PageCount; j++ {
			out[idx] = i
			idx++
		}
	}
	return out
}
