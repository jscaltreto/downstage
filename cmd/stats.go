package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/stats"
	"github.com/spf13/cobra"
)

var (
	statsFormat      string
	statsRatePreset  string
	statsWordsPerMin int
	statsPauseFactor float64
	statsPauseSet    bool
)

var statsCmd = &cobra.Command{
	Use:   "stats <file.ds>",
	Short: "Report manuscript statistics for a .ds file",
	Long: `Parses a Downstage (.ds) file and reports word counts, per-character
dialogue tallies, scene/act counts, and a rough runtime estimate.

The runtime estimate is derived from dialogue word count alone. Choose a
preset with --rate (slow, standard, conversational) or override the rate
directly with --wpm.`,
	Args: cobra.ExactArgs(1),
	RunE: runStats,
}

func init() {
	statsCmd.Flags().StringVarP(&statsFormat, "format", "f", "text", "output format: text or json")
	statsCmd.Flags().StringVar(&statsRatePreset, "rate", "standard", "speaking rate preset: slow, standard, conversational")
	statsCmd.Flags().IntVar(&statsWordsPerMin, "wpm", 0, "override speaking rate in words per minute (disables --rate preset)")
	statsCmd.Flags().Float64Var(&statsPauseFactor, "pause", stats.DefaultPauseFactor, "pause overhead applied to spoken runtime (e.g. 0.10 for 10%)")
	statsCmd.PreRun = func(cmd *cobra.Command, args []string) {
		statsPauseSet = cmd.Flags().Changed("pause")
	}
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	filename := args[0]
	slog.Debug("computing stats", "filename", filename)

	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filename, err)
	}

	doc, errs := parser.Parse(content)
	for _, e := range errs {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s:%d:%d: %s\n",
			filepath.Base(filename),
			e.Range.Start.Line+1,
			e.Range.Start.Column+1,
			e.Message,
		)
	}

	opts := stats.RuntimeOptions{
		Preset:         statsRatePreset,
		WordsPerMinute: statsWordsPerMin,
		PauseFactor:    statsPauseFactor,
	}
	if statsPauseSet {
		opts = opts.WithPauseFactor(statsPauseFactor)
	}
	result := stats.Compute(doc, opts)

	switch statsFormat {
	case "json":
		return writeStatsJSON(cmd.OutOrStdout(), result)
	case "text", "":
		return writeStatsText(cmd.OutOrStdout(), result)
	default:
		return errors.New("unknown --format: use text or json")
	}
}

func writeStatsJSON(w io.Writer, s stats.Stats) error {
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling stats: %w", err)
	}
	fmt.Fprintln(w, string(out))
	return nil
}

func writeStatsText(w io.Writer, s stats.Stats) error {
	fmt.Fprintln(w, "Manuscript")
	fmt.Fprintf(w, "  Acts:             %d\n", s.Acts)
	fmt.Fprintf(w, "  Scenes:           %d\n", s.Scenes)
	fmt.Fprintf(w, "  Songs:            %d\n", s.Songs)
	fmt.Fprintf(w, "  Speeches:         %d\n", s.Speeches)
	fmt.Fprintf(w, "  Stage directions: %d\n", s.StageDirections)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Words")
	fmt.Fprintf(w, "  Total:            %d\n", s.TotalWords)
	fmt.Fprintf(w, "  Dialogue:         %d (%s)\n", s.DialogueWords, percent(s.DialogueWords, s.TotalWords))
	fmt.Fprintf(w, "  Stage directions: %d\n", s.StageDirectionWords)
	fmt.Fprintf(w, "  Dialogue lines:   %d\n", s.DialogueLines)
	fmt.Fprintln(w)

	if len(s.Characters) > 0 {
		fmt.Fprintln(w, "Characters (by speeches)")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tSPEECHES\tLINES\tWORDS")
		for _, c := range s.Characters {
			fmt.Fprintf(tw, "  %s\t%d\t%d\t%d\n", c.Name, c.Speeches, c.DialogueLines, c.DialogueWords)
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Runtime (estimate)")
	fmt.Fprintf(w, "  Preset:           %s\n", s.Runtime.Preset)
	fmt.Fprintf(w, "  Rate:             %d wpm\n", s.Runtime.WordsPerMinute)
	fmt.Fprintf(w, "  Pause factor:     %.0f%%\n", s.Runtime.PauseFactor*100)
	fmt.Fprintf(w, "  Estimated length: %s (~%.1f min)\n", formatRuntime(s.Runtime.Minutes), s.Runtime.Minutes)
	fmt.Fprintln(w, "  Based on spoken dialogue only; treat as a rough guide.")
	return nil
}

func percent(part, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(part)/float64(total)*100)
}

func formatRuntime(minutes float64) string {
	if minutes <= 0 {
		return "0m"
	}
	total := int(math.Round(minutes))
	h := total / 60
	m := total % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", h, m)
}
