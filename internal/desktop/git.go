package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

var ErrNothingToSnapshot = errors.New("downstage: nothing-to-snapshot")

const (
	defaultSnapshotAuthorName  = "Downstage Write"
	defaultSnapshotAuthorEmail = "hello@getdownstage.com"
)

type Revision struct {
	Hash      string `json:"hash"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
}

func snapshotAuthor(r *git.Repository) *object.Signature {
	name := defaultSnapshotAuthorName
	email := defaultSnapshotAuthorEmail

	if cfg, err := r.ConfigScoped(config.GlobalScope); err == nil {
		if cfg.User.Name != "" {
			name = cfg.User.Name
		}
		if cfg.User.Email != "" {
			email = cfg.User.Email
		}
	}

	return &object.Signature{Name: name, Email: email, When: time.Now()}
}

func (a *App) SnapshotFile(relPath string, message string) error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
	}
	if _, err := a.safePath(relPath); err != nil {
		return err
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		r, err = git.PlainInit(a.currentLibrary, false)
	}
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if _, err = w.Add(relPath); err != nil {
		return err
	}

	status, err := w.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		return ErrNothingToSnapshot
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: snapshotAuthor(r),
	})
	return err
}

const defaultRevisionLimit = 100

func (a *App) GetRevisions(relPath string, limit int) ([]Revision, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return []Revision{}, nil
	}
	if limit <= 0 {
		limit = defaultRevisionLimit
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		return []Revision{}, nil
	}
	if err != nil {
		return nil, err
	}

	head, err := r.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return []Revision{}, nil
	}
	if err != nil {
		return nil, err
	}

	iter, err := r.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	tracked := filepath.ToSlash(filepath.Clean(relPath))
	revisions := make([]Revision, 0, limit)

	firstCommit := true
	errStop := errors.New("stop")

	err = iter.ForEach(func(c *object.Commit) error {
		blob, err := blobAtPath(c, tracked)
		if err != nil {
			return err
		}

		var parentBlob plumbing.Hash
		var parent *object.Commit
		if c.NumParents() > 0 {
			parent, err = c.Parent(0)
			if err != nil {
				return err
			}
			parentBlob, _ = blobHashAtPath(parent, tracked)
		}

		addedHere := !blob.IsZero() && parentBlob.IsZero()
		removedHere := blob.IsZero() && !parentBlob.IsZero()

		var renameSource string
		if addedHere && parent != nil {
			if sibling, ok := findSiblingBlob(parent, c, blob, tracked); ok {
				renameSource = sibling
			}
		}

		includeCommit := false
		switch {
		case renameSource != "":
			includeCommit = false
		case firstCommit && !blob.IsZero():
			includeCommit = true
		case parent == nil && !blob.IsZero():
			includeCommit = true
		case blob != parentBlob:
			includeCommit = true
		}

		if includeCommit {
			revisions = append(revisions, Revision{
				Hash:      c.Hash.String(),
				Path:      tracked,
				Message:   c.Message,
				Author:    c.Author.Name,
				Timestamp: c.Author.When.Format(time.RFC3339),
			})
			if len(revisions) >= limit {
				return errStop
			}
		}

		if renameSource != "" {
			tracked = renameSource
		} else if removedHere && parent != nil {
			if newPath, ok := findSiblingBlob(c, parent, parentBlob, tracked); ok {
				tracked = newPath
			}
		}

		firstCommit = false
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return nil, err
	}

	return revisions, nil
}

func blobAtPath(commit *object.Commit, path string) (plumbing.Hash, error) {
	return blobHashAtPath(commit, path)
}

func blobHashAtPath(commit *object.Commit, path string) (plumbing.Hash, error) {
	if commit == nil || path == "" {
		return plumbing.ZeroHash, nil
	}
	tree, err := commit.Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	entry, err := tree.File(path)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return plumbing.ZeroHash, nil
		}
		return plumbing.ZeroHash, err
	}
	return entry.Hash, nil
}

func findSiblingBlob(reference, other *object.Commit, blob plumbing.Hash, excludePath string) (string, bool) {
	if reference == nil || blob.IsZero() {
		return "", false
	}
	refTree, err := reference.Tree()
	if err != nil {
		return "", false
	}
	var foundPath string
	err = refTree.Files().ForEach(func(f *object.File) error {
		if f.Hash != blob {
			return nil
		}
		path := filepath.ToSlash(f.Name)
		if path == excludePath {
			return nil
		}
		if other != nil {
			otherBlob, _ := blobHashAtPath(other, path)
			if otherBlob == blob {
				return nil
			}
		}
		foundPath = path
		return storer.ErrStop
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		return "", false
	}
	return foundPath, foundPath != ""
}

type FileGitStatus struct {
	Dirty     bool   `json:"dirty"`
	HeadAt    string `json:"headAt"`
	HasHead   bool   `json:"hasHead"`
	Untracked bool   `json:"untracked"`
	Missing   bool   `json:"missing"`
}

func (a *App) GetFileGitStatus(relPath string) (FileGitStatus, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return FileGitStatus{}, fmt.Errorf("no library open")
	}
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return FileGitStatus{}, err
	}

	_, statErr := os.Stat(fullPath)
	missing := errors.Is(statErr, fs.ErrNotExist)
	if statErr != nil && !missing {
		return FileGitStatus{}, fmt.Errorf("stat file: %w", statErr)
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		if missing {
			return FileGitStatus{Missing: true}, nil
		}
		return FileGitStatus{Dirty: true, Untracked: true}, nil
	}
	if err != nil {
		return FileGitStatus{}, err
	}

	headAt, hasHead := lastCommitTimeForPath(r, relPath)

	out := FileGitStatus{
		HeadAt:  headAt,
		HasHead: hasHead,
		Missing: missing,
	}
	if missing {
		return out, nil
	}

	w, err := r.Worktree()
	if err != nil {
		return out, err
	}
	status, err := w.Status()
	if err != nil {
		return out, err
	}

	key := filepath.ToSlash(filepath.Clean(relPath))
	entry, ok := status[key]
	if !ok {
		return out, nil
	}
	if entry.Staging == git.Untracked || entry.Worktree == git.Untracked {
		out.Untracked = true
	}
	if entry.Staging != git.Unmodified || entry.Worktree != git.Unmodified {
		out.Dirty = true
	}
	return out, nil
}

func lastCommitTimeForPath(r *git.Repository, relPath string) (string, bool) {
	log, err := r.Log(&git.LogOptions{FileName: &relPath})
	if err != nil {
		return "", false
	}
	defer log.Close()

	c, err := log.Next()
	if err != nil || c == nil {
		return "", false
	}
	return c.Author.When.UTC().Format(time.RFC3339), true
}

type DirtyKind string

const (
	DirtyUntracked DirtyKind = "untracked"
	DirtyModified  DirtyKind = "modified"
	DirtyDeleted   DirtyKind = "deleted"
)

type DirtyPath struct {
	Path string    `json:"path"`
	Kind DirtyKind `json:"kind"`
}

type LibraryDirty struct {
	Plays    []DirtyPath `json:"plays"`
	Sidecars []DirtyPath `json:"sidecars"`
	Other    []DirtyPath `json:"other"`
	Count    int         `json:"count"`
}

const downstageSidecarPrefix = ".downstage/"

func (a *App) GetLibraryDirty() (LibraryDirty, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	empty := LibraryDirty{Plays: []DirtyPath{}, Sidecars: []DirtyPath{}, Other: []DirtyPath{}}
	if a.currentLibrary == "" {
		return empty, nil
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	w, err := r.Worktree()
	if err != nil {
		return empty, err
	}
	status, err := w.Status()
	if err != nil {
		return empty, err
	}

	out := empty
	for path, entry := range status {
		if entry.Staging == git.Unmodified && entry.Worktree == git.Unmodified {
			continue
		}
		kind := classifyDirty(entry)
		dp := DirtyPath{Path: path, Kind: kind}
		switch {
		case strings.HasPrefix(path, downstageSidecarPrefix):
			out.Sidecars = append(out.Sidecars, dp)
		case strings.HasSuffix(path, ".ds"):
			out.Plays = append(out.Plays, dp)
		default:
			out.Other = append(out.Other, dp)
		}
	}
	sortDirty(out.Plays)
	sortDirty(out.Sidecars)
	sortDirty(out.Other)
	out.Count = len(out.Plays) + len(out.Sidecars) + len(out.Other)
	return out, nil
}

func classifyDirty(entry *git.FileStatus) DirtyKind {
	// Worktree state takes priority — it's what the user sees.
	switch entry.Worktree {
	case git.Untracked:
		return DirtyUntracked
	case git.Deleted:
		return DirtyDeleted
	}
	switch entry.Staging {
	case git.Untracked:
		return DirtyUntracked
	case git.Deleted:
		return DirtyDeleted
	}
	return DirtyModified
}

func sortDirty(s []DirtyPath) {
	sort.SliceStable(s, func(i, j int) bool { return s[i].Path < s[j].Path })
}

// CommitPaths stages exactly the paths provided (Add for paths that exist on
// disk, Remove for paths that have been deleted) and commits once with the
// given message. Returns ErrNothingToSnapshot if no path produced a change.
func (a *App) CommitPaths(paths []string, message string) error {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	return a.commitPathsLocked(paths, message)
}

// commitPathsLocked is the lock-free body of CommitPaths. CALLER MUST HOLD
// a.libMu (read or write). Use this from inside other locked methods to
// avoid re-acquiring the lock (Go's sync.RWMutex doesn't recurse safely).
func (a *App) commitPathsLocked(paths []string, message string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no paths provided")
	}
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("commit message is empty")
	}
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		r, err = git.PlainInit(a.currentLibrary, false)
	}
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}

	for _, p := range paths {
		clean := filepath.ToSlash(filepath.Clean(p))
		if _, err := a.safePath(clean); err != nil {
			return fmt.Errorf("path %q: %w", p, err)
		}
		fullPath := filepath.Join(a.currentLibrary, clean)
		if _, statErr := os.Stat(fullPath); statErr == nil {
			if _, err := w.Add(clean); err != nil {
				return fmt.Errorf("add %q: %w", clean, err)
			}
		} else if errors.Is(statErr, fs.ErrNotExist) {
			// File missing on disk: stage the deletion. If the path was
			// untracked (never in the index), Remove errors with
			// ErrEntryNotFound — that just means there's nothing to commit
			// for this path, which is fine for the batch.
			if _, err := w.Remove(clean); err != nil && !errors.Is(err, index.ErrEntryNotFound) {
				return fmt.Errorf("remove %q: %w", clean, err)
			}
		} else {
			return fmt.Errorf("stat %q: %w", clean, statErr)
		}
	}

	status, err := w.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		return ErrNothingToSnapshot
	}

	_, err = w.Commit(message, &git.CommitOptions{Author: snapshotAuthor(r)})
	return err
}

// DiscardPaths reverts each path to its HEAD state without committing.
// Per-kind dispatch: untracked files are removed from disk, modified and
// deleted files are restored from HEAD.
func (a *App) DiscardPaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no paths provided")
	}
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return fmt.Errorf("no library open")
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

	for _, p := range paths {
		clean := filepath.ToSlash(filepath.Clean(p))
		if _, err := a.safePath(clean); err != nil {
			return fmt.Errorf("path %q: %w", p, err)
		}
		entry, ok := status[clean]
		if !ok {
			continue
		}
		kind := classifyDirty(entry)
		switch kind {
		case DirtyUntracked:
			fullPath := filepath.Join(a.currentLibrary, clean)
			if err := os.Remove(fullPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("remove untracked %q: %w", clean, err)
			}
		case DirtyModified, DirtyDeleted:
			if err := restoreFromHEAD(r, a.currentLibrary, clean); err != nil {
				return fmt.Errorf("restore %q: %w", clean, err)
			}
			if _, err := w.Add(clean); err != nil {
				return fmt.Errorf("re-stage %q: %w", clean, err)
			}
		}
	}
	return nil
}

// restoreFromHEAD reads the blob for relPath from HEAD's tree and writes it
// to disk at root/relPath. Caller is responsible for re-staging via w.Add.
// Returns an error if relPath does not exist in HEAD.
func restoreFromHEAD(r *git.Repository, root, relPath string) error {
	head, err := r.Head()
	if err != nil {
		return fmt.Errorf("read HEAD: %w", err)
	}
	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		return fmt.Errorf("load HEAD commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("HEAD tree: %w", err)
	}
	file, err := tree.File(relPath)
	if err != nil {
		return fmt.Errorf("file not in HEAD: %w", err)
	}
	contents, err := file.Contents()
	if err != nil {
		return fmt.Errorf("read blob: %w", err)
	}
	fullPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func (a *App) ReadFileAtRevision(relPath string, hash string) (string, error) {
	a.libMu.RLock()
	defer a.libMu.RUnlock()
	if a.currentLibrary == "" {
		return "", fmt.Errorf("no library open")
	}
	if _, err := a.safePath(relPath); err != nil {
		return "", err
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if err != nil {
		return "", err
	}

	resolved, err := r.ResolveRevision(plumbing.Revision(hash))
	if err != nil {
		return "", fmt.Errorf("resolving revision %q: %w", hash, err)
	}
	commit, err := r.CommitObject(*resolved)
	if err != nil {
		return "", fmt.Errorf("loading commit %s: %w", resolved.String(), err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	treePath := filepath.ToSlash(filepath.Clean(relPath))
	file, err := tree.File(treePath)
	if err != nil {
		return "", fmt.Errorf("file %q not found in revision %s: %w", relPath, resolved.String()[:7], err)
	}
	return file.Contents()
}
