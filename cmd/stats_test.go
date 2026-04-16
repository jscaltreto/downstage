package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jscaltreto/downstage/internal/stats"
)

func runStatsCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	// Reset flag-scoped state between runs.
	statsFormat = "text"
	statsRatePreset = "standard"
	statsWordsPerMin = 0
	statsPauseFactor = stats.DefaultPauseFactor
	statsPauseSet = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"stats"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestStatsCmdTextOutput(t *testing.T) {
	path := writeScript(t, `## Dramatis Personae

ALICE - A lead

## ACT I

### SCENE 1

ALICE
Hello world.
`)
	out, err := runStatsCmd(t, path)
	if err != nil {
		t.Fatalf("runStatsCmd: %v\n%s", err, out)
	}
	for _, want := range []string{"Acts:", "ALICE", "Runtime (estimate)", "standard"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q\n---\n%s", want, out)
		}
	}
}

func TestStatsCmdJSONOutput(t *testing.T) {
	path := writeScript(t, `## Dramatis Personae

ALICE - A lead

## ACT I

### SCENE 1

ALICE
Hello world.
`)
	out, err := runStatsCmd(t, "--format", "json", path)
	if err != nil {
		t.Fatalf("runStatsCmd: %v", err)
	}

	var s stats.Stats
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("unmarshal: %v\n---\n%s", err, out)
	}
	if s.Acts != 1 || s.Lines != 1 {
		t.Errorf("unexpected stats: %+v", s)
	}
	if s.Runtime.Preset != "standard" || s.Runtime.WordsPerMinute != 130 {
		t.Errorf("unexpected runtime: %+v", s.Runtime)
	}
}

func writeScript(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "script.ds")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}
