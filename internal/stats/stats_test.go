package stats_test

import (
	"math"
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/stats"
)

func compute(t *testing.T, src string) stats.Stats {
	t.Helper()
	doc, errs := parser.Parse([]byte(src))
	for _, e := range errs {
		t.Fatalf("parse error: %s", e.Message)
	}
	return stats.Compute(doc, stats.RuntimeOptions{})
}

func TestComputeBasicCounts(t *testing.T) {
	src := `## Dramatis Personae

ALICE - A lead
BOB - Her friend

## ACT I

### SCENE 1

> Alice enters.

ALICE
Hello world.
This is a second line.

BOB
Hi Alice.
`
	s := compute(t, src)

	if s.Acts != 1 {
		t.Errorf("Acts = %d, want 1", s.Acts)
	}
	if s.Scenes != 1 {
		t.Errorf("Scenes = %d, want 1", s.Scenes)
	}
	if s.Lines != 2 {
		t.Errorf("Lines = %d, want 2", s.Lines)
	}
	if s.DialogueWords != 9 {
		t.Errorf("DialogueWords = %d, want 9 (\"Hello world This is a second line Hi Alice\")", s.DialogueWords)
	}
	if s.StageDirections != 1 {
		t.Errorf("StageDirections = %d, want 1", s.StageDirections)
	}
	if len(s.Characters) != 2 {
		t.Fatalf("Characters = %d, want 2", len(s.Characters))
	}
	if s.Characters[0].Name != "ALICE" {
		t.Errorf("top character = %q, want ALICE", s.Characters[0].Name)
	}
	if s.Characters[0].Lines != 1 {
		t.Errorf("ALICE lines = %d, want 1", s.Characters[0].Lines)
	}
}

func TestComputeAliasFolding(t *testing.T) {
	src := `## Dramatis Personae

JAMES/JIM - Son

## ACT I

### SCENE 1

JAMES
Line one.

JIM
Line two.
`
	s := compute(t, src)
	if len(s.Characters) != 1 {
		t.Fatalf("Characters = %d, want 1 (JIM folded into JAMES)", len(s.Characters))
	}
	ch := s.Characters[0]
	if ch.Name != "JAMES" {
		t.Errorf("Name = %q, want JAMES", ch.Name)
	}
	if ch.Lines != 2 {
		t.Errorf("JAMES lines = %d, want 2", ch.Lines)
	}
	if len(ch.Aliases) != 1 || ch.Aliases[0] != "JIM" {
		t.Errorf("Aliases = %v, want [JIM]", ch.Aliases)
	}
}

func TestComputeInlineDirectionExcluded(t *testing.T) {
	src := `## Dramatis Personae

ALICE - A lead

## ACT I

### SCENE 1

ALICE
Hello (softly) world.
`
	s := compute(t, src)
	if s.DialogueWords != 2 {
		t.Errorf("DialogueWords = %d, want 2 (inline direction should be excluded)", s.DialogueWords)
	}
}

func TestComputeSongDialogue(t *testing.T) {
	src := `## Dramatis Personae

ALICE - A lead

## ACT I

### SCENE 1

SONG: Test

ALICE
  A song line here.

SONG END
`
	s := compute(t, src)
	if s.Songs != 1 {
		t.Errorf("Songs = %d, want 1", s.Songs)
	}
	if s.Lines != 1 {
		t.Errorf("Lines = %d, want 1 (dialogue inside song counts)", s.Lines)
	}
	if s.DialogueWords != 4 {
		t.Errorf("DialogueWords = %d, want 4", s.DialogueWords)
	}
}

func TestComputeGenericSectionProseCounted(t *testing.T) {
	src := `# Playwright's Notes

This play was written over four months.
It draws on several conversations with retired actors.
`
	s := compute(t, src)
	if s.TotalWords != 15 {
		t.Errorf("TotalWords = %d, want 15 (generic section prose must count)", s.TotalWords)
	}
}

func TestComputeEmptyDocument(t *testing.T) {
	s := stats.Compute(nil, stats.RuntimeOptions{})
	if s.TotalWords != 0 || s.Runtime.Minutes != 0 {
		t.Errorf("empty doc should produce zeros, got %+v", s)
	}
}

func TestRuntimePresets(t *testing.T) {
	cases := []struct {
		preset  string
		wpm     int
		pause   float64
		setZero bool
		words   int
		want    float64
	}{
		{"standard", 130, stats.DefaultPauseFactor, false, 1300, 1300.0 / 130 * 1.10},
		{"slow", 110, stats.DefaultPauseFactor, false, 1100, 1100.0 / 110 * 1.10},
		{"conversational", 150, stats.DefaultPauseFactor, false, 1500, 1500.0 / 150 * 1.10},
		{"", 130, stats.DefaultPauseFactor, false, 130, 1.10},
		{"unknown", 130, stats.DefaultPauseFactor, false, 130, 1.10},
	}
	for _, tc := range cases {
		opts := stats.RuntimeOptions{Preset: tc.preset}
		got := stats.EstimateRuntime(tc.words, opts)
		if got.WordsPerMinute != tc.wpm {
			t.Errorf("preset %q wpm = %d, want %d", tc.preset, got.WordsPerMinute, tc.wpm)
		}
		if math.Abs(got.Minutes-tc.want) > 1e-6 {
			t.Errorf("preset %q minutes = %v, want %v", tc.preset, got.Minutes, tc.want)
		}
	}
}

func TestRuntimeOverrides(t *testing.T) {
	opts := stats.RuntimeOptions{WordsPerMinute: 200}
	opts = opts.WithPauseFactor(0)
	got := stats.EstimateRuntime(400, opts)
	if got.WordsPerMinute != 200 {
		t.Errorf("wpm = %d, want 200", got.WordsPerMinute)
	}
	if got.PauseFactor != 0 {
		t.Errorf("pauseFactor = %v, want 0 (explicit zero)", got.PauseFactor)
	}
	if math.Abs(got.Minutes-2.0) > 1e-6 {
		t.Errorf("minutes = %v, want 2.0", got.Minutes)
	}
	if got.Preset != "custom" {
		t.Errorf("preset = %q, want custom", got.Preset)
	}
}

func TestRuntimeWPMOverrideReportsCustom(t *testing.T) {
	got := stats.EstimateRuntime(260, stats.RuntimeOptions{
		Preset:         "standard",
		WordsPerMinute: 200,
	})
	if got.Preset != "custom" {
		t.Errorf("preset = %q, want custom when --wpm overrides", got.Preset)
	}
	if got.WordsPerMinute != 200 {
		t.Errorf("wpm = %d, want 200", got.WordsPerMinute)
	}
}

func TestRuntimeNegativePauseClamped(t *testing.T) {
	opts := stats.RuntimeOptions{}.WithPauseFactor(-0.5)
	got := stats.EstimateRuntime(130, opts)
	if got.PauseFactor != 0 {
		t.Errorf("pauseFactor = %v, want 0 (negative clamped)", got.PauseFactor)
	}
	if math.Abs(got.Minutes-1.0) > 1e-6 {
		t.Errorf("minutes = %v, want 1.0", got.Minutes)
	}
}
