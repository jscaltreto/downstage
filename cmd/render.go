package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	htmlrender "github.com/jscaltreto/downstage/internal/render/html"
	"github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/jscaltreto/downstage/internal/render/pdf/impose"
	"github.com/spf13/cobra"
)

var (
	renderFormat        string
	renderOutput        string
	renderPageSize      string
	renderStyle         string
	renderLayout        string
	renderGutter        string
	renderFont          string
	renderStdin         bool
	renderStdout        bool
	renderSourceName    string
	renderSourceAnchors bool
)

var renderCmd = &cobra.Command{
	Use:   "render <file.ds>",
	Short: "Render a .ds file to PDF or other formats",
	Long:  "Reads a Downstage (.ds) file, parses it, and renders it to the specified output format.",
	Args:  cobra.ArbitraryArgs,
	RunE:  runRender,
}

func init() {
	renderCmd.Flags().StringVarP(&renderFormat, "format", "f", "pdf", "output format: pdf, html")
	renderCmd.Flags().StringVarP(&renderOutput, "output", "o", "", "output file (default: input name with format extension)")
	renderCmd.Flags().StringVar(&renderPageSize, "page-size", "letter", "page size: letter, a4")
	renderCmd.Flags().StringVar(&renderStyle, "style", "standard", "rendering style: standard (Manuscript), condensed (Acting Edition)")
	renderCmd.Flags().StringVar(&renderLayout, "pdf-layout", "single", "PDF layout: single, 2up, booklet (2up and booklet are condensed-only)")
	renderCmd.Flags().StringVar(&renderGutter, "gutter", defaultGutter, "booklet inner gutter (e.g. 0.125in or 3mm); only valid with --pdf-layout booklet")
	renderCmd.Flags().StringVar(&renderFont, "font", "", "path to a custom TTF font file")
	renderCmd.Flags().BoolVar(&renderStdin, "stdin", false, "read input from stdin instead of a file")
	renderCmd.Flags().BoolVar(&renderStdout, "stdout", false, "write output to stdout instead of a file")
	renderCmd.Flags().StringVar(&renderSourceName, "source-name", "<stdin>", "source filename for diagnostics when using --stdin")
	renderCmd.Flags().BoolVar(&renderSourceAnchors, "source-anchors", false, "emit source line anchors in HTML output")
	rootCmd.AddCommand(renderCmd)
}

func runRender(cmd *cobra.Command, args []string) error {
	if renderStdin && len(args) > 0 {
		return fmt.Errorf("--stdin does not accept file arguments")
	}
	if !renderStdin && len(args) != 1 {
		return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
	}
	if renderStdin && !renderStdout && renderOutput == "" {
		return fmt.Errorf("--stdin requires --stdout or --output")
	}

	var filename string
	var content []byte

	if renderStdin {
		filename = renderSourceName
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		content = data
	} else {
		filename = args[0]
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("reading %s: %w", filename, err)
		}
		content = data
	}

	slog.Debug("rendering file", "filename", filename)

	doc, errs := parser.Parse(content)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n",
			filepath.Base(filename),
			e.Range.Start.Line+1,
			e.Range.Start.Column+1,
			e.Message,
		)
	}
	if len(errs) > 0 {
		return fmt.Errorf("parse failed with %d error(s)", len(errs))
	}

	cfg := render.DefaultConfig()

	style, err := render.ParseStyle(renderStyle)
	if err != nil {
		return err
	}
	cfg.Style = style
	cfg.SourceAnchors = renderSourceAnchors

	layout, err := render.ParsePDFLayout(renderLayout)
	if err != nil {
		return err
	}

	var nr render.NodeRenderer
	switch renderFormat {
	case "pdf":
		pageSize, err := render.ParsePageSize(renderPageSize)
		if err != nil {
			return err
		}
		cfg.PageSize = pageSize
		cfg.Layout = layout
		cfg.FontPath = renderFont

		if layout == render.LayoutBooklet {
			gutterMM, err := render.ParseMeasurement(renderGutter)
			if err != nil {
				return fmt.Errorf("--gutter: %w", err)
			}
			cfg.BookletGutterMM = gutterMM
		} else if gutterExplicitlySet(cmd) {
			return fmt.Errorf("--gutter is only valid with --pdf-layout booklet")
		}

		switch cfg.Style {
		case render.StyleCondensed:
			nr = pdf.NewCondensedRenderer(cfg)
		default:
			nr = pdf.NewRenderer(cfg)
		}
	case "html":
		if renderPageSize != "letter" {
			return fmt.Errorf("--page-size is only supported for pdf output")
		}
		if layout != render.LayoutSingle {
			return fmt.Errorf("--pdf-layout is only supported for pdf output")
		}
		if gutterExplicitlySet(cmd) {
			return fmt.Errorf("--gutter is only supported for pdf output")
		}
		if renderFont != "" {
			return fmt.Errorf("--font is only supported for pdf output")
		}
		nr = htmlrender.NewRenderer(cfg)
	default:
		return fmt.Errorf("unsupported format: %q", renderFormat)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid render config: %w", err)
	}

	writer, closer, err := openOutput(filename)
	if err != nil {
		return err
	}
	defer closer()

	if err := renderTo(writer, nr, doc, cfg); err != nil {
		return fmt.Errorf("rendering: %w", err)
	}

	if renderStdout {
		return nil
	}
	slog.Info("rendered", "output", renderOutput, "format", renderFormat)
	return nil
}

// defaultGutter matches the flag default; anything else is treated as an
// explicit user override. Tests invoke runRender with ad-hoc cobra commands
// that do not carry flag-changed metadata, so we fall back to a value check.
const defaultGutter = "0.125in"

func gutterExplicitlySet(cmd *cobra.Command) bool {
	if flag := cmd.Flags().Lookup("gutter"); flag != nil && flag.Changed {
		return true
	}
	return renderGutter != defaultGutter
}

func openOutput(filename string) (io.Writer, func(), error) {
	if renderStdout {
		return os.Stdout, func() {}, nil
	}
	if renderOutput == "" {
		renderOutput = strings.TrimSuffix(filename, filepath.Ext(filename)) + "." + renderFormat
	}
	f, err := os.Create(renderOutput)
	if err != nil {
		return nil, nil, fmt.Errorf("creating output file: %w", err)
	}
	return f, func() { f.Close() }, nil
}

// renderTo writes the rendered document to w. For non-single PDF layouts it
// renders into an in-memory buffer and post-processes through the impose
// package to compose 2-up or booklet sheets.
func renderTo(w io.Writer, nr render.NodeRenderer, doc *ast.Document, cfg render.Config) error {
	if cfg.Layout == render.LayoutSingle {
		return render.Walk(nr, doc, w)
	}

	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return err
	}

	sheet, err := cfg.PageSize.SheetDimensions()
	if err != nil {
		return err
	}

	switch cfg.Layout {
	case render.Layout2Up:
		return impose.TwoUp(bytes.NewReader(buf.Bytes()), sheet, w)
	case render.LayoutBooklet:
		return impose.Booklet(bytes.NewReader(buf.Bytes()), sheet, cfg.BookletGutterMM, w)
	default:
		return fmt.Errorf("unsupported layout %q", cfg.Layout)
	}
}
