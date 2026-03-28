package cmd

import (
	"github.com/jscaltreto/downstage/internal/lsp"
	"github.com/spf13/cobra"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Downstage LSP server",
	Long:  "Starts the Language Server Protocol server on stdio for editor integration.",
	RunE:  runLSP,
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

func runLSP(cmd *cobra.Command, args []string) error {
	return lsp.Start(cmd.Context())
}
