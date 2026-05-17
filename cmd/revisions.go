package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/revisions"
	"github.com/spf13/cobra"
)

var (
	revisionsAgainst       string
	revisionsFromRef       string
	revisionsOutput        string
	revisionsPageSize      string
	revisionsStyle         string
	revisionsFont          string
	revisionsMarkChanges   bool
	revisionsPageNumbering string
	revisionsAnchorWindow  int
	revisionsRemovedMarker bool
)

var revisionsCmd = &cobra.Command{
	Use:   "revisions <v2.ds>",
	Short: "Render a revision-pages PDF that highlights what changed since v1",
	Long: `Render a Hollywood-style revision-pages PDF.

Given a current ("v2") .ds file and a prior version, downstage revisions emits
a small PDF containing only the pages that changed, with right-margin
asterisks marking changed blocks and a top-margin note describing where each
page slots into the v1 print. Pages keep v1 numbering with A/B/C suffixes
where inserts overflow.

The v1 source can be supplied either as a separate file (--against) or by
reference to a git revision in the same repo (--from). v1 is rendered once
internally to compute its pagination; the resulting PDF is discarded.

Block-level granularity in v1: when a single word changes inside a long
dialogue cue, the whole cue is marked changed. Per-visual-line marking is a
planned follow-up.`,
	Args: cobra.ExactArgs(1),
	RunE: runRevisions,
}

func init() {
	revisionsCmd.Flags().StringVar(&revisionsAgainst, "against", "", "path to the v1 (prior) .ds file")
	revisionsCmd.Flags().StringVar(&revisionsFromRef, "from", "", "git ref (commit, tag, branch) to read v1 from in the v2 file's repo")
	revisionsCmd.Flags().StringVarP(&revisionsOutput, "output", "o", "", "output PDF (default: <v2-stem>.revisions.pdf)")
	revisionsCmd.Flags().StringVar(&revisionsPageSize, "page-size", "letter", "page size: letter, a4")
	revisionsCmd.Flags().StringVar(&revisionsStyle, "style", "standard", "rendering style: standard (Manuscript). Condensed is reserved for a follow-up.")
	revisionsCmd.Flags().StringVar(&revisionsFont, "font", "", "path to a custom TTF font file")
	revisionsCmd.Flags().BoolVar(&revisionsMarkChanges, "mark-changes", true, "draw a right-margin asterisk on each changed block")
	revisionsCmd.Flags().StringVar(&revisionsPageNumbering, "page-numbers", "v1-labels", "page-number style: v1-labels, natural, none")
	revisionsCmd.Flags().IntVar(&revisionsAnchorWindow, "anchor-window", 4, "merge adjacent change hunks separated by ≤ N equal blocks into one region")
	revisionsCmd.Flags().BoolVar(&revisionsRemovedMarker, "removed-marker", true, "emit a REMOVED placeholder page when a revision shortens v1's pagination")
	rootCmd.AddCommand(revisionsCmd)
}

func runRevisions(cmd *cobra.Command, args []string) error {
	v2Path := args[0]
	if (revisionsAgainst == "") == (revisionsFromRef == "") {
		return errors.New("specify exactly one of --against or --from")
	}

	v2Source, err := os.ReadFile(v2Path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", v2Path, err)
	}

	var v1Source []byte
	var v1Name string
	switch {
	case revisionsAgainst != "":
		v1Source, err = os.ReadFile(revisionsAgainst)
		if err != nil {
			return fmt.Errorf("reading %s: %w", revisionsAgainst, err)
		}
		v1Name = revisionsAgainst
	case revisionsFromRef != "":
		v1Source, err = revisions.ReadFromGit(v2Path, revisionsFromRef)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		v1Name = fmt.Sprintf("%s@%s", filepath.Base(v2Path), revisionsFromRef)
	}

	cfg := render.DefaultConfig()
	pageSize, err := render.ParsePageSize(revisionsPageSize)
	if err != nil {
		return err
	}
	cfg.PageSize = pageSize

	style, err := render.ParseStyle(revisionsStyle)
	if err != nil {
		return err
	}
	if style != render.StyleStandard {
		return fmt.Errorf("--style %q is not supported by `revisions` in v1; use --style standard", revisionsStyle)
	}
	cfg.Style = style
	cfg.FontPath = revisionsFont

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid render config: %w", err)
	}

	pageNumbering, err := parsePageNumbering(revisionsPageNumbering)
	if err != nil {
		return err
	}

	out := revisionsOutput
	if out == "" {
		stem := strings.TrimSuffix(v2Path, filepath.Ext(v2Path))
		out = stem + ".revisions.pdf"
	}

	outDir := filepath.Dir(out)
	tmp, err := os.CreateTemp(outDir, ".revisions-*.pdf")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", outDir, err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if err := revisions.Render(tmp, revisions.Options{
		V1Source:              v1Source,
		V2Source:              v2Source,
		V1Name:                v1Name,
		V2Name:                v2Path,
		Config:                cfg,
		AnchorWindow:          revisionsAnchorWindow,
		MarkChanges:           revisionsMarkChanges,
		IncludeRemovedMarkers: revisionsRemovedMarker,
		PageNumbering:         pageNumbering,
	}); err != nil {
		cleanup()
		if errors.Is(err, revisions.ErrNoDifferences) {
			fmt.Fprintln(os.Stderr, "no differences detected between v1 and v2; nothing to render")
			os.Exit(2)
		}
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, out); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, out, err)
	}
	slog.Info("revisions written", "output", out)
	return nil
}

func parsePageNumbering(s string) (revisions.PageNumberingMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "v1-labels", "":
		return revisions.PageNumberingV1Labels, nil
	case "natural":
		return revisions.PageNumberingNatural, nil
	case "none":
		return revisions.PageNumberingNone, nil
	}
	return 0, fmt.Errorf("unsupported --page-numbers: %q (want v1-labels, natural, or none)", s)
}
