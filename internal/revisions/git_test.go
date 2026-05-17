package revisions

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitInit prepares a fresh repo in dir with a single committed file.
func gitInit(t *testing.T, dir, name, content string) {
	t.Helper()
	mustRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	mustRun("init", "-q")
	mustRun("config", "user.email", "t@example.test")
	mustRun("config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	mustRun("add", name)
	mustRun("commit", "-q", "-m", "v1")
}

func TestReadFromGit_ReadsCommittedContent(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir, "play.ds", "# V1\n\nHAMLET\nLine.\n")

	path := filepath.Join(dir, "play.ds")
	// Update the file on disk to a v2 (uncommitted).
	require.NoError(t, os.WriteFile(path, []byte("# V2\n\nHAMLET\nLine 2.\n"), 0o644))

	got, err := ReadFromGit(path, "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "# V1\n\nHAMLET\nLine.\n", string(got))
}

func TestReadFromGit_ResolvesShortRefs(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir, "play.ds", "# V1\n")
	path := filepath.Join(dir, "play.ds")

	got, err := ReadFromGit(path, "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "# V1\n", string(got))
}

func TestReadFromGit_MissingRefFails(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir, "play.ds", "# V1\n")
	path := filepath.Join(dir, "play.ds")

	_, err := ReadFromGit(path, "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

func TestReadFromGit_FileOutsideRepoFails(t *testing.T) {
	dir := t.TempDir() // not a git repo
	path := filepath.Join(dir, "play.ds")
	require.NoError(t, os.WriteFile(path, []byte("# unmanaged\n"), 0o644))

	_, err := ReadFromGit(path, "HEAD")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not inside a git repository")
}

func TestReadFromGit_FileMissingAtRefFails(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir, "play.ds", "# V1\n")
	missing := filepath.Join(dir, "absent.ds")
	require.NoError(t, os.WriteFile(missing, []byte("# new\n"), 0o644))

	_, err := ReadFromGit(missing, "HEAD")
	require.Error(t, err)
}
