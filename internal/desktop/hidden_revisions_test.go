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

func TestHiddenRevisions_RoundTrip(t *testing.T) {
	a := testApp(t)

	require.NoError(t, a.HideRevision("abc1234567abcdef0000000000000000000000aa"))
	require.NoError(t, a.HideRevision("def4567890abcdef0000000000000000000000bb"))

	got := a.GetHiddenRevisions()
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

	assert.Equal(t, []string{hash}, a.GetHiddenRevisions())
}

func TestHiddenRevisions_UnhideAbsent(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.UnhideRevision("abc1234567abcdef0000000000000000000000aa"))
	assert.Empty(t, a.GetHiddenRevisions())
}

func TestHiddenRevisions_UnhideRemoves(t *testing.T) {
	a := testApp(t)
	keep := "abc1234567abcdef0000000000000000000000aa"
	drop := "def4567890abcdef0000000000000000000000bb"

	require.NoError(t, a.HideRevision(keep))
	require.NoError(t, a.HideRevision(drop))
	require.NoError(t, a.UnhideRevision(drop))

	assert.Equal(t, []string{keep}, a.GetHiddenRevisions())
}

func TestHiddenRevisions_MissingFile(t *testing.T) {
	a := testApp(t)
	got := a.GetHiddenRevisions()
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

	got := a.GetHiddenRevisions()
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
