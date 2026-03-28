package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadTestFile reads a file from the given path.
func LoadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read test file: %s", path)
	return data
}

// LoadGoldenJSON reads and unmarshals a golden JSON file into a generic map.
func LoadGoldenJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data := LoadTestFile(t, path)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result), "failed to unmarshal golden file: %s", path)
	return result
}

// UpdateGoldenFile writes data to a golden file as indented JSON.
func UpdateGoldenFile(t *testing.T, path string, data any) {
	t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	bytes = append(bytes, '\n')
	require.NoError(t, os.WriteFile(path, bytes, 0644))
}

// GoldenTest runs a golden file test. It compares actual output against the
// expected golden file. If the UPDATE_GOLDEN env var is set to "1", it updates
// the golden file instead of comparing.
func GoldenTest(t *testing.T, goldenPath string, actual any) {
	t.Helper()

	actualBytes, err := json.MarshalIndent(actual, "", "  ")
	require.NoError(t, err)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		UpdateGoldenFile(t, goldenPath, actual)
		return
	}

	expected := LoadTestFile(t, goldenPath)
	assert.JSONEq(t, string(expected), string(actualBytes))
}

// TestDataDir returns the absolute path to a testdata directory relative to
// the given path.
func TestDataDir(t *testing.T, relPath string) string {
	t.Helper()
	abs, err := filepath.Abs(relPath)
	require.NoError(t, err)
	return abs
}
