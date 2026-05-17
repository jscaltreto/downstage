// Package pagemap records the [startPage, endPage] span for every block node
// the PDF renderer emits. Callers enable recording by attaching a Recorder to
// render.Config.RecordPageMap; the renderer then calls Begin/End at the start
// and end of each block. Because fpdf's auto-page-break can fire from any
// flowing Write/MultiCell/Ln inside a block, the End call captures the
// authoritative end page regardless of where the break originated.
package pagemap

import "sync"

// Span describes the page range a single block occupies, inclusive on both
// ends. A single-page block has Start == End.
type Span struct {
	Start int
	End   int
}

// Recorder accumulates spans during a render pass. It is safe for use by a
// single renderer goroutine.
type Recorder struct {
	mu    sync.Mutex
	spans map[any]Span
}

// NewRecorder returns a fresh recorder ready to accept Begin/End calls.
func NewRecorder() *Recorder {
	return &Recorder{spans: map[any]Span{}}
}

// Begin records the starting page for a block keyed by its AST node pointer.
// Calling Begin twice for the same key overwrites the prior Start (and resets
// End to the same value).
func (r *Recorder) Begin(key any, page int) {
	if r == nil || key == nil {
		return
	}
	r.mu.Lock()
	r.spans[key] = Span{Start: page, End: page}
	r.mu.Unlock()
}

// End updates the ending page for a block. End may be greater than Start when
// the block's content overflows page boundaries (multi-page dialogue, long
// prose, overflowing verse).
func (r *Recorder) End(key any, page int) {
	if r == nil || key == nil {
		return
	}
	r.mu.Lock()
	if cur, ok := r.spans[key]; ok {
		cur.End = page
		r.spans[key] = cur
	} else {
		r.spans[key] = Span{Start: page, End: page}
	}
	r.mu.Unlock()
}

// Record sets the span for a leaf block (PageBreak, Comment) that has no
// separate Begin/End semantics.
func (r *Recorder) Record(key any, page int) {
	if r == nil || key == nil {
		return
	}
	r.mu.Lock()
	r.spans[key] = Span{Start: page, End: page}
	r.mu.Unlock()
}

// Map returns a snapshot of the recorded spans keyed by AST node pointer.
// The returned map is a copy; callers may mutate it freely.
func (r *Recorder) Map() Map {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(Map, len(r.spans))
	for k, v := range r.spans {
		out[k] = v
	}
	return out
}

// Map is a snapshot of recorded spans. Keys are AST node pointers (e.g.
// *ast.Dialogue, *ast.Section, *ast.SectionLine).
type Map map[any]Span

// Lookup returns the span for a key, or a zero Span and false when absent.
func (m Map) Lookup(key any) (Span, bool) {
	s, ok := m[key]
	return s, ok
}

// LastPage returns the highest End page across all recorded spans. Returns 0
// when the map is empty.
func (m Map) LastPage() int {
	last := 0
	for _, s := range m {
		if s.End > last {
			last = s.End
		}
	}
	return last
}
