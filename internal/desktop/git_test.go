package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isolateGitIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestGetFileGitStatus_CleanTrackedFile(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	require.NoError(t, a.WriteLibraryFile("play.ds", "content"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.False(t, st.Dirty, "tracked file matching HEAD is not dirty")
	assert.True(t, st.HasHead)
	assert.NotEmpty(t, st.HeadAt)
	assert.False(t, st.Untracked)
	assert.False(t, st.Missing)
}

func TestGetFileGitStatus_DirtyTrackedFile(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v2"))

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.True(t, st.Dirty)
	assert.True(t, st.HasHead)
	assert.False(t, st.Untracked)
	assert.False(t, st.Missing)
}

func TestGetFileGitStatus_StagedChange(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v2"))

	r, err := git.PlainOpen(a.currentLibrary)
	require.NoError(t, err)
	w, err := r.Worktree()
	require.NoError(t, err)
	_, err = w.Add("play.ds")
	require.NoError(t, err)

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.True(t, st.Dirty)
	assert.True(t, st.HasHead)
}

func TestGetFileGitStatus_NeverCommittedFile(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	_, err := git.PlainInit(a.currentLibrary, false)
	require.NoError(t, err)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.True(t, st.Dirty, "never-committed file is dirty")
	assert.False(t, st.HasHead)
	assert.Empty(t, st.HeadAt)
	assert.True(t, st.Untracked)
	assert.False(t, st.Missing)
}

func TestGetFileGitStatus_DeletedOnDisk(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	require.NoError(t, os.Remove(filepath.Join(a.currentLibrary, "play.ds")))

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.True(t, st.Missing)
	assert.False(t, st.Dirty, "missing file should not be reported as dirty")
	assert.True(t, st.HasHead)
	assert.NotEmpty(t, st.HeadAt)
}

func TestGetFileGitStatus_NoRepo(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("v1"), 0644))

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.True(t, st.Dirty)
	assert.False(t, st.HasHead)
	assert.True(t, st.Untracked)
	assert.False(t, st.Missing)
}

func TestGetFileGitStatus_NoRepoMissingFile(t *testing.T) {
	a := testApp(t)

	st, err := a.GetFileGitStatus("does-not-exist.ds")
	require.NoError(t, err)
	assert.True(t, st.Missing)
	assert.False(t, st.Dirty)
	assert.False(t, st.HasHead)
}

func TestGetFileGitStatus_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	_, err := a.GetFileGitStatus("../../etc/passwd")
	assert.Error(t, err)
}

func TestGetFileGitStatus_NoLibrary(t *testing.T) {
	a := &App{}

	_, err := a.GetFileGitStatus("play.ds")
	assert.Error(t, err)
}

// --- GetLibraryDirty / CommitPaths / DiscardPaths ---

func TestGetLibraryDirty_CategorizesAndDetectsKinds(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	// Seed: tracked play, then create dirty state of every kind.
	require.NoError(t, a.WriteLibraryFile("act-one.ds", "v1"))
	require.NoError(t, a.WriteLibraryFile("act-two.ds", "v1"))
	require.NoError(t, a.SnapshotFile("act-one.ds", "initial one"))
	require.NoError(t, a.SnapshotFile("act-two.ds", "initial two"))

	// Modify act-one (modified), delete act-two (deleted), add new untracked.
	require.NoError(t, a.WriteLibraryFile("act-one.ds", "v2"))
	require.NoError(t, os.Remove(filepath.Join(a.currentLibrary, "act-two.ds")))
	require.NoError(t, a.WriteLibraryFile("act-three.ds", "fresh"))

	// Sidecar (the dictionary file): write directly via the library API,
	// which leaves it uncommitted.
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, ".downstage"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, ".downstage/dictionary.txt"), []byte("word"), 0644))

	// "Other" — non-.ds file outside .downstage.
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "notes.md"), []byte("hi"), 0644))

	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)

	kindByPath := func(paths []DirtyPath) map[string]DirtyKind {
		m := map[string]DirtyKind{}
		for _, p := range paths {
			m[p.Path] = p.Kind
		}
		return m
	}
	plays := kindByPath(dirty.Plays)
	assert.Equal(t, DirtyModified, plays["act-one.ds"], "act-one is modified")
	assert.Equal(t, DirtyDeleted, plays["act-two.ds"], "act-two is deleted")
	assert.Equal(t, DirtyUntracked, plays["act-three.ds"], "act-three is untracked")

	sidecars := kindByPath(dirty.Sidecars)
	assert.Equal(t, DirtyUntracked, sidecars[".downstage/dictionary.txt"], "sidecar surfaces in its own bucket")

	other := kindByPath(dirty.Other)
	assert.Equal(t, DirtyUntracked, other["notes.md"], "notes.md falls into Other")

	assert.Equal(t, len(dirty.Plays)+len(dirty.Sidecars)+len(dirty.Other), dirty.Count)
}

func TestGetLibraryDirty_CleanRepoIsEmpty(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)
	assert.Equal(t, 0, dirty.Count)
}

func TestGetLibraryDirty_NoRepoIsEmpty(t *testing.T) {
	a := testApp(t)
	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)
	assert.Equal(t, 0, dirty.Count, "no repo means no dirty surface yet")
}

func TestCommitPaths_BatchesAddAndRemoveIntoOneCommit(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("a.ds", "v1"))
	require.NoError(t, a.WriteLibraryFile("b.ds", "v1"))
	require.NoError(t, a.SnapshotFile("a.ds", "seed a"))
	require.NoError(t, a.SnapshotFile("b.ds", "seed b"))

	// Modify a.ds, delete b.ds, add new c.ds.
	require.NoError(t, a.WriteLibraryFile("a.ds", "v2"))
	require.NoError(t, os.Remove(filepath.Join(a.currentLibrary, "b.ds")))
	require.NoError(t, a.WriteLibraryFile("c.ds", "fresh"))

	require.NoError(t, a.CommitPaths([]string{"a.ds", "b.ds", "c.ds"}, "Batch cleanup"))

	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)
	assert.Equal(t, 0, dirty.Count, "all three paths should be clean after CommitPaths")

	// A single new commit should be on HEAD with our message.
	r, err := git.PlainOpen(a.currentLibrary)
	require.NoError(t, err)
	ref, err := r.Head()
	require.NoError(t, err)
	c, err := r.CommitObject(ref.Hash())
	require.NoError(t, err)
	assert.Equal(t, "Batch cleanup", c.Message)
}

func TestCommitPaths_NothingToCommitReturnsSentinel(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "seed"))

	err := a.CommitPaths([]string{"play.ds"}, "noop")
	assert.ErrorIs(t, err, ErrNothingToSnapshot)
}

func TestCommitPaths_RejectsEmptyInput(t *testing.T) {
	a := testApp(t)
	assert.Error(t, a.CommitPaths(nil, "msg"))
	assert.Error(t, a.CommitPaths([]string{"play.ds"}, ""))
}

func TestDiscardPaths_RestoresModifiedAndDeleted_RemovesUntracked(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("mod.ds", "v1"))
	require.NoError(t, a.WriteLibraryFile("gone.ds", "v1"))
	require.NoError(t, a.SnapshotFile("mod.ds", "seed mod"))
	require.NoError(t, a.SnapshotFile("gone.ds", "seed gone"))

	// Modify mod, delete gone, add new untracked.
	require.NoError(t, a.WriteLibraryFile("mod.ds", "DIRTY"))
	require.NoError(t, os.Remove(filepath.Join(a.currentLibrary, "gone.ds")))
	require.NoError(t, a.WriteLibraryFile("new.ds", "throwaway"))

	require.NoError(t, a.DiscardPaths([]string{"mod.ds", "gone.ds", "new.ds"}))

	// mod.ds should now match HEAD ("v1"), gone.ds should be back, new.ds removed.
	modContent, err := a.ReadLibraryFile("mod.ds")
	require.NoError(t, err)
	assert.Equal(t, "v1", modContent)

	goneContent, err := a.ReadLibraryFile("gone.ds")
	require.NoError(t, err)
	assert.Equal(t, "v1", goneContent)

	_, err = os.Stat(filepath.Join(a.currentLibrary, "new.ds"))
	assert.True(t, os.IsNotExist(err), "untracked new.ds should be removed from disk")

	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)
	assert.Equal(t, 0, dirty.Count, "worktree should be clean after discard")
}

// --- DeleteLibraryFile / RestoreLibraryFile ---

func TestDeleteLibraryFile_RemovesAndCommits(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("doomed.ds", "bye"))
	require.NoError(t, a.SnapshotFile("doomed.ds", "seed"))

	require.NoError(t, a.DeleteLibraryFile("doomed.ds"))

	_, err := os.Stat(filepath.Join(a.currentLibrary, "doomed.ds"))
	assert.True(t, os.IsNotExist(err), "file removed from disk")

	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)
	assert.Equal(t, 0, dirty.Count, "deletion was committed; no dirty state remains")

	r, err := git.PlainOpen(a.currentLibrary)
	require.NoError(t, err)
	ref, err := r.Head()
	require.NoError(t, err)
	c, err := r.CommitObject(ref.Hash())
	require.NoError(t, err)
	assert.Contains(t, c.Message, "Delete doomed.ds")
}

func TestDeleteLibraryFile_UntrackedFileRemovedWithoutCommit(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	// Init repo with an unrelated file so HEAD exists.
	require.NoError(t, a.WriteLibraryFile("anchor.ds", "x"))
	require.NoError(t, a.SnapshotFile("anchor.ds", "seed"))

	require.NoError(t, a.WriteLibraryFile("scratch.ds", "throwaway"))
	require.NoError(t, a.DeleteLibraryFile("scratch.ds"))

	_, err := os.Stat(filepath.Join(a.currentLibrary, "scratch.ds"))
	assert.True(t, os.IsNotExist(err))

	// No new commit beyond the seed.
	r, err := git.PlainOpen(a.currentLibrary)
	require.NoError(t, err)
	ref, err := r.Head()
	require.NoError(t, err)
	c, err := r.CommitObject(ref.Hash())
	require.NoError(t, err)
	assert.Equal(t, "seed", c.Message, "untracked deletion should not create a commit")
}

func TestDeleteLibraryFile_RejectsDirectoryAndNonDs(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "notes.md"), []byte("hi"), 0644))

	assert.Error(t, a.DeleteLibraryFile("subdir"), "directories rejected")
	assert.Error(t, a.DeleteLibraryFile("notes.md"), "non-.ds files rejected")
}

func TestRestoreLibraryFile_ReturnsDeletedFileFromHEAD(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("revive.ds", "saved content"))
	require.NoError(t, a.SnapshotFile("revive.ds", "seed"))

	// Out-of-band delete.
	require.NoError(t, os.Remove(filepath.Join(a.currentLibrary, "revive.ds")))

	require.NoError(t, a.RestoreLibraryFile("revive.ds"))

	content, err := a.ReadLibraryFile("revive.ds")
	require.NoError(t, err)
	assert.Equal(t, "saved content", content)

	dirty, err := a.GetLibraryDirty()
	require.NoError(t, err)
	assert.Equal(t, 0, dirty.Count, "restore should leave the worktree clean")
}

func TestRestoreLibraryFile_RefusesWhenFileIsModifiedNotDeleted(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "seed"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v2-LOCAL"))

	err := a.RestoreLibraryFile("play.ds")
	assert.Error(t, err, "modified files should not be silently overwritten via Restore")
}

func TestRestoreLibraryFile_RefusesUnknownPath(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("seed.ds", "x"))
	require.NoError(t, a.SnapshotFile("seed.ds", "anchor"))

	err := a.RestoreLibraryFile("never-existed.ds")
	assert.Error(t, err)
}
