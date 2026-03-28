package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file.ds>",
	Short: "Validate a .ds file and report warnings and errors",
	Long:  "Reads a Downstage (.ds) file, parses it, and outputs any warnings or errors in a human-readable format.",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	filename := args[0]
	slog.Debug("validating file", "filename", filename)

	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filename, err)
	}

	_, errs := parser.Parse(content)

	base := filepath.Base(filename)
	hasErrors := false

	for _, e := range errs {
		fmt.Fprintf(cmd.OutOrStdout(), "%s:%d:%d: error: %s\n",
			base,
			e.Range.Start.Line+1,
			e.Range.Start.Column+1,
			e.Message,
		)
		hasErrors = true
	}

	if len(errs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no issues found")
	}

	if hasErrors {
		return errors.New("validation failed")
	}

	return nil
}
