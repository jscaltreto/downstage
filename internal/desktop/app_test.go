package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	require.NoError(t, os.Symlink(target, filepath.Join(a.currentProject, "leaf")))

	_, err := a.safePath("leaf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path escapes project root")

	// And confirm that no file was created at the outside target.
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr), "target should not have been materialized")
}

// A dangling leaf symlink pointing inside the project is still rejected —
// writers don't need leaf symlinks, and allowing them introduces a TOCTOU
// window where the target could be swapped between safePath and the write.
func TestSafePath_BlocksDanglingSymlinkLeafInside(t *testing.T) {
	a := testApp(t)

	target := filepath.Join(a.currentProject, "nothing-here.ds")
	require.NoError(t, os.Symlink(target, filepath.Join(a.currentProject, "leaf")))

	_, err := a.safePath("leaf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "leaf symlinks are not allowed")
}

// Even a live symlink leaf pointing to a file inside the project is rejected.
func TestSafePath_BlocksLiveSymlinkLeafInsideRoot(t *testing.T) {
	a := testApp(t)

	target := filepath.Join(a.currentProject, "real.ds")
	require.NoError(t, os.WriteFile(target, []byte("content"), 0644))
	require.NoError(t, os.Symlink(target, filepath.Join(a.currentProject, "leaf")))

	_, err := a.safePath("leaf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "leaf symlinks are not allowed")
}

func TestConfigRoundTrip(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.writeConfig(Config{LastProjectPath: "/some/path"}))
	require.NoError(t, a.SetActiveProjectFile("play.ds"))

	data, err := os.ReadFile(a.configPath)
	require.NoError(t, err)

	var config Config
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Equal(t, "/some/path", config.LastProjectPath)
	assert.Equal(t, "play.ds", config.LastActiveProjectFile)
}

// writeConfig is symmetric: empty fields clear. The old merge-based
// saveConfig could not clear LastActiveProjectFile, which broke project
// switches (the previous project's active file stayed behind).
func TestConfigRoundTrip_ClearActiveFile(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetActiveProjectFile("play.ds"))
	require.NoError(t, a.writeConfig(Config{LastProjectPath: "/new", LastActiveProjectFile: ""}))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/new", cfg.LastProjectPath)
	assert.Equal(t, "", cfg.LastActiveProjectFile)
}

// ReadProjectFile used to call saveConfig on every read — a disk write on
// a hot path. The current contract is: config is only touched via
// SetActiveProjectFile (on file switch) or OpenProjectFolder (on project
// switch).
func TestReadProjectFile_DoesNotWriteConfig(t *testing.T) {
	a := testAppWithConfig(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentProject, "play.ds"), []byte("x"), 0644))

	// Pre-seed config with a known value so we can detect mutation.
	require.NoError(t, a.writeConfig(Config{LastProjectPath: "/x", LastActiveProjectFile: "prev.ds"}))
	statBefore, err := os.Stat(a.configPath)
	require.NoError(t, err)

	// Small sleep so mtime would differ if a write did happen.
	time.Sleep(10 * time.Millisecond)

	_, err = a.ReadProjectFile("play.ds")
	require.NoError(t, err)

	statAfter, err := os.Stat(a.configPath)
	require.NoError(t, err)
	assert.Equal(t, statBefore.ModTime(), statAfter.ModTime())

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "prev.ds", cfg.LastActiveProjectFile)
}

func TestSetActiveProjectFile_Persists(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetActiveProjectFile("act1.ds"))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "act1.ds", cfg.LastActiveProjectFile)
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

	revisions, err := a.GetRevisions("play.ds", 0)
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

	revisions, err := a.GetRevisions("play.ds", 0)
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

func TestSnapshotFile_NoProject_ReturnsError(t *testing.T) {
	a := &App{}

	err := a.SnapshotFile("play.ds", "msg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project open")
}

// After a snapshot the worktree is clean — a second call with the same
// contents must not create an empty commit. Frontend matches on
// ErrNothingToSnapshot to show an informational toast rather than an error.
func TestSnapshotFile_NothingToCommit_ReturnsSentinel(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteProjectFile("play.ds", "content"))
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

	require.NoError(t, a.WriteProjectFile("play.ds", "content"))
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

	require.NoError(t, a.WriteProjectFile("play.ds", "content"))
	require.NoError(t, a.SnapshotFile("play.ds", "initial"))

	revisions, err := a.GetRevisions("play.ds", 0)
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, defaultSnapshotAuthorName, revisions[0].Author)
}

func TestGetProjectFiles_Sorted(t *testing.T) {
	a := testApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(a.currentProject, "zulu.ds"), []byte("z"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentProject, "Alpha.ds"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(a.currentProject, "mike.ds"), []byte("m"), 0644))

	files, err := a.GetProjectFiles()
	require.NoError(t, err)
	require.Len(t, files, 3)

	paths := []string{files[0].Path, files[1].Path, files[2].Path}
	assert.Equal(t, []string{"Alpha.ds", "mike.ds", "zulu.ds"}, paths)
}

func TestGetRevisions_BoundedByLimit(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))
	require.NoError(t, a.WriteProjectFile("play.ds", "v2"))
	require.NoError(t, a.SnapshotFile("play.ds", "two"))
	require.NoError(t, a.WriteProjectFile("play.ds", "v3"))
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
	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))
	require.NoError(t, a.WriteProjectFile("play.ds", "v2"))
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
	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
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
		LastProjectPath:       "/tmp/project",
		LastActiveProjectFile: "play.ds",
	}))

	require.NoError(t, a.SetPreferences(Preferences{Theme: "dark"}))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/project", cfg.LastProjectPath)
	assert.Equal(t, "play.ds", cfg.LastActiveProjectFile)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
}

// Switching projects (OpenProjectFolder) must not clear persisted
// preferences. They live in the same Config but are logically decoupled
// from the project pointer.
func TestPreferences_SurvivesProjectSwitch(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetPreferences(Preferences{
		Theme:            "dark",
		SidebarCollapsed: true,
	}))

	// Simulate the project-switch code path: read, mutate project pointer,
	// write.
	cfg, err := a.readConfig()
	require.NoError(t, err)
	cfg.LastProjectPath = "/tmp/other"
	cfg.LastActiveProjectFile = ""
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
// restore path distinguishes "never moved" from legitimate (0, 0).
func TestWindowState_SaveBoundsRoundTrip(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SaveWindowBounds(1400, 900, 120, 60))

	ws, err := a.GetWindowState()
	require.NoError(t, err)
	assert.Equal(t, 1400, ws.Width)
	assert.Equal(t, 900, ws.Height)
	assert.Equal(t, 120, ws.X)
	assert.Equal(t, 60, ws.Y)
	assert.True(t, ws.Placed)
	assert.False(t, ws.Maximized)
}

// SaveWindowMaximized flips the flag without touching persisted bounds.
// The maximize-then-quit path relies on this: we want the previous
// normal bounds preserved for the next unmaximize.
func TestWindowState_MaximizedDoesNotClobberBounds(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SaveWindowBounds(1400, 900, 120, 60))
	require.NoError(t, a.SaveWindowMaximized(true))

	ws, err := a.GetWindowState()
	require.NoError(t, err)
	assert.True(t, ws.Maximized)
	assert.Equal(t, 1400, ws.Width)
	assert.Equal(t, 900, ws.Height)
	assert.Equal(t, 120, ws.X)
	assert.Equal(t, 60, ws.Y)
}

// WindowState writes must not clobber Preferences or project fields.
// This is the key guarantee of the updateConfig migration.
func TestWindowState_DoesNotClobberOtherFields(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.writeConfig(Config{
		LastProjectPath:       "/tmp/project",
		LastActiveProjectFile: "play.ds",
		Preferences:           Preferences{Theme: "dark", SidebarCollapsed: true},
	}))

	require.NoError(t, a.SaveWindowBounds(1400, 900, 120, 60))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/project", cfg.LastProjectPath)
	assert.Equal(t, "play.ds", cfg.LastActiveProjectFile)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
	assert.True(t, cfg.Preferences.SidebarCollapsed)
}

// OpenProjectFolder previously called writeConfig(Config{...}) which
// zeroed Preferences + WindowState on every project switch. The
// updateConfig migration preserves them.
func TestOpenProjectFolder_PreservesPrefsAndWindowState(t *testing.T) {
	a := testAppWithConfig(t)

	require.NoError(t, a.SetPreferences(Preferences{Theme: "dark", SidebarCollapsed: true}))
	require.NoError(t, a.SaveWindowBounds(1400, 900, 120, 60))

	// Simulate the project-switch path — the same mutator
	// OpenProjectFolder uses.
	require.NoError(t, a.updateConfig(func(c *Config) {
		c.LastProjectPath = "/tmp/other"
		c.LastActiveProjectFile = ""
	}))

	cfg, err := a.readConfig()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/other", cfg.LastProjectPath)
	assert.Empty(t, cfg.LastActiveProjectFile)
	assert.Equal(t, "dark", cfg.Preferences.Theme)
	assert.True(t, cfg.Preferences.SidebarCollapsed)
	assert.Equal(t, 1400, cfg.WindowState.Width)
	assert.True(t, cfg.WindowState.Placed)
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
			_ = a.SaveWindowBounds(1200, 800, 10, 20)
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
	assert.True(t, cfg.WindowState.Placed)
}

func TestReadFileAtRevision_RejectsAbsolutePaths(t *testing.T) {
	a := testApp(t)
	require.NoError(t, a.WriteProjectFile("play.ds", "v1"))
	require.NoError(t, a.SnapshotFile("play.ds", "one"))
	revisions, _ := a.GetRevisions("play.ds", 0)

	_, err := a.ReadFileAtRevision("/etc/passwd", revisions[0].Hash)
	assert.Error(t, err)
}
