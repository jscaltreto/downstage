package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	a := &App{currentProject: dir}
	return a
}

func testAppWithConfig(t *testing.T) *App {
	t.Helper()
	a := testApp(t)
	configDir := t.TempDir()
	a.configPath = filepath.Join(configDir, "config.json")
	return a
}

func TestSafePath_AllowsValidPaths(t *testing.T) {
	a := testApp(t)
	os.WriteFile(filepath.Join(a.currentProject, "play.ds"), []byte("test"), 0644)

	got, err := a.safePath("play.ds")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(a.currentProject, "play.ds"), got)
}

func TestSafePath_AllowsNestedPaths(t *testing.T) {
	a := testApp(t)
	sub := filepath.Join(a.currentProject, "subdir")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "play.ds"), []byte("test"), 0644)

	got, err := a.safePath("subdir/play.ds")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(a.currentProject, "subdir", "play.ds"), got)
}

func TestSafePath_AllowsNewFileInExistingDir(t *testing.T) {
	a := testApp(t)

	got, err := a.safePath("newfile.ds")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(a.currentProject, "newfile.ds"), got)
}

func TestSafePath_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	_, err := a.safePath("../../etc/passwd")
	assert.Error(t, err)
}

func TestSafePath_BlocksTraversalAfterDescent(t *testing.T) {
	a := testApp(t)

	_, err := a.safePath("subdir/../../..")
	assert.Error(t, err)
}

func TestSafePath_BlocksSymlinkEscape(t *testing.T) {
	a := testApp(t)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0644)

	os.Symlink(outside, filepath.Join(a.currentProject, "escape"))

	_, err := a.safePath("escape/secret.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path escapes project root")
}

func TestSafePath_NoProject(t *testing.T) {
	a := &App{}

	_, err := a.safePath("anything.ds")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project open")
}

func TestConfigRoundTrip(t *testing.T) {
	a := testAppWithConfig(t)

	a.saveConfig(Config{LastProjectPath: "/some/path"})
	a.saveConfig(Config{LastActiveProjectFile: "play.ds"})

	data, err := os.ReadFile(a.configPath)
	require.NoError(t, err)

	var config Config
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Equal(t, "/some/path", config.LastProjectPath)
	assert.Equal(t, "play.ds", config.LastActiveProjectFile)
}

func TestCreateProjectFile_Dedup(t *testing.T) {
	a := testApp(t)

	path1, err := a.CreateProjectFile("test", "content1")
	require.NoError(t, err)
	assert.Equal(t, "test.ds", path1)

	path2, err := a.CreateProjectFile("test", "content2")
	require.NoError(t, err)
	assert.Equal(t, "test-1.ds", path2)

	path3, err := a.CreateProjectFile("test", "content3")
	require.NoError(t, err)
	assert.Equal(t, "test-2.ds", path3)
}

func TestCreateProjectFile_AddsSuffix(t *testing.T) {
	a := testApp(t)

	path, err := a.CreateProjectFile("My Play", "content")
	require.NoError(t, err)
	assert.Equal(t, "My Play.ds", path)
}

func TestCreateProjectFile_PreservesSuffix(t *testing.T) {
	a := testApp(t)

	path, err := a.CreateProjectFile("play.ds", "content")
	require.NoError(t, err)
	assert.Equal(t, "play.ds", path)
}

func TestGetProjectFiles_Filters(t *testing.T) {
	a := testApp(t)

	os.WriteFile(filepath.Join(a.currentProject, "play.ds"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(a.currentProject, "readme.md"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(a.currentProject, "notes.txt"), []byte("test"), 0644)

	gitDir := filepath.Join(a.currentProject, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref"), 0644)

	files, err := a.GetProjectFiles()
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "play.ds", files[0].Name)
}

func TestGetProjectFiles_SkipsDownstageDir(t *testing.T) {
	a := testApp(t)

	os.WriteFile(filepath.Join(a.currentProject, "play.ds"), []byte("test"), 0644)
	dsDir := filepath.Join(a.currentProject, ".downstage")
	os.MkdirAll(dsDir, 0755)
	os.WriteFile(filepath.Join(dsDir, "internal.ds"), []byte("test"), 0644)

	files, err := a.GetProjectFiles()
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "play.ds", files[0].Name)
}

func TestWriteProjectFile_NoAutoCommit(t *testing.T) {
	a := testApp(t)
	_, err := git.PlainInit(a.currentProject, false)
	require.NoError(t, err)

	err = a.WriteProjectFile("play.ds", "content")
	require.NoError(t, err)

	r, err := git.PlainOpen(a.currentProject)
	require.NoError(t, err)

	_, err = r.Head()
	assert.Error(t, err, "should have no commits after WriteProjectFile")
}

func TestSnapshotFile(t *testing.T) {
	a := testApp(t)

	err := a.WriteProjectFile("play.ds", "content")
	require.NoError(t, err)

	err = a.SnapshotFile("play.ds", "initial version")
	require.NoError(t, err)

	revisions, err := a.GetRevisions("play.ds")
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "initial version", revisions[0].Message)
}

func TestGetRevisions_Order(t *testing.T) {
	a := testApp(t)

	a.WriteProjectFile("play.ds", "v1")
	a.SnapshotFile("play.ds", "first")

	a.WriteProjectFile("play.ds", "v2")
	a.SnapshotFile("play.ds", "second")

	revisions, err := a.GetRevisions("play.ds")
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	assert.Equal(t, "second", revisions[0].Message)
	assert.Equal(t, "first", revisions[1].Message)
}

func TestReadProjectFile_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	_, err := a.ReadProjectFile("../../etc/passwd")
	assert.Error(t, err)
}

func TestWriteProjectFile_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	err := a.WriteProjectFile("../../tmp/evil.txt", "evil")
	assert.Error(t, err)
}

func TestSnapshotFile_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	err := a.SnapshotFile("../../etc/passwd", "steal")
	assert.Error(t, err)
}

func TestGetSpellAllowlist_Empty(t *testing.T) {
	a := testApp(t)

	words := a.GetSpellAllowlist()
	assert.Empty(t, words)
}

func TestAddSpellAllowlistWord(t *testing.T) {
	a := testApp(t)

	added, err := a.AddSpellAllowlistWord("Nebula")
	require.NoError(t, err)
	assert.True(t, added)

	words := a.GetSpellAllowlist()
	assert.Equal(t, []string{"Nebula"}, words)
}

func TestAddSpellAllowlistWord_Dedup(t *testing.T) {
	a := testApp(t)

	added, _ := a.AddSpellAllowlistWord("Nebula")
	assert.True(t, added)

	added, _ = a.AddSpellAllowlistWord("nebula")
	assert.False(t, added)

	words := a.GetSpellAllowlist()
	assert.Len(t, words, 1)
}

func TestRemoveSpellAllowlistWord(t *testing.T) {
	a := testApp(t)

	a.AddSpellAllowlistWord("Nebula")
	a.AddSpellAllowlistWord("Starfall")

	removed, err := a.RemoveSpellAllowlistWord("nebula")
	require.NoError(t, err)
	assert.True(t, removed)

	words := a.GetSpellAllowlist()
	assert.Equal(t, []string{"Starfall"}, words)
}

func TestRemoveSpellAllowlistWord_NotFound(t *testing.T) {
	a := testApp(t)

	removed, err := a.RemoveSpellAllowlistWord("Nebula")
	require.NoError(t, err)
	assert.False(t, removed)
}
