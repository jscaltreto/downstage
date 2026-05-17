package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustGetHidden is a tiny helper so individual test bodies don't have
// to deal with the (slice, error) tuple from GetHiddenRevisions every
// time. Read failures cause test failures, which is the desired
// behavior — none of the happy-path tests should ever trip a read
// error.
func mustGetHidden(t *testing.T, a *App) []string {
	t.Helper()
	got, err := a.GetHiddenRevisions()
	require.NoError(t, err)
	return got
}

func TestHiddenRevisions_RoundTrip(t *testing.T) {
	a := testApp(t)

	require.NoError(t, a.HideRevision("abc1234567abcdef0000000000000000000000aa"))
	require.NoError(t, a.HideRevision("def4567890abcdef0000000000000000000000bb"))

	got := mustGetHidden(t, a)
	sort.Strings(got)
	assert.Equal(t, []string{
		"abc1234567abcdef0000000000000000000000aa",
		"def4567890abcdef0000000000000000000000bb",
	}, got)
}

func TestHiddenRevisions_IdempotentAdd(t *testing.T) {
	a := testApp(t)
	hash := "abc1234567abcdef0000000000000000000000aa"

	require.NoError(t, a.HideRevision(hash))
	require.NoError(t, a.HideRevision(hash))
	require.NoError(t, a.HideRevision(hash))

	assert.Equal(t, []string{hash}, mustGetHidden(t, a))
}

func TestHiddenRevisions_UnhideAbsent(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.UnhideRevision("abc1234567abcdef0000000000000000000000aa"))
	assert.Empty(t, mustGetHidden(t, a))
}

func TestHiddenRevisions_UnhideRemoves(t *testing.T) {
	a := testApp(t)
	keep := "abc1234567abcdef0000000000000000000000aa"
	drop := "def4567890abcdef0000000000000000000000bb"

	require.NoError(t, a.HideRevision(keep))
	require.NoError(t, a.HideRevision(drop))
	require.NoError(t, a.UnhideRevision(drop))

	assert.Equal(t, []string{keep}, mustGetHidden(t, a))
}

func TestHiddenRevisions_MissingFile(t *testing.T) {
	a := testApp(t)
	got := mustGetHidden(t, a)
	// Empty slice, never nil — the wire payload to the frontend is
	// always an array.
	require.NotNil(t, got)
	assert.Empty(t, got)
}

func TestHiddenRevisions_SortedDedupedOnDisk(t *testing.T) {
	a := testApp(t)
	// Insert in non-sorted order with duplicates; on-disk file must be
	// sorted + deduped so contents don't churn between runs.
	a.hiddenMu.Lock()
	require.NoError(t, a.writeHiddenRevisions([]string{
		"def4567890abcdef0000000000000000000000bb",
		"abc1234567abcdef0000000000000000000000aa",
		"abc1234567abcdef0000000000000000000000aa",
	}))
	a.hiddenMu.Unlock()

	data, err := os.ReadFile(filepath.Join(a.currentLibrary, hiddenRevisionsPath))
	require.NoError(t, err)
	assert.Equal(t,
		"abc1234567abcdef0000000000000000000000aa\n"+
			"def4567890abcdef0000000000000000000000bb\n",
		string(data),
	)
}

func TestHiddenRevisions_AtomicWrite_NoTmpResidue(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.HideRevision("abc1234567abcdef0000000000000000000000aa"))

	// On a successful write the .tmp file must have been renamed away;
	// any lingering tmp file means the rename half-failed silently.
	tmpPath := filepath.Join(a.currentLibrary, hiddenRevisionsPath+".tmp")
	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "tmp file should not remain after a successful write")
}

func TestHiddenRevisions_NoLibrary(t *testing.T) {
	a := &App{}
	err := a.HideRevision("abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no library open")
}

func TestHiddenRevisions_EmptyHashRejected(t *testing.T) {
	a := testApp(t)
	require.Error(t, a.HideRevision(""))
	require.Error(t, a.HideRevision("   "))
}

func TestHiddenRevisions_RaceFree(t *testing.T) {
	// 50 concurrent HideRevision calls with distinct hashes must all
	// land. Run with `go test -race` to catch lock issues.
	a := testApp(t)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Pad to 40 hex chars so the format is realistic; uniqueness
			// is what matters here.
			hash := fmt.Sprintf("%040x", i)
			require.NoError(t, a.HideRevision(hash))
		}(i)
	}
	wg.Wait()

	got := mustGetHidden(t, a)
	assert.Len(t, got, 50, "all 50 hashes should be present")

	// Confirm the on-disk file is well-formed (no torn writes, no
	// duplicates introduced by interleaved writers).
	data, err := os.ReadFile(filepath.Join(a.currentLibrary, hiddenRevisionsPath))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	assert.Len(t, lines, 50)
	seen := make(map[string]struct{})
	for _, line := range lines {
		seen[line] = struct{}{}
	}
	assert.Len(t, seen, 50, "no duplicate lines after concurrent writes")
}

// makeHiddenFileUnreadable forces a non-ENOENT read failure by
// chmod'ing the file to 0. Linux/macOS only; the file is restored at
// test teardown via t.Cleanup. We skip if running as root because
// chmod doesn't restrict root's read access.
func makeHiddenFileUnreadable(t *testing.T, a *App) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root — chmod 0 won't block reads")
	}
	path := filepath.Join(a.currentLibrary, hiddenRevisionsPath)
	require.NoError(t, os.Chmod(path, 0))
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })
}

// TestHiddenRevisions_HideRefusesOnReadFailure covers the H2
// data-loss scenario: a non-ENOENT read failure (EACCES here) must
// surface as an error from HideRevision, NOT silently produce a
// single-element file that erases the previously hidden hashes.
func TestHiddenRevisions_HideRefusesOnReadFailure(t *testing.T) {
	a := testApp(t)
	// Seed two real hashes.
	preexisting := []string{
		"abc1234567abcdef0000000000000000000000aa",
		"def4567890abcdef0000000000000000000000bb",
	}
	for _, h := range preexisting {
		require.NoError(t, a.HideRevision(h))
	}

	// Capture on-disk bytes before forcing a read failure so we can
	// assert the file is byte-identical afterward.
	path := filepath.Join(a.currentLibrary, hiddenRevisionsPath)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	makeHiddenFileUnreadable(t, a)

	err = a.HideRevision("999000abcd000000000000000000000000000000")
	require.Error(t, err, "HideRevision must propagate the read error")

	// Restore read access and confirm the file still has the original two.
	require.NoError(t, os.Chmod(path, 0644))
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after,
		"on-disk file must be untouched after a read-error-aborted Hide")
}

func TestHiddenRevisions_UnhideRefusesOnReadFailure(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.HideRevision("abc1234567abcdef0000000000000000000000aa"))
	path := filepath.Join(a.currentLibrary, hiddenRevisionsPath)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	makeHiddenFileUnreadable(t, a)

	err = a.UnhideRevision("abc1234567abcdef0000000000000000000000aa")
	require.Error(t, err, "UnhideRevision must propagate the read error")

	require.NoError(t, os.Chmod(path, 0644))
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after,
		"on-disk file must be untouched after a read-error-aborted Unhide")
}

func TestHiddenRevisions_GetReturnsErrorOnReadFailure(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.HideRevision("abc1234567abcdef0000000000000000000000aa"))

	makeHiddenFileUnreadable(t, a)

	_, err := a.GetHiddenRevisions()
	require.Error(t, err,
		"GetHiddenRevisions must surface real read failures instead of returning an empty slice")
}
