package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
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

func (a *App) ReadFileAtRevision(relPath string, hash string) (string, error) {
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
