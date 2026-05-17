package revisions

import (
	"fmt"
	"strings"
)

// RegionPaging pairs a planned Region with the number of internal fpdf pages
// it produced during rendering.
type RegionPaging struct {
	Region    Region
	PageCount int
}

// Labels returns one label per internal page in document order. Pass the
// resulting slice to a PageLabelFormatter closure that indexes by
// internalPage-1.
//
// Numbering follows the Hollywood revision-pages convention:
//
//   - Insert after v1 page N: NA, NB, NC, …
//   - Replace [N, M], v2 produces p pages:
//   - p ≤ M-N+1: N, N+1, …; trailing slots (if --no-removed-marker is off)
//     absorb a REMOVED placeholder labeled with the lowest leftover number.
//   - p > M-N+1: N, NA, NB, …; subsequent overflow continues with letter
//     suffixes off of N.
//   - Delete [N, M]: a single placeholder labeled N.
//
// The opts.RemovedMarker flag controls whether REMOVED placeholders are
// included (caller responsibility for actually rendering them); the numbering
// scheme leaves a slot for them when applicable.
type LabelOptions struct {
	// IncludeRemovedMarkers, when true, leaves room for a REMOVED placeholder
	// page at the end of an under-replaced region. The orchestrator is
	// responsible for emitting the placeholder content into the PDF; this
	// function only reserves the label.
	IncludeRemovedMarkers bool
}

func Labels(paging []RegionPaging, opts LabelOptions) []string {
	var out []string
	for _, rp := range paging {
		out = append(out, regionLabels(rp, opts)...)
	}
	return out
}

func regionLabels(rp RegionPaging, opts LabelOptions) []string {
	r := rp.Region
	p := rp.PageCount

	switch r.Kind {
	case RegionInsert:
		// Labels NA, NB, NC, …, anchored to V1FirstPage.
		out := make([]string, p)
		for i := 0; i < p; i++ {
			out[i] = fmt.Sprintf("%d%s", r.V1FirstPage, letterSuffix(i+1))
		}
		return out

	case RegionDelete:
		// Single placeholder labeled with the first replaced v1 page.
		out := make([]string, p)
		for i := 0; i < p; i++ {
			out[i] = fmt.Sprintf("%d", r.V1FirstPage)
		}
		return out

	case RegionReplace:
		span := r.V1LastPage - r.V1FirstPage + 1
		if span < 1 {
			span = 1
		}
		out := make([]string, 0, p)
		if p <= span {
			// Sufficient slots — sequential numbering.
			for i := 0; i < p; i++ {
				out = append(out, fmt.Sprintf("%d", r.V1FirstPage+i))
			}
			if opts.IncludeRemovedMarkers && p < span {
				// The orchestrator inserts one REMOVED page; label it with
				// the lowest leftover v1 page so binder operators know what
				// it covers.
				out = append(out, fmt.Sprintf("%d", r.V1FirstPage+p))
			}
		} else {
			// Overflow — keep N labeled normally, then letter-suffix the
			// rest off of N.
			out = append(out, fmt.Sprintf("%d", r.V1FirstPage))
			for i := 1; i < p; i++ {
				out = append(out, fmt.Sprintf("%d%s", r.V1FirstPage, letterSuffix(i)))
			}
		}
		return out
	}
	return nil
}

// letterSuffix maps 1→"A", 2→"B", …, 26→"Z", 27→"AA", 28→"AB", … in the
// Excel-column style that revision pages use when more than 26 inserts
// stack up.
func letterSuffix(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for n > 0 {
		n--
		b.WriteByte(byte('A' + n%26))
		n /= 26
	}
	// Reverse.
	r := b.String()
	out := make([]byte, len(r))
	for i := range r {
		out[len(r)-1-i] = r[i]
	}
	return string(out)
}

// Formatter returns a closure suitable for plugging into
// render.Config.PageLabelFormatter. internalPage is 1-indexed.
func Formatter(labels []string) func(int) string {
	return func(internalPage int) string {
		idx := internalPage - 1
		if idx < 0 || idx >= len(labels) {
			return fmt.Sprintf("%d", internalPage)
		}
		return labels[idx]
	}
}
