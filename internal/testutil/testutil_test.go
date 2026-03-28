package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTestFile(t *testing.T) {
	data := LoadTestFile(t, filepath.Join("testdata", "sample.txt"))
	assert.Contains(t, string(data), "sample test file")
	assert.Contains(t, string(data), "multiple lines")
}

func TestLoadTestFile_Missing(t *testing.T) {
	// Verify that loading a nonexistent file would fail.
	// We can't easily test require.NoError causing FailNow in a sub-test,
	// so we just verify the file doesn't exist.
	_, err := os.ReadFile(filepath.Join("testdata", "nonexistent.txt"))
	assert.Error(t, err)
}

func TestLoadGoldenJSON(t *testing.T) {
	result := LoadGoldenJSON(t, filepath.Join("testdata", "sample.golden.json"))
	assert.Equal(t, "sample", result["name"])
	assert.Equal(t, float64(3), result["count"]) // JSON numbers are float64
	assert.Equal(t, true, result["valid"])
}

func TestGoldenTest(t *testing.T) {
	actual := map[string]any{
		"name":  "sample",
		"count": 3,
		"valid": true,
	}
	GoldenTest(t, filepath.Join("testdata", "sample.golden.json"), actual)
}

func TestGoldenTest_Update(t *testing.T) {
	// Test the update path by writing to a temp file.
	tmpDir := t.TempDir()
	goldenPath := filepath.Join(tmpDir, "update.golden.json")

	actual := map[string]any{
		"key": "value",
	}

	t.Setenv("UPDATE_GOLDEN", "1")
	GoldenTest(t, goldenPath, actual)

	// Verify the file was written.
	data, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"key"`)
	assert.Contains(t, string(data), `"value"`)
}

func TestTestDataDir(t *testing.T) {
	dir := TestDataDir(t, "testdata")
	assert.True(t, filepath.IsAbs(dir))
	assert.Contains(t, dir, "testdata")
}
