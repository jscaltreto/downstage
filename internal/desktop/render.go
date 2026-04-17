package desktop

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"os"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	htmlrender "github.com/jscaltreto/downstage/internal/render/html"
	pdfrender "github.com/jscaltreto/downstage/internal/render/pdf"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileFilter describes a file type filter for save dialogs.
type FileFilter struct {
	DisplayName string `json:"displayName"`
	Pattern     string `json:"pattern"`
}

func (a *App) RenderHTML(source string, style string) string {
	doc, _ := parser.Parse([]byte(source))
	cfg := render.DefaultConfig()
	cfg.SourceAnchors = true
	if style == "condensed" {
		cfg.Style = render.StyleCondensed
	}

	nr := htmlrender.NewRenderer(cfg)
	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return err.Error()
	}
	html := buf.String()
	slog.Debug("rendered HTML", "bytes", len(html))
	return html
}

func (a *App) RenderPDF(source string, style string) string {
	doc, _ := parser.Parse([]byte(source))
	cfg := render.DefaultConfig()
	if style == "condensed" {
		cfg.Style = render.StyleCondensed
	}

	var nr render.NodeRenderer
	if cfg.Style == render.StyleCondensed {
		nr = pdfrender.NewCondensedRenderer(cfg)
	} else {
		nr = pdfrender.NewRenderer(cfg)
	}

	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
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
