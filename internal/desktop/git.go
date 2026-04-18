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

// Revision represents a git commit for a file.
type Revision struct {
	Hash      string `json:"hash"`
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

// SnapshotFile stages and commits a file in the project's Git repository.
// This is an explicit user action, not called automatically on every save.
// Returns ErrNothingToSnapshot when the working tree is clean after staging.
func (a *App) SnapshotFile(relPath string, message string) error {
	if a.currentProject == "" {
		return fmt.Errorf("no project open")
	}
	if _, err := a.safePath(relPath); err != nil {
		return err
	}

	r, err := git.PlainOpen(a.currentProject)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		r, err = git.PlainInit(a.currentProject, false)
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
// projects; pagination can be added later if needed.
const defaultRevisionLimit = 100

func (a *App) GetRevisions(relPath string, limit int) ([]Revision, error) {
	if a.currentProject == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultRevisionLimit
	}

	r, err := git.PlainOpen(a.currentProject)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		return []Revision{}, nil
	}
	if err != nil {
		return nil, err
	}

	log, err := r.Log(&git.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, err
	}

	revisions := make([]Revision, 0, limit)
	// Use a sentinel error to stop ForEach once we've collected enough.
	var errStop = errors.New("stop")
	err = log.ForEach(func(c *object.Commit) error {
		revisions = append(revisions, Revision{
			Hash:      c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Timestamp: c.Author.When.Format(time.RFC3339),
		})
		if len(revisions) >= limit {
			return errStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return nil, err
	}

	return revisions, nil
}

// FileGitStatus summarizes git-level state for a single project file, as
// surfaced in the desktop status bar. Booleans are negative-semantic where
// helpful so the zero value aligns with the common "tracked, clean" case.
//
//   - HasHead is true when at least one commit in the project's history
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
// When the project has no git repo yet, the file is reported as
// untracked and dirty — "no snapshots exist" is the truthful read.
func (a *App) GetFileGitStatus(relPath string) (FileGitStatus, error) {
	if a.currentProject == "" {
		return FileGitStatus{}, fmt.Errorf("no project open")
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

	r, err := git.PlainOpen(a.currentProject)
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
	if a.currentProject == "" {
		return "", fmt.Errorf("no project open")
	}
	if _, err := a.safePath(relPath); err != nil {
		// Allow the file to be absent from disk — we may be reading a
		// revision of a file that was later renamed or deleted. safePath
		// rejects traversal/absolute/symlink inputs regardless.
		// If it failed for any of THOSE reasons, refuse.
		return "", err
	}

	r, err := git.PlainOpen(a.currentProject)
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
