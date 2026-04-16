package stats

import "strings"

// Speaking rate presets in words per minute. These are coarse bands meant
// to bracket likely pacing; they are not performance predictions. See
// docs/stats.md for the rationale.
const (
	WPMSlow           = 110
	WPMStandard       = 130
	WPMConversational = 150

	// DefaultPauseFactor accounts for short transitions, breaths, and
	// beats between speeches that spoken-word timing alone does not
	// capture. Kept intentionally small so the estimate stays anchored to
	// the dialogue word count.
	DefaultPauseFactor = 0.10
)

// RuntimeOptions configures the runtime heuristic. Zero values fall back
// to the "standard" preset and DefaultPauseFactor.
type RuntimeOptions struct {
	// Preset selects a built-in WPM band: "slow", "standard",
	// "conversational", or "" (standard). Ignored when WordsPerMinute is
	// set to a non-zero value.
	Preset string
	// WordsPerMinute overrides the preset when > 0.
	WordsPerMinute int
	// PauseFactor is a multiplicative overhead applied to the spoken-word
	// runtime. A negative value is treated as zero.
	PauseFactor float64
	// pauseFactorSet disambiguates "omitted" (use default) from
	// "explicitly zero" when PauseFactor == 0.
	pauseFactorSet bool
}

// WithPauseFactor returns opts with PauseFactor explicitly set, including
// zero. Use this when a caller wants to disable the pause adjustment.
func (o RuntimeOptions) WithPauseFactor(f float64) RuntimeOptions {
	o.PauseFactor = f
	o.pauseFactorSet = true
	return o
}

// RuntimeEstimate describes a rough performance length in minutes. It is
// derived from dialogue word count and is explicitly an estimate, not a
// prediction.
type RuntimeEstimate struct {
	Preset         string  `json:"preset"`
	WordsPerMinute int     `json:"wordsPerMinute"`
	PauseFactor    float64 `json:"pauseFactor"`
	DialogueWords  int     `json:"dialogueWords"`
	Minutes        float64 `json:"minutes"`
}

// EstimateRuntime applies the documented heuristic:
//
//	spoken = dialogueWords / wordsPerMinute
//	minutes = spoken * (1 + pauseFactor)
//
// Only spoken dialogue contributes; stage directions and prose do not.
func EstimateRuntime(dialogueWords int, opts RuntimeOptions) RuntimeEstimate {
	preset, wpm := resolvePreset(opts)
	pause := opts.PauseFactor
	if !opts.pauseFactorSet && pause == 0 {
		pause = DefaultPauseFactor
	}
	if pause < 0 {
		pause = 0
	}
	minutes := 0.0
	if dialogueWords > 0 && wpm > 0 {
		minutes = float64(dialogueWords) / float64(wpm) * (1 + pause)
	}
	return RuntimeEstimate{
		Preset:         preset,
		WordsPerMinute: wpm,
		PauseFactor:    pause,
		DialogueWords:  dialogueWords,
		Minutes:        minutes,
	}
}

func resolvePreset(opts RuntimeOptions) (string, int) {
	if opts.WordsPerMinute > 0 {
		// A non-zero WordsPerMinute is always an explicit override; report
		// it as "custom" so the surfaced preset matches the rate in use.
		return "custom", opts.WordsPerMinute
	}
	switch strings.ToLower(strings.TrimSpace(opts.Preset)) {
	case "slow":
		return "slow", WPMSlow
	case "conversational":
		return "conversational", WPMConversational
	case "", "standard":
		return "standard", WPMStandard
	default:
		return "standard", WPMStandard
	}
}
