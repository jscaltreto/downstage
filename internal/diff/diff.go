package diff

// HunkKind describes the relationship between an aligned pair of block runs.
type HunkKind int

const (
	HunkEqual HunkKind = iota
	HunkInsert
	HunkDelete
	HunkModify
)

// Hunk is a contiguous alignment between v1 and v2 block streams.
//
//   - HunkEqual:  V1Start..V1End == V2Start..V2End (semantically equal)
//   - HunkInsert: V1Start == V1End (no v1 blocks), V2 blocks added
//   - HunkDelete: V2Start == V2End (no v2 blocks), V1 blocks removed
//   - HunkModify: both spans non-empty and not equal
//
// All indices are half-open: [Start, End).
type Hunk struct {
	Kind    HunkKind
	V1Start int
	V1End   int
	V2Start int
	V2End   int
}

// Diff aligns two flattened block streams and returns the resulting hunks in
// document order. The result always covers every block in both streams; the
// concatenation of v1 indices equals [0, len(v1)) and likewise for v2.
//
// The algorithm is a Patience-style diff over block fingerprints, falling
// back to a Myers LCS within sub-spans where Patience finds no anchors
// (e.g. long stretches of repeated dialogue).
func Diff(v1, v2 []Block) []Hunk {
	var hunks []Hunk
	emitDiff(v1, v2, 0, 0, &hunks)
	return mergeAdjacent(hunks)
}

type anchorPair struct {
	v1, v2 int
}

type diffStep struct {
	op   byte // 'E' equal, 'D' delete (v1), 'I' insert (v2)
	i, j int
}

// emitDiff recursively diffs v1[0:] vs v2[0:] (the slices are already shifted)
// and appends hunks to out, using the supplied (v1Base, v2Base) offsets so
// emitted indices are in the caller's coordinate system.
func emitDiff(v1, v2 []Block, v1Base, v2Base int, out *[]Hunk) {
	// Strip a common prefix.
	prefix := 0
	for prefix < len(v1) && prefix < len(v2) && v1[prefix].Fingerprint == v2[prefix].Fingerprint {
		prefix++
	}
	if prefix > 0 {
		appendHunk(out, Hunk{
			Kind:    HunkEqual,
			V1Start: v1Base, V1End: v1Base + prefix,
			V2Start: v2Base, V2End: v2Base + prefix,
		})
		emitDiff(v1[prefix:], v2[prefix:], v1Base+prefix, v2Base+prefix, out)
		return
	}

	// Strip a common suffix.
	suffix := 0
	for suffix < len(v1) && suffix < len(v2) &&
		v1[len(v1)-1-suffix].Fingerprint == v2[len(v2)-1-suffix].Fingerprint {
		suffix++
	}
	if suffix > 0 {
		mid1 := v1[:len(v1)-suffix]
		mid2 := v2[:len(v2)-suffix]
		emitDiff(mid1, mid2, v1Base, v2Base, out)
		appendHunk(out, Hunk{
			Kind:    HunkEqual,
			V1Start: v1Base + len(mid1), V1End: v1Base + len(v1),
			V2Start: v2Base + len(mid2), V2End: v2Base + len(v2),
		})
		return
	}

	if len(v1) == 0 && len(v2) == 0 {
		return
	}
	if len(v1) == 0 {
		appendHunk(out, Hunk{Kind: HunkInsert, V1Start: v1Base, V1End: v1Base, V2Start: v2Base, V2End: v2Base + len(v2)})
		return
	}
	if len(v2) == 0 {
		appendHunk(out, Hunk{Kind: HunkDelete, V1Start: v1Base, V1End: v1Base + len(v1), V2Start: v2Base, V2End: v2Base})
		return
	}

	// Try Patience anchors over this slice.
	if anchors := patienceAnchors(v1, v2); len(anchors) > 0 {
		prevV1, prevV2 := 0, 0
		for _, a := range anchors {
			emitDiff(v1[prevV1:a.v1], v2[prevV2:a.v2], v1Base+prevV1, v2Base+prevV2, out)
			appendHunk(out, Hunk{
				Kind:    HunkEqual,
				V1Start: v1Base + a.v1, V1End: v1Base + a.v1 + 1,
				V2Start: v2Base + a.v2, V2End: v2Base + a.v2 + 1,
			})
			prevV1, prevV2 = a.v1+1, a.v2+1
		}
		emitDiff(v1[prevV1:], v2[prevV2:], v1Base+prevV1, v2Base+prevV2, out)
		return
	}

	// No Patience anchors — fall back to Myers LCS.
	myersDiff(v1, v2, v1Base, v2Base, out)
}

// patienceAnchors returns a longest-increasing-subsequence of fingerprints
// that appear exactly once in each of v1 and v2, in v1 order.
func patienceAnchors(v1, v2 []Block) []anchorPair {
	v1Counts := map[string]int{}
	v1Index := map[string]int{}
	for i, b := range v1 {
		v1Counts[b.Fingerprint]++
		v1Index[b.Fingerprint] = i
	}
	v2Counts := map[string]int{}
	v2Index := map[string]int{}
	for i, b := range v2 {
		v2Counts[b.Fingerprint]++
		v2Index[b.Fingerprint] = i
	}

	var pairs []anchorPair
	for fp, c1 := range v1Counts {
		if c1 != 1 {
			continue
		}
		if v2Counts[fp] != 1 {
			continue
		}
		pairs = append(pairs, anchorPair{v1: v1Index[fp], v2: v2Index[fp]})
	}
	if len(pairs) == 0 {
		return nil
	}

	// Sort by v1 index ascending (insertion sort; pairs is small in practice).
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j-1].v1 > pairs[j].v1; j-- {
			pairs[j-1], pairs[j] = pairs[j], pairs[j-1]
		}
	}

	// LIS on v2 indices.
	return longestIncreasingSubsequence(pairs)
}

// longestIncreasingSubsequence returns the LIS of pairs by .v2, preserving
// the v1 ordering that the input is assumed to already have.
func longestIncreasingSubsequence(pairs []anchorPair) []anchorPair {
	if len(pairs) == 0 {
		return nil
	}
	n := len(pairs)
	tails := make([]int, 0, n)
	prev := make([]int, n)
	for i := range prev {
		prev[i] = -1
	}
	for i, p := range pairs {
		lo, hi := 0, len(tails)
		for lo < hi {
			mid := (lo + hi) / 2
			if pairs[tails[mid]].v2 < p.v2 {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo > 0 {
			prev[i] = tails[lo-1]
		}
		if lo == len(tails) {
			tails = append(tails, i)
		} else {
			tails[lo] = i
		}
	}

	k := tails[len(tails)-1]
	out := make([]anchorPair, 0, len(tails))
	for k != -1 {
		out = append(out, pairs[k])
		k = prev[k]
	}
	// Reverse.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// myersDiff implements a standard LCS-based diff for sub-spans where Patience
// found no anchors. The textbook DP is acceptable here because Patience
// catches the long-script case; only short, anchor-free sub-spans hit this
// fallback.
func myersDiff(v1, v2 []Block, v1Base, v2Base int, out *[]Hunk) {
	n, m := len(v1), len(v2)
	if n == 0 && m == 0 {
		return
	}

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if v1[i-1].Fingerprint == v2[j-1].Fingerprint {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var steps []diffStep
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && v1[i-1].Fingerprint == v2[j-1].Fingerprint:
			steps = append(steps, diffStep{op: 'E', i: i - 1, j: j - 1})
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			steps = append(steps, diffStep{op: 'I', i: i, j: j - 1})
			j--
		default:
			steps = append(steps, diffStep{op: 'D', i: i - 1, j: j})
			i--
		}
	}
	for k, l := 0, len(steps)-1; k < l; k, l = k+1, l-1 {
		steps[k], steps[l] = steps[l], steps[k]
	}

	*out = append(*out, buildHunksFromSteps(steps, v1Base, v2Base)...)
}

type stepSpan struct {
	i1, j1 int // inclusive lo (i for v1, j for v2)
	i2, j2 int // exclusive hi
}

// buildHunksFromSteps walks a forward-ordered step sequence (E/D/I) and
// produces the merged hunk list, coalescing adjacent D+I into HunkModify.
func buildHunksFromSteps(steps []diffStep, v1Base, v2Base int) []Hunk {
	var hunks []Hunk
	var (
		eq, del, ins          stepSpan
		hasEq, hasDel, hasIns bool
	)
	reset := func() { hasEq, hasDel, hasIns = false, false, false }
	flush := func() {
		if hasDel && hasIns {
			hunks = append(hunks, Hunk{
				Kind:    HunkModify,
				V1Start: v1Base + del.i1, V1End: v1Base + del.i2,
				V2Start: v2Base + ins.j1, V2End: v2Base + ins.j2,
			})
		} else if hasDel {
			hunks = append(hunks, Hunk{
				Kind:    HunkDelete,
				V1Start: v1Base + del.i1, V1End: v1Base + del.i2,
				V2Start: v2Base + del.j1, V2End: v2Base + del.j1,
			})
		} else if hasIns {
			hunks = append(hunks, Hunk{
				Kind:    HunkInsert,
				V1Start: v1Base + ins.i1, V1End: v1Base + ins.i1,
				V2Start: v2Base + ins.j1, V2End: v2Base + ins.j2,
			})
		}
		if hasEq {
			hunks = append(hunks, Hunk{
				Kind:    HunkEqual,
				V1Start: v1Base + eq.i1, V1End: v1Base + eq.i2,
				V2Start: v2Base + eq.j1, V2End: v2Base + eq.j2,
			})
		}
		reset()
	}

	for _, s := range steps {
		switch s.op {
		case 'E':
			if hasDel || hasIns {
				flush()
			}
			if !hasEq {
				eq = stepSpan{i1: s.i, j1: s.j, i2: s.i + 1, j2: s.j + 1}
				hasEq = true
			} else {
				eq.i2 = s.i + 1
				eq.j2 = s.j + 1
			}
		case 'D':
			if hasEq {
				flush()
			}
			if !hasDel {
				del = stepSpan{i1: s.i, j1: s.j, i2: s.i + 1, j2: s.j}
				hasDel = true
			} else {
				del.i2 = s.i + 1
			}
		case 'I':
			if hasEq {
				flush()
			}
			if !hasIns {
				ins = stepSpan{i1: s.i, j1: s.j, i2: s.i, j2: s.j + 1}
				hasIns = true
			} else {
				ins.j2 = s.j + 1
			}
		}
	}
	flush()
	return hunks
}

// mergeAdjacent collapses consecutive same-kind hunks that abut perfectly,
// which can arise from recursive emitDiff calls split at Patience anchors.
func mergeAdjacent(hunks []Hunk) []Hunk {
	if len(hunks) <= 1 {
		return hunks
	}
	out := hunks[:0]
	out = append(out, hunks[0])
	for _, h := range hunks[1:] {
		last := &out[len(out)-1]
		if last.Kind == h.Kind && last.V1End == h.V1Start && last.V2End == h.V2Start {
			last.V1End = h.V1End
			last.V2End = h.V2End
			continue
		}
		out = append(out, h)
	}
	return out
}

func appendHunk(out *[]Hunk, h Hunk) {
	if h.V1Start == h.V1End && h.V2Start == h.V2End {
		return
	}
	*out = append(*out, h)
}
