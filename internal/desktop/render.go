package desktop

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	htmlrender "github.com/jscaltreto/downstage/internal/render/html"
	pdfrender "github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/jscaltreto/downstage/internal/render/pdf/impose"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PdfExportOptions mirrors the frontend's ExportPdfOptions so the Wails
// bridge can forward the user's export choices end-to-end.
type PdfExportOptions struct {
	PageSize      string `json:"pageSize"`
	Style         string `json:"style"`
	Layout        string `json:"layout"`
	BookletGutter string `json:"bookletGutter"`
}

// FileFilter describes a file type filter for save dialogs.
type FileFilter struct {
	DisplayName string `json:"displayName"`
	Pattern     string `json:"pattern"`
}

func (a *App) RenderHTML(source string, style string) (string, error) {
	doc, _ := parser.Parse([]byte(source))
	cfg := render.DefaultConfig()
	cfg.SourceAnchors = true
	if style == "condensed" {
		cfg.Style = render.StyleCondensed
	}

	nr := htmlrender.NewRenderer(cfg)
	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return "", fmt.Errorf("render html: %w", err)
	}
	html := buf.String()
	slog.Debug("rendered HTML", "bytes", len(html))
	return html, nil
}

func (a *App) RenderPDF(source string, options PdfExportOptions) (string, error) {
	doc, _ := parser.Parse([]byte(source))
	cfg := render.DefaultConfig()
	if options.Style == "condensed" {
		cfg.Style = render.StyleCondensed
	}
	if options.PageSize != "" {
		ps, err := render.ParsePageSize(options.PageSize)
		if err != nil {
			return "", fmt.Errorf("pageSize: %w", err)
		}
		cfg.PageSize = ps
	}
	if options.Layout != "" {
		layout, err := render.ParsePDFLayout(options.Layout)
		if err != nil {
			return "", fmt.Errorf("layout: %w", err)
		}
		cfg.Layout = layout
	}
	if cfg.Layout == render.LayoutBooklet && options.BookletGutter != "" {
		gutterMM, err := render.ParseMeasurement(options.BookletGutter)
		if err != nil {
			return "", fmt.Errorf("gutter: %w", err)
		}
		cfg.BookletGutterMM = gutterMM
	}

	if err := cfg.Validate(); err != nil {
		return "", fmt.Errorf("validate pdf config: %w", err)
	}

	var nr render.NodeRenderer
	if cfg.Style == render.StyleCondensed {
		nr = pdfrender.NewCondensedRenderer(cfg)
	} else {
		nr = pdfrender.NewRenderer(cfg)
	}

	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return "", fmt.Errorf("render pdf: %w", err)
	}

	if cfg.Layout != render.LayoutSingle {
		sheet, err := cfg.PageSize.SheetDimensions()
		if err != nil {
			return "", fmt.Errorf("sheet dimensions: %w", err)
		}
		var imposed bytes.Buffer
		switch cfg.Layout {
		case render.Layout2Up:
			if err := impose.TwoUp(bytes.NewReader(buf.Bytes()), sheet, &imposed); err != nil {
				return "", fmt.Errorf("impose 2up: %w", err)
			}
		case render.LayoutBooklet:
			if err := impose.Booklet(bytes.NewReader(buf.Bytes()), sheet, cfg.BookletGutterMM, &imposed); err != nil {
				return "", fmt.Errorf("impose booklet: %w", err)
			}
		}
		buf = imposed
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (a *App) SaveFile(filename string, contentBase64 string, isBinary bool, filters []FileFilter) error {
	var data []byte
	var err error

	if isBinary {
		data, err = base64.StdEncoding.DecodeString(contentBase64)
		if err != nil {
			return err
		}
	} else {
		data = []byte(contentBase64)
	}

	wailsFilters := make([]runtime.FileFilter, len(filters))
	for i, f := range filters {
		wailsFilters[i] = runtime.FileFilter{
			DisplayName: f.DisplayName,
			Pattern:     f.Pattern,
		}
	}

	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: filename,
		Title:           "Save File",
		Filters:         wailsFilters,
	})
	if err != nil {
		return err
	}
	if selection == "" {
		return nil
	}

	slog.Debug("saving file", "path", selection, "bytes", len(data))
	return os.WriteFile(selection, data, 0644)
}
