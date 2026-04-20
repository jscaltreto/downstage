package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const dictionaryPath = ".downstage/dictionary.txt"

// LibraryFile represents a .ds file in the library directory. Kept as
// the wire type returned inside `LibraryNode.Kind == "file"` children;
// paths use forward-slash separators so the frontend can do path math
// without branching on platform.
type LibraryFile struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

// LibraryNode is a single entry in the library tree returned by
// GetLibraryTree. Folders carry Children; files carry UpdatedAt. Path is
// always library-root-relative with forward-slash separators. Kind is
// "folder" or "file" — the frontend branches on it.
type LibraryNode struct {
	Path      string        `json:"path"`
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	Children  []LibraryNode `json:"children,omitempty"`
	UpdatedAt string        `json:"updatedAt,omitempty"`
}

// ChangeLibraryLocation shows a folder picker and switches the active
// library to the chosen directory. Returns the selected path, or "" when
// the user cancels.
func (a *App) ChangeLibraryLocation() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose Library Location",
	})
	if err != nil {
		return "", err
	}
	if selection == "" {
		return "", nil
	}
	a.currentLibrary = selection

	// Switch libraries: update only the library fields. Prior to the
	// updateConfig migration this path wrote Config{...} wholesale,
	// silently zeroing Preferences and WindowState every time the user
	// changed folders. Now other subtrees are preserved.
	if err := a.updateConfig(func(c *Config) {
		c.LastLibraryPath = selection
		c.LastActiveLibraryFile = ""
	}); err != nil {
		slog.Warn("persisting library switch failed", "err", err)
	}

	_ = a.ensureGitRepo()
	return selection, nil
}

// GetLibraryTree returns the library as a nested tree: folders first
// (alpha per level), then files (alpha per level). `.git` and
// `.downstage` are skipped, and only `.ds` files appear as leaves.
// Always returns a non-nil slice so the frontend's `.length` / `.map`
// are safe on a fresh install.
func (a *App) GetLibraryTree() ([]LibraryNode, error) {
	if a.currentLibrary == "" {
		return []LibraryNode{}, nil
	}
	nodes, err := buildLibraryChildren(a.currentLibrary, "")
	if err != nil {
		return []LibraryNode{}, err
	}
	return nodes, nil
}

// buildLibraryChildren reads the entries at root/relDir and returns the
// LibraryNode children sorted folders-first then files-alpha. Nested
// folders recurse.
func buildLibraryChildren(root, relDir string) ([]LibraryNode, error) {
	fullDir := filepath.Join(root, relDir)
	entries, err := os.ReadDir(fullDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []LibraryNode{}, nil
		}
		return []LibraryNode{}, err
	}
	folders := []LibraryNode{}
	files := []LibraryNode{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if name == ".git" || name == ".downstage" {
				continue
			}
			childRel := filepath.ToSlash(filepath.Join(relDir, name))
			children, err := buildLibraryChildren(root, filepath.Join(relDir, name))
			if err != nil {
				slog.Warn("library tree: skipping folder", "path", childRel, "err", err)
				continue
			}
			folders = append(folders, LibraryNode{
				Path:     childRel,
				Name:     name,
				Kind:     "folder",
				Children: children,
			})
			continue
		}
		if !strings.HasSuffix(name, ".ds") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			slog.Warn("library tree: stat failed", "name", name, "err", err)
			continue
		}
		childRel := filepath.ToSlash(filepath.Join(relDir, name))
		files = append(files, LibraryNode{
			Path:      childRel,
			Name:      name,
			Kind:      "file",
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
	}
	sort.SliceStable(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})
	sort.SliceStable(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	out := make([]LibraryNode, 0, len(folders)+len(files))
	out = append(out, folders...)
	out = append(out, files...)
	return out, nil
}

// CreateLibraryFolder creates a new folder inside the library. Unlike
// CreateLibraryFile, the name is taken as-is — no collision suffix,
// because folder creation is an explicit user action and a silent
// rename would be surprising. Returns an error if the path exists.
func (a *App) CreateLibraryFolder(relPath string) error {
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("a file or folder already exists at %q", relPath)
	}
	return os.MkdirAll(fullPath, 0755)
}

// MoveLibraryEntry moves srcRel to dstRel inside the library. Rejects
// when the destination already exists (no overwrite, no auto-suffix)
// and when the destination would land inside the source (can't move a
// folder into itself). Returns the new rel path with forward slashes
// so the frontend can update its active-file tracking.
//
// The move is auto-committed. A filesystem-only rename would leave the
// repo in an inconsistent intermediate state (old path tracked-but-
// missing, new path untracked); a structural action like move is
// deliberate user input, so recording it as a commit up front keeps
// the repo clean and gives git's rename detection an unambiguous
// signal that `GetRevisions`'s follow-renames walk can hook into.
// Auto-commit here does not extend to content saves — those remain
// explicit via SnapshotFile.
func (a *App) MoveLibraryEntry(srcRel, dstRel string) (string, error) {
	srcFull, err := a.safePath(srcRel)
	if err != nil {
		return "", fmt.Errorf("source: %w", err)
	}
	dstFull, err := a.safePath(dstRel)
	if err != nil {
		return "", fmt.Errorf("destination: %w", err)
	}
	if _, err := os.Stat(srcFull); err != nil {
		return "", fmt.Errorf("stat source: %w", err)
	}
	if _, err := os.Stat(dstFull); err == nil {
		return "", fmt.Errorf("a file or folder already exists at %q", dstRel)
	}
	// Reject "move folder into itself" before calling Rename.
	srcClean := filepath.Clean(srcFull)
	dstClean := filepath.Clean(dstFull)
	if pathInsideRoot(dstClean, srcClean) {
		return "", fmt.Errorf("cannot move a folder into itself")
	}
	if err := os.MkdirAll(filepath.Dir(dstFull), 0755); err != nil {
		return "", fmt.Errorf("make destination dir: %w", err)
	}

	// Collect the pre-move relative paths to stage as deletions. For
	// files this is just srcRel; for folders we need every .ds file
	// underneath, so git records the full rename pattern rather than
	// a single directory entry (go-git stages files, not directories).
	srcSubPaths, err := collectTrackedPaths(srcFull, srcRel)
	if err != nil {
		return "", fmt.Errorf("enumerate source: %w", err)
	}

	if err := os.Rename(srcFull, dstFull); err != nil {
		return "", fmt.Errorf("rename: %w", err)
	}

	cleanDst := filepath.ToSlash(filepath.Clean(dstRel))

	// Commit the move. On a fresh library with no commits, this is the
	// first commit and creates HEAD. We treat a repo-init failure or
	// commit failure as non-fatal — the filesystem move already
	// succeeded, and the next snapshot will pick up the changes — but
	// we surface the error so the frontend can toast it.
	if err := a.commitMove(srcSubPaths, cleanDst); err != nil {
		slog.Warn("library move: commit failed (filesystem move succeeded)", "err", err)
	}

	return cleanDst, nil
}

// collectTrackedPaths returns the library-relative forward-slash paths
// that will need to be staged as deletions after a rename. For a
// single file, that's one path. For a folder, it's every .ds file
// inside (git stages files, not directories).
func collectTrackedPaths(fullPath, relPath string) ([]string, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{filepath.ToSlash(relPath)}, nil
	}
	var paths []string
	err = filepath.WalkDir(fullPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".downstage" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".ds") {
			return nil
		}
		rel, err := filepath.Rel(fullPath, p)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(relPath, rel)))
		return nil
	})
	return paths, err
}

// commitMove stages the pre-move source paths as deletions and the
// post-move destination as added files, then commits. Assumes the
// filesystem rename already happened.
func (a *App) commitMove(srcPaths []string, dstRel string) error {
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
	}
	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		r, err = git.PlainInit(a.currentLibrary, false)
	}
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	// Stage deletions (old paths). `Remove` is a no-op-with-error when
	// the path wasn't previously tracked (e.g. a brand-new file that
	// hadn't been snapshotted yet) — swallow that case so the first-
	// snapshot-also-moved flow still works.
	for _, p := range srcPaths {
		if _, err := w.Remove(p); err != nil && !errors.Is(err, index.ErrEntryNotFound) {
			slog.Warn("library move: remove failed", "path", p, "err", err)
		}
	}

	// Stage additions. If dst is a directory, walk it and Add every
	// .ds file. AddWithOptions(All: true, Path: dstRel) would do this
	// in one go but we keep the behavior explicit so we can log per-
	// file failures.
	fullDst := filepath.Join(a.currentLibrary, dstRel)
	info, err := os.Stat(fullDst)
	if err != nil {
		return fmt.Errorf("stat destination: %w", err)
	}
	if info.IsDir() {
		err := filepath.WalkDir(fullDst, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(d.Name(), ".ds") {
				return nil
			}
			rel, err := filepath.Rel(a.currentLibrary, p)
			if err != nil {
				return err
			}
			_, addErr := w.Add(filepath.ToSlash(rel))
			return addErr
		})
		if err != nil {
			return fmt.Errorf("add destination files: %w", err)
		}
	} else {
		if _, err := w.Add(dstRel); err != nil {
			return fmt.Errorf("add destination: %w", err)
		}
	}

	// Nothing to commit? (e.g. user moved an untracked file to a new
	// location — no tree delta from HEAD's perspective.) Skip silently.
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	if status.IsClean() {
		return nil
	}

	message := fmt.Sprintf("Move to %s", dstRel)
	if len(srcPaths) > 0 {
		message = fmt.Sprintf("Move %s → %s", srcPaths[0], dstRel)
	}
	_, err = w.Commit(message, &git.CommitOptions{Author: snapshotAuthor(r)})
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// RenameLibraryEntry renames srcRel's basename to newName, preserving
// its parent directory. newName must not contain separators. Returns
// the new rel path with forward slashes.
func (a *App) RenameLibraryEntry(srcRel, newName string) (string, error) {
	if newName == "" {
		return "", fmt.Errorf("new name is empty")
	}
	if strings.ContainsRune(newName, '/') || strings.ContainsRune(newName, filepath.Separator) {
		return "", fmt.Errorf("new name must not contain path separators")
	}
	if newName == "." || newName == ".." {
		return "", fmt.Errorf("invalid new name")
	}
	parent := filepath.ToSlash(filepath.Dir(filepath.Clean(srcRel)))
	var dstRel string
	if parent == "." || parent == "/" {
		dstRel = newName
	} else {
		dstRel = parent + "/" + newName
	}
	return a.MoveLibraryEntry(srcRel, dstRel)
}

func (a *App) ReadLibraryFile(relPath string) (string, error) {
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) WriteLibraryFile(relPath string, content string) error {
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

func (a *App) CreateLibraryFile(name string, content string) (string, error) {
	if a.currentLibrary == "" {
		return "", fmt.Errorf("no library open")
	}

	filename := name
	if !strings.HasSuffix(strings.ToLower(filename), ".ds") {
		filename += ".ds"
	}

	base := strings.TrimSuffix(filename, ".ds")
	finalName := filename
	counter := 1
	for {
		fullPath, err := a.safePath(finalName)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		finalName = fmt.Sprintf("%s-%d.ds", base, counter)
		counter++
	}

	fullPath, err := a.safePath(finalName)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return finalName, nil
}

// ExternalFileResult is returned by ReadExternalFile. When the chosen
// path happens to live inside the current library, InsideLibrary is
// true and RelativePath is its library-relative path — the frontend
// should route through the normal selectFile flow instead of showing
// the external-file banner.
type ExternalFileResult struct {
	Content       string `json:"content"`
	InsideLibrary bool   `json:"insideLibrary"`
	RelativePath  string `json:"relativePath"`
}

// externalFileMaxBytes caps how much we'll pull off disk for an
// external .ds open. Plaintext manuscripts fit in well under 100 KiB;
// the 5 MiB cap is 50× generous and guards against symlinks-to-huge-
// files or malformed inputs.
const externalFileMaxBytes = 5 * 1024 * 1024

// OpenExternalFileDialog shows a native open-file dialog filtered to
// .ds files. Returns the chosen absolute path, or "" when the user
// cancels. This is a separate binding from ChangeLibraryLocation (a
// directory picker) because the frontend's File → Open flow needs a
// file picker, not a folder picker.
func (a *App) OpenExternalFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Downstage Files (*.ds)", Pattern: "*.ds"},
		},
	})
}

// ReadExternalFile reads a .ds file from an arbitrary absolute path
// into memory for the read-only "File → Open" preview. Guards:
//
//   - absolute path required (File → Open dialog returns absolute paths;
//     reject anything else at the boundary)
//   - case-insensitive .ds extension required
//   - leaf symlinks rejected (mirrors safePath's rule; guards against
//     symlinks-to-hostile-targets)
//   - size capped at externalFileMaxBytes
//
// When the resolved path lives inside the current library, returns
// InsideLibrary=true + RelativePath so the frontend can route through
// the normal selectFile flow — it should NOT present the external-file
// banner for a file the library already owns.
func (a *App) ReadExternalFile(absPath string) (ExternalFileResult, error) {
	if !filepath.IsAbs(absPath) {
		return ExternalFileResult{}, fmt.Errorf("absolute path required")
	}
	if !strings.HasSuffix(strings.ToLower(absPath), ".ds") {
		return ExternalFileResult{}, fmt.Errorf("only .ds files can be opened")
	}

	// Reject leaf symlinks before following them. os.Lstat reveals the
	// link itself; EvalSymlinks would quietly follow.
	if info, err := os.Lstat(absPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return ExternalFileResult{}, fmt.Errorf("symlinks are not allowed")
		}
	} else {
		return ExternalFileResult{}, fmt.Errorf("stat path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return ExternalFileResult{}, fmt.Errorf("resolve path: %w", err)
	}

	// Detect in-library: a path under the active library should flow
	// through the normal file-open path, not the external-file banner.
	var insideLibrary bool
	var relativePath string
	if a.currentLibrary != "" {
		if libRoot, err := filepath.EvalSymlinks(a.currentLibrary); err == nil {
			if pathInsideRoot(resolved, libRoot) {
				insideLibrary = true
				if rel, err := filepath.Rel(libRoot, resolved); err == nil {
					relativePath = filepath.ToSlash(rel)
				}
			}
		}
	}

	// Size-cap the read. Anything larger is almost certainly not a
	// manuscript the editor can render usefully.
	f, err := os.Open(resolved)
	if err != nil {
		return ExternalFileResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return ExternalFileResult{}, fmt.Errorf("stat file: %w", err)
	}
	if stat.Size() > externalFileMaxBytes {
		return ExternalFileResult{}, fmt.Errorf("file is too large (max %d bytes)", externalFileMaxBytes)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return ExternalFileResult{}, fmt.Errorf("read file: %w", err)
	}
	return ExternalFileResult{
		Content:       string(data),
		InsideLibrary: insideLibrary,
		RelativePath:  relativePath,
	}, nil
}

// AddExternalFileToLibrary copies absSrc into the current library under
// destRelDir (empty string = library root) and returns the new path
// relative to the library. Collision handling matches CreateLibraryFile:
// an existing target name gets a `-N` suffix.
func (a *App) AddExternalFileToLibrary(absSrc string, destRelDir string) (string, error) {
	if a.currentLibrary == "" {
		return "", fmt.Errorf("no library open")
	}
	if !filepath.IsAbs(absSrc) {
		return "", fmt.Errorf("absolute source path required")
	}
	if !strings.HasSuffix(strings.ToLower(absSrc), ".ds") {
		return "", fmt.Errorf("only .ds files can be added")
	}

	srcData, err := os.ReadFile(absSrc)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}

	base := filepath.Base(absSrc)
	// Strip `.ds` suffix for the collision-suffix loop so we get
	// `foo-1.ds`, not `foo.ds-1`.
	stem := strings.TrimSuffix(base, ".ds")
	finalRel := base
	counter := 1
	for {
		candidate := filepath.Join(destRelDir, finalRel)
		fullPath, err := a.safePath(candidate)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		finalRel = fmt.Sprintf("%s-%d.ds", stem, counter)
		counter++
	}

	candidate := filepath.Join(destRelDir, finalRel)
	fullPath, err := a.safePath(candidate)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("make destination dir: %w", err)
	}
	if err := os.WriteFile(fullPath, srcData, 0644); err != nil {
		return "", fmt.Errorf("write destination: %w", err)
	}
	return filepath.ToSlash(candidate), nil
}

func (a *App) dictionaryFile() string {
	if a.currentLibrary == "" {
		return ""
	}
	return filepath.Join(a.currentLibrary, dictionaryPath)
}

func (a *App) readDictionary() []string {
	path := a.dictionaryFile()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var words []string
	for _, line := range strings.Split(string(data), "\n") {
		w := strings.TrimSpace(line)
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}

func (a *App) writeDictionary(words []string) error {
	path := a.dictionaryFile()
	if path == "" {
		return fmt.Errorf("no library open")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	sort.Strings(words)
	return os.WriteFile(path, []byte(strings.Join(words, "\n")+"\n"), 0644)
}

func (a *App) GetSpellAllowlist() []string {
	words := a.readDictionary()
	if words == nil {
		return []string{}
	}
	return words
}

func (a *App) AddSpellAllowlistWord(word string) (bool, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return false, nil
	}
	words := a.readDictionary()
	key := strings.ToLower(word)
	for _, existing := range words {
		if strings.ToLower(existing) == key {
			return false, nil
		}
	}
	words = append(words, word)
	if err := a.writeDictionary(words); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) RemoveSpellAllowlistWord(word string) (bool, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return false, nil
	}
	words := a.readDictionary()
	key := strings.ToLower(word)
	var next []string
	found := false
	for _, existing := range words {
		if strings.ToLower(existing) == key {
			found = true
		} else {
			next = append(next, existing)
		}
	}
	if !found {
		return false, nil
	}
	if err := a.writeDictionary(next); err != nil {
		return false, err
	}
	return true, nil
}
