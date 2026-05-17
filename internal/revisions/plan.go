// Package revisions orchestrates the revision-pages workflow: diffing two
// parsed Downstage documents, mapping the diff to v1's rendered pagination,
// and producing the materials needed by the PDF renderer to emit a small
// "swap-into-binder" revision PDF.
package revisions

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/diff"
	"github.com/jscaltreto/downstage/internal/render/pdf/pagemap"
)

// RegionKind describes the editorial relationship between v1 and v2 for a
// revision region.
type RegionKind int

const (
	// RegionInsert: v1 contributes no blocks; v2 added new content. The
	// printed revision pages get inserted into the v1 binder after
	// V1FirstPage.
	RegionInsert RegionKind = iota
	// RegionDelete: v2 contributes no blocks; v1 content has been removed.
	// The revision PDF emits a single placeholder page covering the
	// deleted range.
	RegionDelete
	// RegionReplace: both sides contribute. The v2 content replaces v1's
	// pages in the binder.
	RegionReplace
)

// Region is the planner's per-revision unit. The orchestrator renders each
// Region's V2Nodes into one or more PDF pages, labels them via the numbering
// formatter, and stamps a margin note describing where they slot into v1.
type Region struct {
	Kind RegionKind
	// V1FirstPage / V1LastPage describe the v1 page range this region
	// replaces (inclusive). For RegionInsert both values point to the v1
	// page after which the inserted pages slot in.
	V1FirstPage int
	V1LastPage  int
	// ContextHeading is the v2 enclosing-section context at the region's
	// first node, e.g. "ACT II — SCENE 3".
	ContextHeading string
	// MarginNote is the writer-facing top-margin annotation: "Insert after
	// p. 22", "Replace pp. 23–24", "Remove pp. 24–26", etc.
	MarginNote string
	// V2Nodes are the AST nodes the renderer should walk for this region.
	// For RegionDelete this slice is empty.
	V2Nodes []ast.Node
	// ChangedNodes is the set of v2 AST node pointers that diffed as
	// non-Equal within this region — used by the renderer to draw
	// right-margin asterisks on changed lines.
	ChangedNodes map[any]bool
}

// PlanOptions tunes region computation.
type PlanOptions struct {
	// AnchorWindow is the largest run of Equal blocks that may sit between
	// two non-Equal hunks before they're merged into a single region. A
	// value of 0 disables merging; the default (used when zero) is 4.
	AnchorWindow int
}

// Plan computes the revision regions for a (v1, v2) document pair given the
// rendered v1 page map.
//
//   - v1Blocks / v2Blocks are diff.FlattenedBlocks of their respective docs.
//   - hunks is the result of diff.Diff(v1Blocks, v2Blocks).
//   - v1Pages is the page-span map produced when v1 was rendered with a
//     pagemap.Recorder.
func Plan(v1Blocks, v2Blocks []diff.Block, hunks []diff.Hunk, v1Pages pagemap.Map, opts PlanOptions) []Region {
	window := opts.AnchorWindow
	if window <= 0 {
		window = 4
	}

	// Collect indices of non-Equal hunks.
	var nonEq []int
	for i, h := range hunks {
		if h.Kind != diff.HunkEqual {
			nonEq = append(nonEq, i)
		}
	}
	if len(nonEq) == 0 {
		return nil
	}

	// Merge adjacent non-Equal hunks separated by ≤ window Equal blocks.
	type group struct{ first, last int }
	var groups []group
	cur := group{first: nonEq[0], last: nonEq[0]}
	for _, idx := range nonEq[1:] {
		// Sum the v1+v2 lengths of the Equal hunks between cur.last and idx.
		gap := 0
		for k := cur.last + 1; k < idx; k++ {
			h := hunks[k]
			eqV1 := h.V1End - h.V1Start
			eqV2 := h.V2End - h.V2Start
			if eqV1 > eqV2 {
				gap += eqV1
			} else {
				gap += eqV2
			}
		}
		if gap <= window {
			cur.last = idx
		} else {
			groups = append(groups, cur)
			cur = group{first: idx, last: idx}
		}
	}
	groups = append(groups, cur)

	regions := make([]Region, 0, len(groups))
	lastPage := v1Pages.LastPage()
	if lastPage == 0 {
		lastPage = 1
	}

	for _, g := range groups {
		// Compute the merged v1 / v2 ranges from first.V1Start to last.V1End
		// and likewise for v2.
		first, last := hunks[g.first], hunks[g.last]
		v1Lo, v1Hi := first.V1Start, last.V1End
		v2Lo, v2Hi := first.V2Start, last.V2End

		// ChangedNodes set: every block in non-Equal sub-hunks of this
		// group, identified by v2 AST pointer. Computed from the bare
		// hunks (before any context expansion) so asterisks stay scoped
		// to the actually-changed content.
		changed := map[any]bool{}
		for k := g.first; k <= g.last; k++ {
			h := hunks[k]
			if h.Kind == diff.HunkEqual {
				continue
			}
			for _, b := range v2Blocks[h.V2Start:h.V2End] {
				key := blockKey(b)
				if key != nil {
					changed[key] = true
				}
			}
		}

		v1FirstPage, v1LastPage := computeV1Pages(v1Blocks, v1Lo, v1Hi, v1Pages, lastPage)
		kind := classify(v1Lo, v1Hi, v2Lo, v2Hi)

		// Promote a mid-page Insert to a Replace of that page. When new v2
		// content is added between two v1 blocks that both live on the
		// same v1 page, v2's pagination of that page actually differs from
		// v1's — the page is split by the new content. Treating it as a
		// bare Insert produces a revision page with only the added lines
		// (no context) and a misleading "Insert after p. N" margin note.
		// Promote so the iterative expansion below pulls in the page's
		// surrounding v1 blocks and the corresponding v2 content.
		if kind == RegionInsert && v1Lo > 0 && v1Lo < len(v1Blocks) {
			prevSpan, prevOK := lookupBlockSpan(v1Blocks[v1Lo-1], v1Pages)
			nextSpan, nextOK := lookupBlockSpan(v1Blocks[v1Lo], v1Pages)
			if prevOK && nextOK && prevSpan.End >= nextSpan.Start {
				insertPage := nextSpan.Start
				if prevSpan.End < insertPage {
					insertPage = prevSpan.End
				}
				// Walk v1 indices outward to cover blocks on insertPage.
				lo := v1Lo - 1
				for lo > 0 {
					span, ok := lookupBlockSpan(v1Blocks[lo-1], v1Pages)
					if !ok || span.End < insertPage {
						break
					}
					lo--
				}
				hi := v1Lo
				for hi < len(v1Blocks) {
					span, ok := lookupBlockSpan(v1Blocks[hi], v1Pages)
					if !ok || span.Start > insertPage {
						break
					}
					hi++
				}
				v1Lo, v1Hi = lo, hi
				v1FirstPage, v1LastPage = insertPage, insertPage
				kind = RegionReplace
			}
		}

		// Expand Replace regions outward to the v1 page boundaries so the
		// rendered revision pages contain a full page of context, not just
		// the bare changed blocks. This matches how Hollywood revision
		// pages actually look: a normal-looking script page where changed
		// lines are marked with asterisks. Pure Inserts (at page
		// boundaries) and Delete regions are left at their natural hunk
		// bounds — Insert pages slot between v1 pages, Delete renders a
		// placeholder.
		if kind == RegionReplace {
			// Iteratively expand and widen the v1 page range until stable.
			//
			// Pass 1: expand v1 indices outward to include any block whose
			//   span overlaps [V1FirstPage, V1LastPage]. This pulls in
			//   blocks that bleed across the trailing boundary.
			// Pass 2: widen V1FirstPage/V1LastPage so they cover the full
			//   span of every included block (so a bleed-block whose tail
			//   sits on page N+1 advances V1LastPage to N+1).
			// Pass 3: re-expand with the new wider range — this pulls in
			//   the *other* blocks on the newly-covered v1 page (the ones
			//   that come after our bleed-block on that page) so we don't
			//   lose their content when the user removes the v1 page.
			// Repeat until no further blocks are pulled in. In dense
			// documents this terminates as soon as the trailing edge lands
			// on a block whose span doesn't extend further. (Capped at 32
			// iterations as a safety net.)
			expV1Lo, expV1Hi := v1Lo, v1Hi
			for iter := 0; iter < 32; iter++ {
				prevLo, prevHi := expV1Lo, expV1Hi
				prevFirst, prevLast := v1FirstPage, v1LastPage
				expV1Lo, expV1Hi = expandToPageBoundaries(v1Blocks, v1Pages, v1FirstPage, v1LastPage, expV1Lo, expV1Hi)
				v1FirstPage, v1LastPage = extendBoundsToCoverBlocks(v1Blocks[expV1Lo:expV1Hi], v1Pages, v1FirstPage, v1LastPage)
				if expV1Lo == prevLo && expV1Hi == prevHi && v1FirstPage == prevFirst && v1LastPage == prevLast {
					break
				}
			}
			v2Lo, v2Hi = mapV1RangeToV2(hunks, expV1Lo, expV1Hi, v2Lo, v2Hi)
		}

		v2Slice := v2Blocks[v2Lo:v2Hi]
		v2Nodes := nodesFrom(v2Slice)

		regions = append(regions, Region{
			Kind:           kind,
			V1FirstPage:    v1FirstPage,
			V1LastPage:     v1LastPage,
			ContextHeading: contextHeading(v2Slice),
			MarginNote:     marginNote(kind, v1FirstPage, v1LastPage, lastPage),
			V2Nodes:        v2Nodes,
			ChangedNodes:   changed,
		})
	}
	return regions
}

// extendBoundsToCoverBlocks returns the v1 page range that fully encloses
// every span in the expanded block slice. Called after expandToPageBoundaries
// so the revision's V1FirstPage / V1LastPage label the *actual* v1 pages
// the revision content occupied — not just the changed-blocks' span.
func extendBoundsToCoverBlocks(blocks []diff.Block, v1Pages pagemap.Map, first, last int) (int, int) {
	for _, b := range blocks {
		span, ok := lookupBlockSpan(b, v1Pages)
		if !ok {
			continue
		}
		if span.Start < first {
			first = span.Start
		}
		if span.End > last {
			last = span.End
		}
	}
	return first, last
}

// expandToPageBoundaries walks outward from a region's natural hunk bounds
// [v1Lo, v1Hi) until it reaches v1 block indices that bound the requested
// v1 page range. The result is a wider v1 slice whose blocks all overlap
// pages [v1FirstPage, v1LastPage].
//
// "Overlap" rather than "fully contained" semantics — a block whose start is
// on V1LastPage but whose tail bleeds onto V1LastPage+1 is still included.
// The reader gets the start of that block in the revision (preserving
// context) at the cost of duplicating its tail on v1's unchanged page
// M+1. This is the lesser evil compared to dropping the block entirely
// and leaving v1 page M+1 starting mid-block with no setup.
func expandToPageBoundaries(v1Blocks []diff.Block, v1Pages pagemap.Map, v1FirstPage, v1LastPage, v1Lo, v1Hi int) (int, int) {
	if len(v1Blocks) == 0 {
		return v1Lo, v1Hi
	}
	lo := v1Lo
	for lo > 0 {
		prev := v1Blocks[lo-1]
		span, ok := lookupBlockSpan(prev, v1Pages)
		if !ok || span.End < v1FirstPage {
			break
		}
		lo--
	}
	hi := v1Hi
	for hi < len(v1Blocks) {
		next := v1Blocks[hi]
		span, ok := lookupBlockSpan(next, v1Pages)
		if !ok || span.Start > v1LastPage {
			break
		}
		hi++
	}
	return lo, hi
}

// mapV1RangeToV2 translates an expanded v1 block range to the corresponding
// v2 indices via the diff hunks. For positions inside an Equal hunk the
// mapping is exact; for positions inside Modify/Delete the v2 boundary is
// snapped to the hunk's edges so the resulting v2 slice still covers every
// v2 block that aligns with the requested v1 range. fallbackLo / fallbackHi
// are returned when a v1 index cannot be located (degenerate input).
func mapV1RangeToV2(hunks []diff.Hunk, v1Lo, v1Hi, fallbackLo, fallbackHi int) (int, int) {
	v2Lo, okLo := mapV1ToV2Lo(hunks, v1Lo)
	v2Hi, okHi := mapV1ToV2Hi(hunks, v1Hi)
	if !okLo {
		v2Lo = fallbackLo
	}
	if !okHi {
		v2Hi = fallbackHi
	}
	if v2Hi < v2Lo {
		v2Hi = v2Lo
	}
	return v2Lo, v2Hi
}

func mapV1ToV2Lo(hunks []diff.Hunk, v1Idx int) (int, bool) {
	for _, h := range hunks {
		if v1Idx < h.V1Start {
			return h.V2Start, true
		}
		if v1Idx < h.V1End {
			switch h.Kind {
			case diff.HunkEqual:
				return h.V2Start + (v1Idx - h.V1Start), true
			default:
				return h.V2Start, true
			}
		}
	}
	if len(hunks) > 0 {
		return hunks[len(hunks)-1].V2End, true
	}
	return 0, false
}

func mapV1ToV2Hi(hunks []diff.Hunk, v1Idx int) (int, bool) {
	for _, h := range hunks {
		if v1Idx <= h.V1Start {
			return h.V2Start, true
		}
		if v1Idx <= h.V1End {
			switch h.Kind {
			case diff.HunkEqual:
				return h.V2Start + (v1Idx - h.V1Start), true
			default:
				return h.V2End, true
			}
		}
	}
	if len(hunks) > 0 {
		return hunks[len(hunks)-1].V2End, true
	}
	return 0, false
}

func classify(v1Lo, v1Hi, v2Lo, v2Hi int) RegionKind {
	v1Empty := v1Lo == v1Hi
	v2Empty := v2Lo == v2Hi
	switch {
	case v1Empty && !v2Empty:
		return RegionInsert
	case v2Empty && !v1Empty:
		return RegionDelete
	default:
		return RegionReplace
	}
}

// computeV1Pages returns the v1 page range this region replaces. For an
// Insert region (empty v1 span) we anchor at the preceding v1 block's end
// page so the margin note reads "Insert after p. N". The Tier-4 sentinel
// fallback is implicit in the bounds: if the region runs off either end of
// v1, we clamp to page 1 / lastPage respectively.
func computeV1Pages(v1Blocks []diff.Block, v1Lo, v1Hi int, v1Pages pagemap.Map, lastPage int) (int, int) {
	if v1Hi > v1Lo {
		// Region has v1 content — use those blocks' span.
		first, firstOK := lookupBlockSpan(v1Blocks[v1Lo], v1Pages)
		last, lastOK := lookupBlockSpan(v1Blocks[v1Hi-1], v1Pages)
		switch {
		case firstOK && lastOK:
			return first.Start, last.End
		case firstOK:
			return first.Start, lastPage
		case lastOK:
			return 1, last.End
		default:
			return 1, lastPage
		}
	}

	// Empty v1 span — Insert. Anchor at the preceding v1 block's end page
	// (or page 1 if at the start of the document; Tier-4 start sentinel).
	if v1Lo == 0 {
		return 1, 1
	}
	prev, ok := lookupBlockSpan(v1Blocks[v1Lo-1], v1Pages)
	if !ok {
		return lastPage, lastPage
	}
	return prev.End, prev.End
}

func lookupBlockSpan(b diff.Block, m pagemap.Map) (pagemap.Span, bool) {
	if key := blockKey(b); key != nil {
		if s, ok := m.Lookup(key); ok {
			return s, true
		}
	}
	return pagemap.Span{}, false
}

func blockKey(b diff.Block) any {
	if b.Node != nil {
		return b.Node
	}
	if b.Line != nil {
		return b.Line
	}
	return nil
}

func nodesFrom(blocks []diff.Block) []ast.Node {
	var nodes []ast.Node
	for _, b := range blocks {
		switch {
		case b.Node != nil:
			// Section-header blocks share their Section pointer with the
			// following child blocks; only emit the top-level node once,
			// when we first see it via a SectionHeader.
			if b.Kind == diff.BlockSectionHeader {
				nodes = append(nodes, b.Node)
			} else {
				// All other block kinds are leaf-ish at the renderer level
				// and emit their own node.
				if !isContainedBySection(b, nodes) {
					nodes = append(nodes, b.Node)
				}
			}
		case b.Line != nil:
			// Section lines are emitted by their parent Section; the
			// renderer walks Section.OrderedItems(). They are not
			// independently re-added here.
		}
	}
	return nodes
}

// isContainedBySection reports whether b's node is inside a Section already
// in nodes — in which case the renderer will reach it naturally and we
// should not double-emit.
func isContainedBySection(b diff.Block, nodes []ast.Node) bool {
	if b.Node == nil {
		return false
	}
	for _, n := range nodes {
		s, ok := n.(*ast.Section)
		if !ok {
			continue
		}
		if sectionContains(s, b.Node) {
			return true
		}
	}
	return false
}

func sectionContains(s *ast.Section, target ast.Node) bool {
	for _, c := range s.Children {
		if c == target {
			return true
		}
		if cs, ok := c.(*ast.Section); ok && sectionContains(cs, target) {
			return true
		}
	}
	return false
}

func contextHeading(v2Slice []diff.Block) string {
	// Pick the deepest enclosing-section path in the slice so the heading
	// describes the most specific location of the change (e.g. "Play —
	// ACT II — SCENE 3"), not just the outermost section that happens to
	// be included via region expansion.
	var deepest []string
	for _, b := range v2Slice {
		if len(b.SectionPath) > len(deepest) {
			deepest = b.SectionPath
		}
	}
	parts := make([]string, 0, len(deepest))
	for _, p := range deepest {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, " — ")
}

func marginNote(kind RegionKind, first, last, _ int) string {
	switch kind {
	case RegionInsert:
		if first <= 1 {
			return "Insert before p. 1"
		}
		return fmt.Sprintf("Insert after p. %d", first)
	case RegionDelete:
		if first == last {
			return fmt.Sprintf("Remove p. %d of v1", first)
		}
		return fmt.Sprintf("Remove pp. %d–%d of v1", first, last)
	case RegionReplace:
		if first == last {
			return fmt.Sprintf("Replace p. %d of v1", first)
		}
		return fmt.Sprintf("Replace pp. %d–%d of v1", first, last)
	}
	return ""
}
