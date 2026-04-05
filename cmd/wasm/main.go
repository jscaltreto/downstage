//go:build js && wasm

package main

import (
	"bytes"
	"encoding/json"
	"syscall/js"

	"github.com/jscaltreto/downstage/internal/lsp"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/render"
	htmlrender "github.com/jscaltreto/downstage/internal/render/html"
	pdfrender "github.com/jscaltreto/downstage/internal/render/pdf"
	"go.lsp.dev/protocol"
)

func main() {
	ds := js.Global().Get("Object").New()

	ds.Set("parse", js.FuncOf(parse))
	ds.Set("diagnostics", js.FuncOf(diagnostics))
	ds.Set("renderHTML", js.FuncOf(renderHTML))
	ds.Set("renderPDF", js.FuncOf(renderPDF))
	ds.Set("semanticTokens", js.FuncOf(semanticTokens))
	ds.Set("tokenTypeNames", tokenTypeNamesArray())

	js.Global().Set("downstage", ds)

	// Keep the Go runtime alive.
	select {}
}

type parseErrorJSON struct {
	Message string `json:"message"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	EndLine int    `json:"endLine"`
	EndCol  int    `json:"endCol"`
}

type diagnosticJSON struct {
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	EndLine  int    `json:"endLine"`
	EndCol   int    `json:"endCol"`
	Code     string `json:"code,omitempty"`
}

func parse(_ js.Value, args []js.Value) any {
	source := args[0].String()
	_, errs := parser.Parse([]byte(source))

	out := make([]parseErrorJSON, len(errs))
	for i, e := range errs {
		out[i] = parseErrorJSON{
			Message: e.Message,
			Line:    e.Range.Start.Line,
			Col:     e.Range.Start.Column,
			EndLine: e.Range.End.Line,
			EndCol:  e.Range.End.Column,
		}
	}

	data, _ := json.Marshal(map[string]any{"errors": out})
	return js.Global().Get("JSON").Call("parse", string(data))
}

func diagnostics(_ js.Value, args []js.Value) any {
	source := args[0].String()
	doc, errs := parser.Parse([]byte(source))
	diags := lsp.ComputeDiagnostics(doc, errs)

	out := make([]diagnosticJSON, len(diags))
	for i, diag := range diags {
		out[i] = diagnosticJSON{
			Message:  diag.Message,
			Severity: diagnosticSeverity(diag.Severity),
			Line:     int(diag.Range.Start.Line),
			Col:      int(diag.Range.Start.Character),
			EndLine:  int(diag.Range.End.Line),
			EndCol:   int(diag.Range.End.Character),
			Code:     diagnosticCode(diag.Code),
		}
	}

	data, _ := json.Marshal(map[string]any{"diagnostics": out})
	return js.Global().Get("JSON").Call("parse", string(data))
}

func renderHTML(_ js.Value, args []js.Value) any {
	source := args[0].String()
	doc, _ := parser.Parse([]byte(source))

	cfg := render.DefaultConfig()
	cfg.SourceAnchors = true
	if len(args) > 1 && args[1].String() == "condensed" {
		cfg.Style = render.StyleCondensed
	}

	nr := htmlrender.NewRenderer(cfg)
	var buf bytes.Buffer
	if err := render.Walk(nr, doc, &buf); err != nil {
		return err.Error()
	}
	return buf.String()
}

func renderPDF(_ js.Value, args []js.Value) any {
	source := args[0].String()
	doc, _ := parser.Parse([]byte(source))

	cfg := render.DefaultConfig()
	if len(args) > 1 && args[1].String() == "condensed" {
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
		return js.Null()
	}

	data := buf.Bytes()
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	return arr
}

func semanticTokens(_ js.Value, args []js.Value) any {
	source := args[0].String()
	doc, errs := parser.Parse([]byte(source))
	tokens := lsp.ComputeSemanticTokens(doc, errs)

	arr := js.Global().Get("Uint32Array").New(len(tokens))
	for i, v := range tokens {
		arr.SetIndex(i, v)
	}
	return arr
}

func tokenTypeNamesArray() js.Value {
	names := lsp.SemanticTokenTypeNames
	arr := js.Global().Get("Array").New(len(names))
	for i, name := range names {
		arr.SetIndex(i, name)
	}
	return arr
}

func diagnosticSeverity(severity protocol.DiagnosticSeverity) string {
	switch severity {
	case protocol.DiagnosticSeverityError:
		return "error"
	case protocol.DiagnosticSeverityWarning:
		return "warning"
	case protocol.DiagnosticSeverityInformation:
		return "info"
	default:
		return "error"
	}
}

func diagnosticCode(code any) string {
	switch v := code.(type) {
	case string:
		return v
	default:
		return ""
	}
}
