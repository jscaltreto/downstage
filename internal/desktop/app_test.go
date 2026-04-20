package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	a := &App{currentLibrary: dir}
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
	os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("test"), 0644)

	got, err := a.safePath("play.ds")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(a.currentLibrary, "play.ds"), got)
}

func TestSafePath_AllowsNestedPaths(t *testing.T) {
	a := testApp(t)
	sub := filepath.Join(a.currentLibrary, "subdir")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "play.ds"), []byte("test"), 0644)

	got, err := a.safePath("subdir/play.ds")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(a.currentLibrary, "subdir", "play.ds"), got)
}

func TestSafePath_AllowsNewFileInExistingDir(t *testing.T) {
	a := testApp(t)

	got, err := a.safePath("newfile.ds")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(a.currentLibrary, "newfile.ds"), got)
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

	os.Symlink(outside, filepath.Join(a.currentLibrary, "escape"))

	_, err := a.safePath("escape/secret.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path escapes library root")
}

func TestSafePath_NoLibrary(t *testing.T) {
	a := &App{}

	_, err := a.safePath("anything.ds")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no library open")
}

func TestSafePath_BlocksAbsoluteInput(t *testing.T) {
	a := testApp(t)

	_, err := a.safePath("/etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute")
}

// Regression: a dangling symlink leaf whose target is outside the project
// previously passed validation because EvalSymlinks(joined) errored and the
// old code fell back to trusting the parent. os.WriteFile would then follow
// the symlink outside the project.
func TestSafePath_BlocksDanglingSymlinkLeafOutside(t *testing.T) {
	a := testApp(t)

	// Target outside the project that does not exist.
	outsideDir := t.TempDir()
	target := filepath.Join(outsideDir, "does-not-exist")

	require.NoError(t, os.Symlink(target, filepath.Join(a.currentLibrary, "leaf")))

	_, err := a.safePath("leaf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path escapes library root")

	// And confirm that no file was created at the outside target.
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr), "target should not have been materialized")
}

// A dangling leaf symlink pointing inside the project is still rejected —
// writers don't need leaf symlinks, and allowing them introduces a TOCTOU
// window where the target could be swapped between safePath and the write.
func TestSafePath_BlocksDanglingSymlinkLeafInside(t *testing.T) {
	a := testApp(t)

	target := filepath.Join(a.currentLibrary, "nothing-here.ds")
	require.NoError(t, os.Symlink(target, filepath.Join(a.currentLibrary, "leaf")))

	_, err := a.safePath("leaf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "leaf symlinks are not allowed")
}

// Even a live symlink leaf pointing to a file inside the project is rejected.
func TestSafePath_BlocksLiveSymlinkLeafInsideRoot(t *testing.T) {
	a := testApp(t)

	target := filepath.Join(a.currentLibrary, "real.ds")
	require.NoError(t, os.WriteFile(target, []byte("content"), 0644))
	require.NoError(t, os.Symlink(target, filepath.Join(a.currentLibrary, "leaf")))

	_, err := a.safePath("leaf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "leaf symlinks are not allowed")
}

func TestConfigRoundTrip(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.writeConfig(Config{LastLibraryPath: "/some/path"}))
	require.NoError(t, a.SetActiveLibraryFile("play.ds"))

	data, err := os.ReadFile(a.configPath)
	require.NoError(t, err)

	var config Config
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Equal(t, "/some/path", config.LastLibraryPath)
	assert.Equal(t, "play.ds", config.LastActiveLibraryFile)
}

// writeConfig is symmetric: empty fields clear. The old merge-based
// saveConfig could not clear LastActiveLibraryFile, which broke project
// switches (the previous project's active file stayed behind).
func TestConfigRoundTrip_ClearActiveFile(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetActiveLibraryFile("play.ds"))
	require.NoError(t, a.writeConfig(Config{LastLibraryPath: "/new", LastActiveLibraryFile: ""}))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/new", cfg.LastLibraryPath)
	assert.Equal(t, "", cfg.LastActiveLibraryFile)
}

// Regression: a pre-rename config on disk uses `lastProjectPath` and
// `lastActiveProjectFile`. readConfig must migrate both into the new
// field names so the first launch after upgrade finds the user's library
// and last-opened file. Dropping either is a data-loss bug.
func TestReadConfig_MigratesLegacyFields(t *testing.T) {
	a := testAppWithConfig(t)

	legacy := []byte(`{"lastProjectPath":"/old/library","lastActiveProjectFile":"play.ds"}`)
	require.NoError(t, os.WriteFile(a.configPath, legacy, 0644))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/old/library", cfg.LastLibraryPath)
	assert.Equal(t, "play.ds", cfg.LastActiveLibraryFile)
}

// Regression: when the new fields are already populated, a legacy field
// must NOT clobber them. This handles the case where a user downgrades,
// the old build writes both shapes, then they upgrade again.
func TestReadConfig_NewFieldsTakePrecedenceOverLegacy(t *testing.T) {
	a := testAppWithConfig(t)

	mixed := []byte(`{"lastLibraryPath":"/new","lastProjectPath":"/old"}`)
	require.NoError(t, os.WriteFile(a.configPath, mixed, 0644))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/new", cfg.LastLibraryPath)
}

// RevealLibraryInExplorer must refuse to spawn anything when no library
// is open. The positive path shells out to a platform-native command,
// which we don't cover in unit tests.
func TestRevealLibraryInExplorer_NoLibrary(t *testing.T) {
	a := &App{}
	err := a.RevealLibraryInExplorer()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no library open")
}

// ReadLibraryFile used to call saveConfig on every read — a disk write on
// a hot path. The current contract is: config is only touched via
// SetActiveLibraryFile (on file switch) or ChangeLibraryLocation (on library
// switch).
func TestReadLibraryFile_DoesNotWriteConfig(t *testing.T) {
	a := testAppWithConfig(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("x"), 0644))

	// Pre-seed config with a known value so we can detect mutation.
	require.NoError(t, a.writeConfig(Config{LastLibraryPath: "/x", LastActiveLibraryFile: "prev.ds"}))
	statBefore, err := os.Stat(a.configPath)
	require.NoError(t, err)

	// Small sleep so mtime would differ if a write did happen.
	time.Sleep(10 * time.Millisecond)

	_, err = a.ReadLibraryFile("play.ds")
	require.NoError(t, err)

	statAfter, err := os.Stat(a.configPath)
	require.NoError(t, err)
	assert.Equal(t, statBefore.ModTime(), statAfter.ModTime())

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "prev.ds", cfg.LastActiveLibraryFile)
}

func TestSetActiveLibraryFile_Persists(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetActiveLibraryFile("act1.ds"))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "act1.ds", cfg.LastActiveLibraryFile)
}

func TestCreateLibraryFile_Dedup(t *testing.T) {
	a := testApp(t)

	path1, err := a.CreateLibraryFile("test", "content1")
	require.NoError(t, err)
	assert.Equal(t, "test.ds", path1)

	path2, err := a.CreateLibraryFile("test", "content2")
	require.NoError(t, err)
	assert.Equal(t, "test-1.ds", path2)

	path3, err := a.CreateLibraryFile("test", "content3")
	require.NoError(t, err)
	assert.Equal(t, "test-2.ds", path3)
}

func TestCreateLibraryFile_AddsSuffix(t *testing.T) {
	a := testApp(t)

	path, err := a.CreateLibraryFile("My Play", "content")
	require.NoError(t, err)
	assert.Equal(t, "My Play.ds", path)
}

func TestCreateLibraryFile_PreservesSuffix(t *testing.T) {
	a := testApp(t)

	path, err := a.CreateLibraryFile("play.ds", "content")
	require.NoError(t, err)
	assert.Equal(t, "play.ds", path)
}

func TestGetLibraryTree_Filters(t *testing.T) {
	a := testApp(t)

	os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(a.currentLibrary, "readme.md"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(a.currentLibrary, "notes.txt"), []byte("test"), 0644)

	gitDir := filepath.Join(a.currentLibrary, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref"), 0644)

	nodes, err := a.GetLibraryTree()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "play.ds", nodes[0].Name)
	assert.Equal(t, "file", nodes[0].Kind)
}

func TestGetLibraryTree_EmptyLibraryReturnsEmptySlice(t *testing.T) {
	// Regression: return a non-nil slice so JSON serializes as [] and
	// the frontend's `.length` / `.map` are safe.
	a := testApp(t)

	nodes, err := a.GetLibraryTree()
	require.NoError(t, err)
	require.NotNil(t, nodes)
	require.Len(t, nodes, 0)
}

func TestGetLibraryTree_NoLibraryReturnsEmptySlice(t *testing.T) {
	a := &App{}

	nodes, err := a.GetLibraryTree()
	require.NoError(t, err)
	require.NotNil(t, nodes)
	require.Len(t, nodes, 0)
}

func TestGetLibraryTree_SkipsDownstageDir(t *testing.T) {
	a := testApp(t)

	os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("test"), 0644)
	dsDir := filepath.Join(a.currentLibrary, ".downstage")
	os.MkdirAll(dsDir, 0755)
	os.WriteFile(filepath.Join(dsDir, "internal.ds"), []byte("test"), 0644)

	nodes, err := a.GetLibraryTree()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "play.ds", nodes[0].Name)
}

func TestGetLibraryTree_NestsFoldersFirstAlpha(t *testing.T) {
	a := testApp(t)

	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "b-folder"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "a-folder"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "a-folder", "nested.ds"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "c.ds"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "b.ds"), []byte("x"), 0644))

	nodes, err := a.GetLibraryTree()
	require.NoError(t, err)
	require.Len(t, nodes, 4)

	// Folders first, alpha within.
	assert.Equal(t, "a-folder", nodes[0].Name)
	assert.Equal(t, "folder", nodes[0].Kind)
	assert.Equal(t, "b-folder", nodes[1].Name)
	// Then files, alpha.
	assert.Equal(t, "b.ds", nodes[2].Name)
	assert.Equal(t, "c.ds", nodes[3].Name)

	// Nested file lives under the folder's Children.
	require.Len(t, nodes[0].Children, 1)
	assert.Equal(t, "nested.ds", nodes[0].Children[0].Name)
	assert.Equal(t, "a-folder/nested.ds", nodes[0].Children[0].Path)
}

// --- Folder operations ---

func TestCreateLibraryFolder_CreatesDirectory(t *testing.T) {
	a := testApp(t)

	require.NoError(t, a.CreateLibraryFolder("act-one"))
	info, err := os.Stat(filepath.Join(a.currentLibrary, "act-one"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateLibraryFolder_RejectsExistingPath(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "exists"), []byte("x"), 0644))

	err := a.CreateLibraryFolder("exists")
	assert.Error(t, err)
}

func TestCreateLibraryFolder_RejectsTraversal(t *testing.T) {
	a := testApp(t)
	err := a.CreateLibraryFolder("../outside")
	assert.Error(t, err)
}

func TestMoveLibraryEntry_MovesFile(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("x"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "archive"), 0755))

	newPath, err := a.MoveLibraryEntry("play.ds", "archive/play.ds")
	require.NoError(t, err)
	assert.Equal(t, "archive/play.ds", newPath)

	_, err = os.Stat(filepath.Join(a.currentLibrary, "play.ds"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(a.currentLibrary, "archive", "play.ds"))
	require.NoError(t, err)
}

func TestMoveLibraryEntry_RejectsExistingDst(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "a.ds"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "b.ds"), []byte("y"), 0644))

	_, err := a.MoveLibraryEntry("a.ds", "b.ds")
	assert.Error(t, err)
}

func TestMoveLibraryEntry_RejectsMoveIntoSelf(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "outer", "inner"), 0755))

	_, err := a.MoveLibraryEntry("outer", "outer/inner/outer")
	assert.Error(t, err)
}

func TestMoveLibraryEntry_RejectsTraversal(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("x"), 0644))

	_, err := a.MoveLibraryEntry("play.ds", "../escape.ds")
	assert.Error(t, err)
}

func TestRenameLibraryEntry_RenamesInPlace(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "sub", "play.ds"), []byte("x"), 0644))

	newPath, err := a.RenameLibraryEntry("sub/play.ds", "renamed.ds")
	require.NoError(t, err)
	assert.Equal(t, "sub/renamed.ds", newPath)
}

func TestRenameLibraryEntry_RejectsSeparatorsInNewName(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("x"), 0644))

	_, err := a.RenameLibraryEntry("play.ds", "sub/escape.ds")
	assert.Error(t, err)
}

func TestRenameLibraryEntry_RejectsDotNames(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "play.ds"), []byte("x"), 0644))

	_, err := a.RenameLibraryEntry("play.ds", "..")
	assert.Error(t, err)
}

// A move after the file has at least one snapshot should auto-commit
// a "Move ... → ..." commit so the repo stays in a clean state, and
// so GetRevisions's follow-renames walk has an explicit rename
// boundary to detect.
// The move IS recorded as a commit in git (repo hygiene — clean
// staged state + explicit rename signal for follow-renames), but
// should not appear in the revisions panel since it's a structural
// bookkeeping commit with no content change.
func TestMoveLibraryEntry_CreatesRenameCommitNotSurfacedInRevisions(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "archive"), 0755))

	_, err := a.MoveLibraryEntry("play.ds", "archive/play.ds")
	require.NoError(t, err)

	// The revisions panel only shows "initial" — the move commit is
	// filtered out. `git log` on the CLI would still show it.
	revs, err := a.GetRevisions("archive/play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revs, 1)
	assert.Contains(t, revs[0].Message, "initial")
	// And it's attributed to the pre-rename path.
	assert.Equal(t, "play.ds", revs[0].Path)

	// Confirm there are actually two commits in the repo (the move
	// was committed in git, just filtered from the panel).
	r, err := git.PlainOpen(a.currentLibrary)
	require.NoError(t, err)
	head, err := r.Head()
	require.NoError(t, err)
	iter, err := r.Log(&git.LogOptions{From: head.Hash()})
	require.NoError(t, err)
	count := 0
	require.NoError(t, iter.ForEach(func(_ *object.Commit) error {
		count++
		return nil
	}))
	assert.Equal(t, 2, count, "move commit exists in git, filtered from panel")
}

// GetRevisions must follow a file's history across renames. The panel
// should show all commits that touched the content, regardless of
// the path it lived at at the time.
func TestGetRevisions_FollowsRename(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial draft"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v2"))
	require.NoError(t, a.SnapshotFile("play.ds", "second draft"))
	require.NoError(t, os.MkdirAll(filepath.Join(a.currentLibrary, "archive"), 0755))

	_, err := a.MoveLibraryEntry("play.ds", "archive/play.ds")
	require.NoError(t, err)

	require.NoError(t, a.WriteLibraryFile("archive/play.ds", "v3"))
	require.NoError(t, a.SnapshotFile("archive/play.ds", "after move"))

	revs, err := a.GetRevisions("archive/play.ds", 0)
	require.NoError(t, err)

	// Expected, newest first. The pure-rename commit (no content
	// change, just a move) is deliberately filtered out — restoring
	// it would be a no-op, and the user's mental model is "saves of
	// content," not "structural moves."
	//  - "after move" (at archive/play.ds)
	//  - "second draft" (at play.ds)
	//  - "initial draft" (at play.ds)
	require.Len(t, revs, 3)
	assert.Contains(t, revs[0].Message, "after move")
	assert.Equal(t, "archive/play.ds", revs[0].Path)
	assert.Contains(t, revs[1].Message, "second draft")
	assert.Equal(t, "play.ds", revs[1].Path)
	assert.Contains(t, revs[2].Message, "initial draft")
	assert.Equal(t, "play.ds", revs[2].Path)

	// Viewing a pre-rename revision uses the historical path.
	content, err := a.ReadFileAtRevision(revs[2].Path, revs[2].Hash)
	require.NoError(t, err)
	assert.Equal(t, "v1", content)
	content, err = a.ReadFileAtRevision(revs[1].Path, revs[1].Hash)
	require.NoError(t, err)
	assert.Equal(t, "v2", content)
}

// A move performed before the file has ever been snapshotted should
// still work — there's nothing to stage as a deletion, nothing to
// commit (no tracked changes relative to HEAD), and the function
// must not error.
func TestMoveLibraryEntry_BeforeFirstSnapshot(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("draft.ds", "v1"))

	newPath, err := a.MoveLibraryEntry("draft.ds", "renamed.ds")
	require.NoError(t, err)
	assert.Equal(t, "renamed.ds", newPath)

	// File is at the new location.
	_, err = os.Stat(filepath.Join(a.currentLibrary, "renamed.ds"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(a.currentLibrary, "draft.ds"))
	assert.True(t, os.IsNotExist(err))
}

func TestWriteLibraryFile_NoAutoCommit(t *testing.T) {
	a := testApp(t)
	_, err := git.PlainInit(a.currentLibrary, false)
	require.NoError(t, err)

	err = a.WriteLibraryFile("play.ds", "content")
	require.NoError(t, err)

	r, err := git.PlainOpen(a.currentLibrary)
	require.NoError(t, err)

	_, err = r.Head()
	assert.Error(t, err, "should have no commits after WriteLibraryFile")
}

func TestSnapshotFile(t *testing.T) {
	a := testApp(t)

	err := a.WriteLibraryFile("play.ds", "content")
	require.NoError(t, err)

	err = a.SnapshotFile("play.ds", "initial version")
	require.NoError(t, err)

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "initial version", revisions[0].Message)
}

func TestGetRevisions_Order(t *testing.T) {
	a := testApp(t)

	a.WriteLibraryFile("play.ds", "v1")
	a.SnapshotFile("play.ds", "first")

	a.WriteLibraryFile("play.ds", "v2")
	a.SnapshotFile("play.ds", "second")

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	assert.Equal(t, "second", revisions[0].Message)
	assert.Equal(t, "first", revisions[1].Message)
}

func TestGetRevisions_NoLibraryReturnsEmptySlice(t *testing.T) {
	a := &App{}

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.NotNil(t, revisions, "must return non-nil slice so JSON emits []")
	require.Len(t, revisions, 0)
}

func TestReadLibraryFile_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	_, err := a.ReadLibraryFile("../../etc/passwd")
	assert.Error(t, err)
}

func TestWriteLibraryFile_BlocksTraversal(t *testing.T) {
	a := testApp(t)

	err := a.WriteLibraryFile("../../tmp/evil.txt", "evil")
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

func TestSnapshotFile_NoLibrary_ReturnsError(t *testing.T) {
	a := &App{}

	err := a.SnapshotFile("play.ds", "msg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no library open")
}

// After a snapshot the worktree is clean — a second call with the same
// contents must not create an empty commit. Frontend matches on
// ErrNothingToSnapshot to show an informational toast rather than an error.
func TestSnapshotFile_NothingToCommit_ReturnsSentinel(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "content"))
	require.NoError(t, a.SnapshotFile("play.ds", "first"))

	err := a.SnapshotFile("play.ds", "noop")
	assert.ErrorIs(t, err, ErrNothingToSnapshot)
}

// When a user has a global git identity configured, snapshots must be
// attributed to them — not to the "Downstage Write" default.
func TestSnapshotFile_UsesGlobalGitIdentity(t *testing.T) {
	a := testApp(t)

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	gitconfig := "[user]\n\tname = Ada Lovelace\n\temail = ada@example.com\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpHome, ".gitconfig"), []byte(gitconfig), 0644))

	require.NoError(t, a.WriteLibraryFile("play.ds", "content"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "Ada Lovelace", revisions[0].Author)
}

func TestSnapshotFile_FallsBackToDefaultIdentity(t *testing.T) {
	a := testApp(t)

	// Isolate from the developer's own gitconfig so the test is
	// deterministic on any machine.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	require.NoError(t, a.WriteLibraryFile("play.ds", "content"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, defaultSnapshotAuthorName, revisions[0].Author)
}

func TestGetLibraryTree_Sorted(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "zulu.ds"), []byte("z"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "Alpha.ds"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "mike.ds"), []byte("m"), 0644))

	nodes, err := a.GetLibraryTree()
	require.NoError(t, err)
	require.Len(t, nodes, 3)

	paths := []string{nodes[0].Path, nodes[1].Path, nodes[2].Path}
	assert.Equal(t, []string{"Alpha.ds", "mike.ds", "zulu.ds"}, paths)
}

func TestGetRevisions_BoundedByLimit(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v2"))
	require.NoError(t, a.SnapshotFile("play.ds", "two"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v3"))
	require.NoError(t, a.SnapshotFile("play.ds", "three"))

	revisions, err := a.GetRevisions("play.ds", 2)
	require.NoError(t, err)
	assert.Len(t, revisions, 2)
	// Newest first — limit cuts from the tail.
	assert.Equal(t, "three", revisions[0].Message)
	assert.Equal(t, "two", revisions[1].Message)
}

func TestDiagnosticSeverity_UnknownDefaultsToInfo(t *testing.T) {
	// Zero-value DiagnosticSeverity is 0, which is not one of the known
	// protocol constants. We deliberately default unknowns to "info" rather
	// than "error" so an uncategorized diagnostic never blocks a user.
	assert.Equal(t, "info", diagnosticSeverity(0))
}

// ReadFileAtRevision must return the blob content at the named commit,
// even if the working copy has since diverged. This is the core primitive
// behind the desktop restore flow.
func TestReadFileAtRevision_ReturnsContentAtCommit(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))
	require.NoError(t, a.WriteLibraryFile("play.ds", "v2"))
	require.NoError(t, a.SnapshotFile("play.ds", "two"))

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	// revisions[1] is the older commit ("one") — its blob should be "v1"
	// even though the working copy is now "v2".
	content, err := a.ReadFileAtRevision("play.ds", revisions[1].Hash)
	require.NoError(t, err)
	assert.Equal(t, "v1", content)

	// Newest revision round-trips too.
	content, err = a.ReadFileAtRevision("play.ds", revisions[0].Hash)
	require.NoError(t, err)
	assert.Equal(t, "v2", content)
}

func TestReadFileAtRevision_UnknownHashReturnsError(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))

	_, err := a.ReadFileAtRevision("play.ds", "deadbeef")
	assert.Error(t, err)
}

// GetPreferences must round-trip every field SetPreferences wrote. This is
// the contract Store/Workspace relies on; a regression here breaks
// persisted UI prefs silently.
func TestPreferences_RoundTrip(t *testing.T) {
	a := testAppWithConfig(t)

	in := Preferences{
		Theme:              "dark",
		PreviewHidden:      true,
		SpellcheckDisabled: true,
		SidebarCollapsed:   true,
	}
	require.NoError(t, a.SetPreferences(in))

	out, err := a.GetPreferences()
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

// Fresh install: Theme empty in JSON → normalized to "system". Other
// zero-value booleans match the default behavior so no normalization is
// needed there.
func TestPreferences_DefaultTheme(t *testing.T) {
	a := testAppWithConfig(t)

	out, err := a.GetPreferences()
	require.NoError(t, err)
	assert.Equal(t, "system", out.Theme)
	assert.False(t, out.PreviewHidden)
	assert.False(t, out.SpellcheckDisabled)
	assert.False(t, out.SidebarCollapsed)
}

// SetPreferences must not clobber non-preference Config fields. If it did,
// persisting a theme change would forget the user's last project path.
func TestPreferences_DoesNotClobberNonPrefFields(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.writeConfig(Config{
		LastLibraryPath:       "/tmp/project",
		LastActiveLibraryFile: "play.ds",
	}))

	require.NoError(t, a.SetPreferences(Preferences{Theme: "dark"}))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/project", cfg.LastLibraryPath)
	assert.Equal(t, "play.ds", cfg.LastActiveLibraryFile)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
}

// Switching projects (ChangeLibraryLocation) must not clear persisted
// preferences. They live in the same Config but are logically decoupled
// from the project pointer.
func TestPreferences_SurvivesLibrarySwitch(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetPreferences(Preferences{
		Theme:            "dark",
		SidebarCollapsed: true,
	}))

	// Simulate the project-switch code path: read, mutate project pointer,
	// write.
	cfg, err := a.readConfig()
	require.NoError(t, err)
	cfg.LastLibraryPath = "/tmp/other"
	cfg.LastActiveLibraryFile = ""
	require.NoError(t, a.writeConfig(cfg))

	out, err := a.GetPreferences()
	require.NoError(t, err)
	assert.Equal(t, "dark", out.Theme)
	assert.True(t, out.SidebarCollapsed)
}

// GetWindowState returns a zero value when no state has been saved.
func TestWindowState_DefaultZero(t *testing.T) {
	a := testAppWithConfig(t)

	ws, err := a.GetWindowState()
	require.NoError(t, err)
	assert.Equal(t, WindowState{}, ws)
}

// SaveWindowBounds persists size+position and sets Placed=true so the
func TestWindowState_SaveBoundsRoundTrip(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SaveWindowBounds(1400, 900))

	ws, err := a.GetWindowState()
	require.NoError(t, err)
	assert.Equal(t, 1400, ws.Width)
	assert.Equal(t, 900, ws.Height)
}

// A pre-release config that still carries the dropped fields
// (x, y, maximized, placed) must not break unmarshal. JSON ignores
// unknown fields, so the read succeeds and the stale keys are silently
// dropped on the next write.
func TestWindowState_IgnoresLegacyFields(t *testing.T) {
	a := testAppWithConfig(t)

	legacy := []byte(`{"windowState":{"width":1400,"height":900,"x":120,"y":60,"maximized":true,"placed":true}}`)
	require.NoError(t, os.WriteFile(a.configPath, legacy, 0644))

	ws, err := a.GetWindowState()
	require.NoError(t, err)
	assert.Equal(t, 1400, ws.Width)
	assert.Equal(t, 900, ws.Height)
}

// WindowState writes must not clobber Preferences or project fields.
// This is the key guarantee of the updateConfig migration.
func TestWindowState_DoesNotClobberOtherFields(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.writeConfig(Config{
		LastLibraryPath:       "/tmp/project",
		LastActiveLibraryFile: "play.ds",
		Preferences:           Preferences{Theme: "dark", SidebarCollapsed: true},
	}))

	require.NoError(t, a.SaveWindowBounds(1400, 900))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/project", cfg.LastLibraryPath)
	assert.Equal(t, "play.ds", cfg.LastActiveLibraryFile)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
	assert.True(t, cfg.Preferences.SidebarCollapsed)
}

// ChangeLibraryLocation previously called writeConfig(Config{...}) which
// zeroed Preferences + WindowState on every project switch. The
// updateConfig migration preserves them.
func TestChangeLibraryLocation_PreservesPrefsAndWindowState(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetPreferences(Preferences{Theme: "dark", SidebarCollapsed: true}))
	require.NoError(t, a.SaveWindowBounds(1400, 900))

	// Simulate the project-switch path — the same mutator
	// ChangeLibraryLocation uses.
	require.NoError(t, a.updateConfig(func(c *Config) {
		c.LastLibraryPath = "/tmp/other"
		c.LastActiveLibraryFile = ""
	}))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/other", cfg.LastLibraryPath)
	assert.Empty(t, cfg.LastActiveLibraryFile)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
	assert.True(t, cfg.Preferences.SidebarCollapsed)
	assert.Equal(t, 1400, cfg.WindowState.Width)
}

// Concurrent writers across subtrees (Preferences vs WindowState)
// must each land in the final Config. The updateConfig lock guards
// the RMW cycle, so interleaved goroutines can't lose each other's
// subtrees.
func TestConfig_ConcurrentSubtreeWritesPreserveBoth(t *testing.T) {
	a := testAppWithConfig(t)

	const iterations = 40
	done := make(chan struct{}, 2)

	go func() {
		for i := 0; i < iterations; i++ {
			_ = a.SetPreferences(Preferences{Theme: "dark"})
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < iterations; i++ {
			_ = a.SaveWindowBounds(1200, 800)
		}
		done <- struct{}{}
	}()

	<-done
	<-done

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
	assert.Equal(t, 1200, cfg.WindowState.Width)
	assert.Equal(t, 800, cfg.WindowState.Height)
}

// ShowAboutDialog must not panic when Wails hasn't initialized the
// runtime context — defensive early-return covers edge cases like the
// dialog being dispatched during teardown or tests.
func TestShowAboutDialog_NoCtxNoPanic(t *testing.T) {
	a := &App{}
	assert.NotPanics(t, func() {
		_ = a.ShowAboutDialog()
	})
}

// Version defaults to "dev" when ldflags doesn't inject anything.
// Release builds override this via -X; we only assert the fallback
// here so the About dialog never shows an empty version string.
func TestVersion_HasFallback(t *testing.T) {
	assert.NotEmpty(t, Version)
}

func TestReadFileAtRevision_RejectsAbsolutePaths(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteLibraryFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))
	revisions, _ := a.GetRevisions("play.ds", 0)

	_, err := a.ReadFileAtRevision("/etc/passwd", revisions[0].Hash)
	assert.Error(t, err)
}

// --- External-file open flow ---

func TestReadExternalFile_RejectsRelativePath(t *testing.T) {
	a := testApp(t)
	_, err := a.ReadExternalFile("play.ds")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute path")
}

func TestReadExternalFile_RejectsNonDsExtension(t *testing.T) {
	a := testApp(t)
	tmp := t.TempDir()
	bogus := filepath.Join(tmp, "readme.txt")
	require.NoError(t, os.WriteFile(bogus, []byte("not a manuscript"), 0644))

	_, err := a.ReadExternalFile(bogus)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".ds")
}

func TestReadExternalFile_RejectsLeafSymlink(t *testing.T) {
	a := testApp(t)
	tmp := t.TempDir()
	target := filepath.Join(tmp, "real.ds")
	link := filepath.Join(tmp, "link.ds")
	require.NoError(t, os.WriteFile(target, []byte("real content"), 0644))
	require.NoError(t, os.Symlink(target, link))

	_, err := a.ReadExternalFile(link)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestReadExternalFile_ReadsContent(t *testing.T) {
	a := testApp(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "ext.ds")
	require.NoError(t, os.WriteFile(path, []byte("# External\n"), 0644))

	result, err := a.ReadExternalFile(path)
	require.NoError(t, err)
	assert.Equal(t, "# External\n", result.Content)
}

// A file that lives inside the current library should be flagged so
// the frontend routes through the normal selectFile flow, not the
// external-file banner.
func TestReadExternalFile_DetectsInsideLibrary(t *testing.T) {
	a := testApp(t)
	inside := filepath.Join(a.currentLibrary, "sub", "doc.ds")
	require.NoError(t, os.MkdirAll(filepath.Dir(inside), 0755))
	require.NoError(t, os.WriteFile(inside, []byte("# Inside\n"), 0644))

	result, err := a.ReadExternalFile(inside)
	require.NoError(t, err)
	assert.True(t, result.InsideLibrary)
	assert.Equal(t, "sub/doc.ds", result.RelativePath)
}

func TestAddExternalFileToLibrary_CopiesToRoot(t *testing.T) {
	a := testApp(t)
	src := filepath.Join(t.TempDir(), "ext.ds")
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

	rel, err := a.AddExternalFileToLibrary(src, "")
	require.NoError(t, err)
	assert.Equal(t, "ext.ds", rel)

	data, err := os.ReadFile(filepath.Join(a.currentLibrary, "ext.ds"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestAddExternalFileToLibrary_CollisionSuffix(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentLibrary, "ext.ds"), []byte("existing"), 0644))

	src := filepath.Join(t.TempDir(), "ext.ds")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0644))

	rel, err := a.AddExternalFileToLibrary(src, "")
	require.NoError(t, err)
	assert.Equal(t, "ext-1.ds", rel)
}

func TestAddExternalFileToLibrary_NoLibrary(t *testing.T) {
	a := &App{}
	src := filepath.Join(t.TempDir(), "ext.ds")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0644))

	_, err := a.AddExternalFileToLibrary(src, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no library open")
}

func TestAddExternalFileToLibrary_RejectsTraversal(t *testing.T) {
	a := testApp(t)
	src := filepath.Join(t.TempDir(), "ext.ds")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0644))

	_, err := a.AddExternalFileToLibrary(src, "../outside")
	assert.Error(t, err)
}
