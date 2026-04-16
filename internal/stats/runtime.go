package stats

import "strings"

// Speaking rate presets in words per minute.
const (
	WPMSlow           = 110
	WPMStandard       = 130
	WPMConversational = 150

	// DefaultPauseFactor is a small overhead for pauses between speeches.
	DefaultPauseFactor = 0.10
)

// RuntimeOptions configures the runtime heuristic.
type RuntimeOptions struct {
	// Preset selects a built-in WPM band.
	Preset string
	// WordsPerMinute overrides the preset when > 0.
	WordsPerMinute int
	// PauseFactor is a multiplicative overhead applied to the runtime.
	PauseFactor    float64
	pauseFactorSet bool
}

// WithPauseFactor sets PauseFactor explicitly, including zero.
func (o RuntimeOptions) WithPauseFactor(f float64) RuntimeOptions {
	o.PauseFactor = f
	o.pauseFactorSet = true
	return o
}

// RuntimeEstimate describes a rough performance length in minutes.
type RuntimeEstimate struct {
	Preset         string  `json:"preset"`
	WordsPerMinute int     `json:"wordsPerMinute"`
	PauseFactor    float64 `json:"pauseFactor"`
	DialogueWords  int     `json:"dialogueWords"`
	Minutes        float64 `json:"minutes"`
}

// EstimateRuntime applies the documented heuristic.
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
