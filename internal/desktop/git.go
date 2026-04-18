package desktop

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
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
