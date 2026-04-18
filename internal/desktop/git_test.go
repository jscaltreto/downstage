package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateGitIdentity points HOME at a scratch dir so the developer's
// personal gitconfig doesn't leak into GetFileGitStatus tests — commit
// metadata we assert on should be deterministic per run.
func isolateGitIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestGetFileGitStatus_CleanTrackedFile(t *testing.T) {
	isolateGitIdentity(t)
	a := testApp(t)

	require.NoError(t, a.WriteProjectFile("play.ds", "content"))
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

	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))
	require.NoError(t, a.WriteProjectFile("play.ds", "v2"))

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

	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))
	require.NoError(t, a.WriteProjectFile("play.ds", "v2"))

	// Stage the change without committing.
	r, err := git.PlainOpen(a.currentProject)
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

	// Init a repo so "no repo" isn't what we're testing here.
	_, err := git.PlainInit(a.currentProject, false)
	require.NoError(t, err)
	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))

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

	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	// Delete the working copy so Missing fires; git history still
	// remembers it.
	require.NoError(t, os.Remove(filepath.Join(a.currentProject, "play.ds")))

	st, err := a.GetFileGitStatus("play.ds")
	require.NoError(t, err)
	assert.True(t, st.Missing)
	assert.False(t, st.Dirty, "missing file should not be reported as dirty")
	assert.True(t, st.HasHead)
	assert.NotEmpty(t, st.HeadAt)
}

func TestGetFileGitStatus_NoRepo(t *testing.T) {
	a := testApp(t)
	// No git init. WriteProjectFile does not implicitly init.
	require.NoError(t, os.WriteFile(filepath.Join(a.currentProject, "play.ds"), []byte("v1"), 0644))

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

func TestGetFileGitStatus_NoProject(t *testing.T) {
	a := &App{}

	_, err := a.GetFileGitStatus("play.ds")
	assert.Error(t, err)
}
