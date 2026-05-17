package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRevisions_GeneratesPDF(t *testing.T) {
	dir := t.TempDir()
	v1Path := filepath.Join(dir, "v1.ds")
	v2Path := filepath.Join(dir, "v2.ds")
	outPath := filepath.Join(dir, "rev.pdf")

	require.NoError(t, os.WriteFile(v1Path, []byte("# Play\n\n## ACT I\n\nHAMLET\nA.\n"), 0o644))
	require.NoError(t, os.WriteFile(v2Path, []byte("# Play\n\n## ACT I\n\nHAMLET\nA.\n\nHAMLET\nB.\n"), 0o644))

	resetRevisionFlags()
	revisionsAgainst = v1Path
	revisionsOutput = outPath
	revisionsPageSize = "letter"
	revisionsStyle = "standard"
	revisionsPageNumbering = "v1-labels"
	revisionsMarkChanges = true
	revisionsRemovedMarker = true
	revisionsAnchorWindow = 4

	err := runRevisions(revisionsCmd, []string{v2Path})
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), "%PDF-"))
}

func TestRunRevisions_RejectsBothInputModes(t *testing.T) {
	dir := t.TempDir()
	v1 := filepath.Join(dir, "v1.ds")
	v2 := filepath.Join(dir, "v2.ds")
	require.NoError(t, os.WriteFile(v1, []byte("# P\n"), 0o644))
	require.NoError(t, os.WriteFile(v2, []byte("# P\n"), 0o644))

	resetRevisionFlags()
	revisionsAgainst = v1
	revisionsFromRef = "HEAD"

	err := runRevisions(revisionsCmd, []string{v2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestRunRevisions_RejectsNeitherInputMode(t *testing.T) {
	dir := t.TempDir()
	v2 := filepath.Join(dir, "v2.ds")
	require.NoError(t, os.WriteFile(v2, []byte("# P\n"), 0o644))

	resetRevisionFlags()

	err := runRevisions(revisionsCmd, []string{v2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestRunRevisions_CondensedRejected(t *testing.T) {
	dir := t.TempDir()
	v1 := filepath.Join(dir, "v1.ds")
	v2 := filepath.Join(dir, "v2.ds")
	require.NoError(t, os.WriteFile(v1, []byte("# P\n"), 0o644))
	require.NoError(t, os.WriteFile(v2, []byte("# P\n\nHAMLET\nLine.\n"), 0o644))

	resetRevisionFlags()
	revisionsAgainst = v1
	revisionsOutput = filepath.Join(dir, "out.pdf")
	revisionsStyle = "condensed"

	err := runRevisions(revisionsCmd, []string{v2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "condensed")
}

func TestParsePageNumbering(t *testing.T) {
	cases := map[string]bool{
		"v1-labels": true,
		"natural":   true,
		"none":      true,
		"":          true,
		"weird":     false,
	}
	for in, ok := range cases {
		_, err := parsePageNumbering(in)
		if ok {
			assert.NoError(t, err, in)
		} else {
			assert.Error(t, err, in)
		}
	}
}

func resetRevisionFlags() {
	revisionsAgainst = ""
	revisionsFromRef = ""
	revisionsOutput = ""
	revisionsPageSize = "letter"
	revisionsStyle = "standard"
	revisionsFont = ""
	revisionsMarkChanges = true
	revisionsPageNumbering = "v1-labels"
	revisionsAnchorWindow = 4
	revisionsRemovedMarker = true
}

func TestRunRevisions_RenderFailureLeavesNoOutputFile(t *testing.T) {
	dir := t.TempDir()
	v1 := filepath.Join(dir, "v1.ds")
	v2 := filepath.Join(dir, "v2.ds")
	require.NoError(t, os.WriteFile(v1, []byte("# Play\n\n## ACT I\n\nHAMLET\nFirst.\n"), 0o644))
	require.NoError(t, os.WriteFile(v2, []byte("# Play\n\n## ACT I\n\nSONG 1: Test\n\nHAMLET\nNo SONG END.\n"), 0o644))

	out := filepath.Join(dir, "rev.pdf")
	resetRevisionFlags()
	revisionsAgainst = v1
	revisionsOutput = out

	err := runRevisions(revisionsCmd, []string{v2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SONG")

	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr),
		"no PDF should remain at the destination path on render failure (got: %v)", statErr)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".revisions-"))
	}
}
