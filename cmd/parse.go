package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse <file.ds>",
	Short: "Parse a .ds file and dump the AST as JSON",
	Long:  "Reads a Downstage (.ds) file, parses it, and outputs the AST as pretty-printed JSON to stdout.",
	Args:  cobra.ExactArgs(1),
	RunE:  runParse,
}

func init() {
	rootCmd.AddCommand(parseCmd)
}

func runParse(cmd *cobra.Command, args []string) error {
	filename := args[0]
	slog.Debug("parsing file", "filename", filename)

	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filename, err)
	}

	doc, errs := parser.Parse(content)

	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n",
			filepath.Base(filename),
			e.Range.Start.Line+1,
			e.Range.Start.Column+1,
			e.Message,
		)
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling AST: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}
