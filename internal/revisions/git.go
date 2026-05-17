package revisions

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadFromGit returns the bytes of filePath as recorded at the given git ref,
// resolved against the repo enclosing filePath. The function shells out to
// `git`; it fails cleanly when:
//
//   - the file is not inside a git working tree
//   - the ref does not exist
//   - the file did not exist at that ref
//
// Callers should treat the returned error as exit-code 3 surface-level
// information; the message is suitable for printing to stderr verbatim.
func ReadFromGit(filePath, ref string) ([]byte, error) {
	if filePath == "" {
		return nil, errors.New("git: empty file path")
	}
	if ref == "" {
		return nil, errors.New("git: empty ref")
	}

	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("git: resolving absolute path of %q: %w", filePath, err)
	}
	dir := filepath.Dir(abs)

	repoRootBytes, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("git: %s is not inside a git repository (%w)", filePath, err)
	}
	repoRoot := strings.TrimSpace(string(repoRootBytes))

	rel, err := filepath.Rel(repoRoot, abs)
	if err != nil {
		return nil, fmt.Errorf("git: computing repo-relative path for %s: %w", filePath, err)
	}
	// Git always uses forward slashes inside object paths.
	rel = filepath.ToSlash(rel)

	// `git show <ref>:<relpath>` is the standard way to extract a file
	// version from a ref. It fails with a clear non-zero exit when either
	// the ref or the file is missing.
	out, err := runGit(repoRoot, "show", ref+":"+rel)
	if err != nil {
		return nil, fmt.Errorf("git: reading %s at %s: %w", rel, ref, err)
	}
	return out, nil
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Surface git's stderr in the error so users can see "fatal: ...".
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}
