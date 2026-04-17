package desktop

import (
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Revision represents a git commit for a file.
type Revision struct {
	Hash      string `json:"hash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
}

// SnapshotFile stages and commits a file in the project's Git repository.
// This is an explicit user action, not called automatically on every save.
func (a *App) SnapshotFile(relPath string, message string) error {
	if a.currentProject == "" {
		return nil
	}
	if _, err := a.safePath(relPath); err != nil {
		return err
	}

	r, err := git.PlainOpen(a.currentProject)
	if err == git.ErrRepositoryNotExists {
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

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Downstage Write",
			Email: "hello@getdownstage.com",
			When:  time.Now(),
		},
	})
	return err
}

func (a *App) GetRevisions(relPath string) ([]Revision, error) {
	if a.currentProject == "" {
		return nil, nil
	}

	r, err := git.PlainOpen(a.currentProject)
	if err == git.ErrRepositoryNotExists {
		return []Revision{}, nil
	}
	if err != nil {
		return nil, err
	}

	log, err := r.Log(&git.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, err
	}

	var revisions []Revision
	err = log.ForEach(func(c *object.Commit) error {
		revisions = append(revisions, Revision{
			Hash:      c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Timestamp: c.Author.When.Format(time.RFC3339),
		})
		return nil
	})

	return revisions, err
}
