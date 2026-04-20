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

// ErrNothingToSnapshot is returned by SnapshotFile when the working tree has
// no staged changes for the file. The frontend matches on the error message
// prefix "downstage: nothing-to-snapshot" to show an informational toast
// rather than a red-banner error.
var ErrNothingToSnapshot = errors.New("downstage: nothing-to-snapshot")

// Default snapshot author used when the user has no global git identity.
const (
	defaultSnapshotAuthorName  = "Downstage Write"
	defaultSnapshotAuthorEmail = "hello@getdownstage.com"
)

// Revision represents a git commit for a file. Path is the library-
// relative location the file had at that specific commit — which may
// differ from the current path if the file has been moved/renamed
// since. The frontend uses Path when fetching a revision's content
// via ReadFileAtRevision so lookups work across rename boundaries.
type Revision struct {
	Hash      string `json:"hash"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
}

// snapshotAuthor returns the commit author for a snapshot. Prefer the user's
// global git identity (from ~/.gitconfig or $XDG_CONFIG_HOME/git/config) so
// snapshots show up under their normal author name in git log. Fall back to
// the default when no identity is configured — this keeps the desktop app
// functional on fresh machines without forcing an initial git setup.
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

// SnapshotFile stages and commits a file in the library's Git repository.
// This is an explicit user action, not called automatically on every save.
// Returns ErrNothingToSnapshot when the working tree is clean after staging.
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

// defaultRevisionLimit bounds how many revisions GetRevisions returns when
// the caller doesn't specify. Prevents unbounded IPC payloads on long-lived
// libraries; pagination can be added later if needed.
const defaultRevisionLimit = 100

func (a *App) GetRevisions(relPath string, limit int) ([]Revision, error) {
	// Same nil-slice-is-null hazard as GetLibraryTree: always hand
	// the frontend a real empty slice so `.length`/`.map` are safe.
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
		// Fresh repo with no commits yet — nothing to report.
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

	// Walk the log in reverse chronological order, tracking the file's
	// path as it changes across renames. This is the equivalent of
	// `git log --follow <path>` without go-git's native support for it.
	//
	// Starting at HEAD, the tracked path is the current path. At each
	// commit we check the blob at the tracked path; if the commit
	// introduced that blob AND a sibling path in the same commit held
	// the identical blob in the parent, treat it as a rename and move
	// the tracked path to the parent's name for subsequent iterations.
	//
	// Pure-rename commits (same blob, different path) are detected
	// and followed for history tracking but NOT surfaced as versions.
	// The user's mental model is "snapshots of content," not
	// "structural moves" — restoring a pure rename is semantically a
	// no-op since its content equals its neighbors'. Keeping them
	// internal to the walk also spares the revisions panel clutter
	// every time a file gets reorganized.
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

		// Was the tracked path introduced (absent in parent, present
		// here) or a newly-added blob? Candidate for a rename.
		addedHere := !blob.IsZero() && parentBlob.IsZero()
		removedHere := blob.IsZero() && !parentBlob.IsZero()

		// Look for a rename: same blob lived at a different path in
		// the parent. If found, this commit is a pure rename of the
		// tracked file — skip from the output but advance `tracked`
		// so pre-rename commits are attributed to the old path.
		var renameSource string
		if addedHere && parent != nil {
			if sibling, ok := findSiblingBlob(parent, c, blob, tracked); ok {
				renameSource = sibling
			}
		}

		// Include the commit iff it genuinely changed the tracked
		// file's content, and it's not a pure-rename bookkeeping
		// commit.
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

		// Advance `tracked` across rename boundaries for subsequent
		// iterations, regardless of whether we emitted this commit.
		if renameSource != "" {
			tracked = renameSource
		} else if removedHere && parent != nil {
			// File deleted (or renamed away) at this commit. If the
			// parent held the same blob at another path, follow it.
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

// blobAtPath returns the blob hash at `path` in `commit`. Returns a
// zero hash (with no error) if the path doesn't exist in the commit.
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

// findSiblingBlob searches `reference`'s tree for a file with the
// given blob hash that doesn't exist (or has a different hash) in
// `other`'s tree. Used to spot the pre-rename path of a file that
// was added at `tracked` in `other` and whose identical blob lived
// elsewhere in `reference`. Returns the sibling path when a unique
// match is found, "" when none, and only the first hit when multiple
// (rare — would require duplicate-content files).
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
		// Confirm the candidate path didn't hold this blob in the
		// other commit — otherwise it's not a rename, just a pre-
		// existing duplicate.
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

// FileGitStatus summarizes git-level state for a single library file, as
// surfaced in the desktop status bar. Booleans are negative-semantic where
// helpful so the zero value aligns with the common "tracked, clean" case.
//
//   - HasHead is true when at least one commit in the library's history
//     has touched relPath. HeadAt carries that commit's timestamp.
//   - Untracked is true when the file exists on disk but git has never
//     seen it (no index entry, no history).
//   - Missing is true when the file does not exist at relPath on disk —
//     deleted or renamed since the frontend last stored activeFile. The
//     UI treats this as "file moved or deleted" and shows neither a dirty
//     dot nor a snapshot age.
//   - Dirty is true when the file diverges from HEAD: either untracked,
//     or any non-Unmodified status in the worktree/staging tree.
type FileGitStatus struct {
	Dirty     bool   `json:"dirty"`
	HeadAt    string `json:"headAt"`
	HasHead   bool   `json:"hasHead"`
	Untracked bool   `json:"untracked"`
	Missing   bool   `json:"missing"`
}

// GetFileGitStatus returns the dirty/snapshot-age surface the desktop
// status bar renders for the active file.
//
// Path hygiene goes through safePath (same rules as the other writers).
// safePath accepts missing files by design, so existence is probed here
// explicitly: a renamed-away or deleted file flips Missing=true and the
// status bar shows a neutral label instead of stale metadata.
//
// When the library has no git repo yet, the file is reported as
// untracked and dirty — "no snapshots exist" is the truthful read.
func (a *App) GetFileGitStatus(relPath string) (FileGitStatus, error) {
	if a.currentLibrary == "" {
		return FileGitStatus{}, fmt.Errorf("no library open")
	}
	fullPath, err := a.safePath(relPath)
	if err != nil {
		return FileGitStatus{}, err
	}

	// Existence probe. safePath allows not-yet-existing paths; we must
	// check ourselves so Missing is reported cleanly.
	_, statErr := os.Stat(fullPath)
	missing := errors.Is(statErr, fs.ErrNotExist)
	if statErr != nil && !missing {
		return FileGitStatus{}, fmt.Errorf("stat file: %w", statErr)
	}

	r, err := git.PlainOpen(a.currentLibrary)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		// No repo means there is no HEAD and no tracking information at
		// all. A present-on-disk file is untracked+dirty; a missing file
		// is simply missing.
		if missing {
			return FileGitStatus{Missing: true}, nil
		}
		return FileGitStatus{Dirty: true, Untracked: true}, nil
	}
	if err != nil {
		return FileGitStatus{}, err
	}

	// History lookup (runs regardless of Missing so the UI can still
	// indicate "this path had history under this name" if useful).
	headAt, hasHead := lastCommitTimeForPath(r, relPath)

	out := FileGitStatus{
		HeadAt:  headAt,
		HasHead: hasHead,
		Missing: missing,
	}
	if missing {
		// A missing file is not meaningfully dirty — it's absent. The UI
		// surfaces Missing as its own state; dirty should stay false.
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

	// go-git indexes status entries by forward-slash path regardless of
	// platform. Callers pass OS-native separators, so normalize.
	key := filepath.ToSlash(filepath.Clean(relPath))
	entry, ok := status[key]
	if !ok {
		// Not in the status map → worktree matches HEAD.
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

// lastCommitTimeForPath returns the author timestamp (RFC3339 UTC) of the
// most recent commit whose diff includes relPath, and whether any such
// commit exists. Used by GetFileGitStatus for the "Last snapshot N ago"
// label. A zero-commit repo or a never-touched path returns ("", false).
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

// ReadFileAtRevision returns the contents of relPath at the given commit.
// Used by the frontend to preview and restore older versions. The path is
// validated with safePath (input hygiene only — the file need not currently
// exist on disk to have existed at an older revision). The hash must be a
// full or prefix SHA that resolves to a single commit.
func (a *App) ReadFileAtRevision(relPath string, hash string) (string, error) {
	if a.currentLibrary == "" {
		return "", fmt.Errorf("no library open")
	}
	if _, err := a.safePath(relPath); err != nil {
		// Allow the file to be absent from disk — we may be reading a
		// revision of a file that was later renamed or deleted. safePath
		// rejects traversal/absolute/symlink inputs regardless.
		// If it failed for any of THOSE reasons, refuse.
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

	// go-git uses forward-slash paths in trees. Normalize to avoid surprises
	// on Windows where callers might pass backslashes.
	treePath := filepath.ToSlash(filepath.Clean(relPath))
	file, err := tree.File(treePath)
	if err != nil {
		return "", fmt.Errorf("file %q not found in revision %s: %w", relPath, resolved.String()[:7], err)
	}
	return file.Contents()
}
