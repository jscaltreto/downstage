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

const hiddenRevisionsPath = ".downstage/hidden-revisions.txt"

type LibraryFile struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

type LibraryNode struct {
	Path      string        `json:"path"`
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	Children  []LibraryNode `json:"children,omitempty"`
	UpdatedAt string        `json:"updatedAt,omitempty"`
}

func (a *App) ChangeLibraryLocation() (string, error) {
	// Run the directory dialog WITHOUT libMu held — the user may sit
	// in the picker indefinitely and we mustn't block every other RPC
	// while they decide.
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose Library Location",
	})
	if err != nil {
		return "", err
	}
	if selection == "" {
		return "", nil
	}

	// Now take the writer Lock for the brief field swap; release
	// before the slower disk operations (config write + git init).
	a.libMu.Lock()
	a.currentLibrary = selection
	a.libMu.Unlock()

	if err := a.updateConfig(func(c *Config) {
		c.LastLibraryPath = selection
		c.LastActiveLibraryFile = ""
	}); err != nil {
		slog.Warn("persisting library switch failed", "err", err)
	}

	_ = a.ensureGitRepo()
	return selection, nil
}

func (a *App) GetLibraryTree() ([]LibraryNode, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return []LibraryNode{}, nil
	}
	nodes, err := buildLibraryChildren(a.currentLibrary, "")
	if err != nil {
		return []LibraryNode{}, err
	}
	return nodes, nil
}

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

func (a *App) CreateLibraryFolder(relPath string) error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("a file or folder already exists at %q", relPath)
	}
	return os.MkdirAll(fullPath, 0755)
}

func (a *App) MoveLibraryEntry(srcRel, dstRel string) (string, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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
	srcClean := filepath.Clean(srcFull)
	dstClean := filepath.Clean(dstFull)
	if pathInsideRoot(dstClean, srcClean) {
		return "", fmt.Errorf("cannot move a folder into itself")
	}
	if err := os.MkdirAll(filepath.Dir(dstFull), 0755); err != nil {
		return "", fmt.Errorf("make destination dir: %w", err)
	}

	srcSubPaths, err := collectTrackedPaths(srcFull, srcRel)
	if err != nil {
		return "", fmt.Errorf("enumerate source: %w", err)
	}

	if err := os.Rename(srcFull, dstFull); err != nil {
		return "", fmt.Errorf("rename: %w", err)
	}

	cleanDst := filepath.ToSlash(filepath.Clean(dstRel))

	if err := a.commitMove(srcSubPaths, cleanDst); err != nil {
		slog.Warn("library move: commit failed (filesystem move succeeded)", "err", err)
	}

	return cleanDst, nil
}

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

// commitMove is private; callers (MoveLibraryEntry, RenameLibraryEntry)
// hold libMu.RLock for the duration of their work, so this helper
// must NOT acquire a second RLock (would deadlock against a pending
// writer).
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

	for _, p := range srcPaths {
		if _, err := w.Remove(p); err != nil && !errors.Is(err, index.ErrEntryNotFound) {
			slog.Warn("library move: remove failed", "path", p, "err", err)
		}
	}

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

// DeleteLibraryFile removes a single .ds file from disk WITHOUT committing
// the deletion. For tracked files this leaves the worktree in a
// status=Deleted state — the UI surfaces it in the Deleted section so the
// user can Restore (from HEAD) or Permanently delete (commit). Untracked
// files are gone for good, since there's no HEAD blob to restore from.
//
// Sibling changes are intentionally NOT picked up — the snapshot /
// review-changes flows are the explicit paths for those.
func (a *App) DeleteLibraryFile(relPath string) error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if !strings.HasSuffix(clean, ".ds") {
		return fmt.Errorf("only .ds files can be deleted via this API")
	}
	fullPath, err := a.safePath(clean)
	if err != nil {
		return err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory; use MoveLibraryEntry for folders")
	}
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

// RestoreLibraryFile reads the path's HEAD blob and writes it back to disk,
// re-staging the file so the worktree returns to clean. Precondition: the
// path must currently be in Deleted state in the worktree. Restore over a
// Modified file is rejected — the user should use DiscardPaths instead so
// they can opt into losing local changes explicitly.
func (a *App) RestoreLibraryFile(relPath string) error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if _, err := a.safePath(clean); err != nil {
		return err
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		return fmt.Errorf("library is not a git repository")
	}
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	entry, ok := status[clean]
	if !ok {
		return fmt.Errorf("file %q is not in a restorable state", clean)
	}
	if classifyDirty(entry) != DirtyDeleted {
		return fmt.Errorf("file %q is not deleted (use DiscardPaths to revert modifications)", clean)
	}

	if err := restoreFromHEAD(r, a.currentLibrary, clean); err != nil {
		return err
	}
	if _, err := w.Add(clean); err != nil {
		return fmt.Errorf("re-stage: %w", err)
	}
	return nil
}

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
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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
		_, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("stat %q: %w", fullPath, err)
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

type ExternalFileResult struct {
	Content       string `json:"content"`
	InsideLibrary bool   `json:"insideLibrary"`
	RelativePath  string `json:"relativePath"`
}

const externalFileMaxBytes = 5 * 1024 * 1024

func (a *App) OpenExternalFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Downstage Files (*.ds)", Pattern: "*.ds"},
		},
	})
}

func (a *App) ReadExternalFile(absPath string) (ExternalFileResult, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if !filepath.IsAbs(absPath) {
		return ExternalFileResult{}, fmt.Errorf("absolute path required")
	}
	if !strings.HasSuffix(strings.ToLower(absPath), ".ds") {
		return ExternalFileResult{}, fmt.Errorf("only .ds files can be opened")
	}

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

func (a *App) AddExternalFileToLibrary(absSrc string, destRelDir string) (string, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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
	stem := strings.TrimSuffix(base, ".ds")
	finalRel := base
	counter := 1
	for {
		candidate := filepath.Join(destRelDir, finalRel)
		fullPath, err := a.safePath(candidate)
		if err != nil {
			return "", err
		}
		_, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("stat %q: %w", fullPath, err)
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
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	words := a.readDictionary()
	if words == nil {
		return []string{}
	}
	return words
}

func (a *App) AddSpellAllowlistWord(word string) (bool, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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
	a.libMu.RLock()
	defer a.libMu.RUnlock()
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

func (a *App) hiddenRevisionsFile() string {
	if a.currentLibrary == "" {
		return ""
	}
	return filepath.Join(a.currentLibrary, hiddenRevisionsPath)
}

func (a *App) readHiddenRevisions() ([]string, error) {
	path := a.hiddenRevisionsFile()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read hidden revisions: %w", err)
	}
	var hashes []string
	for _, line := range strings.Split(string(data), "\n") {
		h := strings.TrimSpace(line)
		if h != "" {
			hashes = append(hashes, h)
		}
	}
	return hashes, nil
}

func (a *App) writeHiddenRevisions(hashes []string) error {
	path := a.hiddenRevisionsFile()
	if path == "" {
		return fmt.Errorf("no library open")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(hashes))
	cleaned := make([]string, 0, len(hashes))
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, dup := seen[h]; dup {
			continue
		}
		seen[h] = struct{}{}
		cleaned = append(cleaned, h)
	}
	sort.Strings(cleaned)

	body := strings.Join(cleaned, "\n")
	if body != "" {
		body += "\n"
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (a *App) GetHiddenRevisions() ([]string, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	a.hiddenMu.Lock()
	defer a.hiddenMu.Unlock()
	h, err := a.readHiddenRevisions()
	if err != nil {
		return nil, err
	}
	if h == nil {
		return []string{}, nil
	}
	return h, nil
}

func (a *App) HideRevision(hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return fmt.Errorf("hash is empty")
	}
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	a.hiddenMu.Lock()
	defer a.hiddenMu.Unlock()
	current, err := a.readHiddenRevisions()
	if err != nil {
		return err
	}
	for _, existing := range current {
		if existing == hash {
			return nil
		}
	}
	current = append(current, hash)
	return a.writeHiddenRevisions(current)
}

func (a *App) UnhideRevision(hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return fmt.Errorf("hash is empty")
	}
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	a.hiddenMu.Lock()
	defer a.hiddenMu.Unlock()
	current, err := a.readHiddenRevisions()
	if err != nil {
		return err
	}
	next := make([]string, 0, len(current))
	found := false
	for _, existing := range current {
		if existing == hash {
			found = true
			continue
		}
		next = append(next, existing)
	}
	if !found {
		return nil
	}
	return a.writeHiddenRevisions(next)
}
