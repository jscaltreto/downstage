//go:build js && wasm

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/jscaltreto/downstage/internal/lsp"
	"github.com/jscaltreto/downstage/internal/migrate"
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
	ds.Set("spellcheckContext", js.FuncOf(spellcheckContext))
	ds.Set("upgradeV1", js.FuncOf(upgradeV1))
	ds.Set("completion", js.FuncOf(completion))
	ds.Set("codeActions", js.FuncOf(codeActions))
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

const codeActionsURI protocol.DocumentURI = "inmemory://document.ds"

type diagnosticJSON struct {
	Message    string   `json:"message"`
	Severity   string   `json:"severity"`
	Line       int      `json:"line"`
	Col        int      `json:"col"`
	EndLine    int      `json:"endLine"`
	EndCol     int      `json:"endCol"`
	Code       string   `json:"code,omitempty"`
	QuickFixes []string `json:"quickFixes,omitempty"`
}

type spellcheckContextJSON struct {
	AllowWords    []string         `json:"allowWords"`
	IgnoredRanges []protocol.Range `json:"ignoredRanges"`
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

		actions := lsp.ComputeCodeActions(doc, source, codeActionsURI, []protocol.Diagnostic{diag}, diags)
		titles := actionTitles(actions)
		if len(titles) > 0 {
			out[i].QuickFixes = titles
		}
	}

	data, _ := json.Marshal(map[string]any{"diagnostics": out})
	return js.Global().Get("JSON").Call("parse", string(data))
}

func spellcheckContext(_ js.Value, args []js.Value) any {
	source := args[0].String()
	doc, errs := parser.Parse([]byte(source))
	ctx := lsp.ComputeSpellcheckContext(doc, errs)

	data, _ := json.Marshal(spellcheckContextJSON{
		AllowWords:    ctx.AllowWords,
		IgnoredRanges: ctx.IgnoredRanges,
	})
	return js.Global().Get("JSON").Call("parse", string(data))
}

func actionTitles(actions []protocol.CodeAction) []string {
	titles := make([]string, 0, len(actions))
	for _, a := range actions {
		if a.Edit == nil {
			continue
		}
		if edits := a.Edit.Changes[codeActionsURI]; len(edits) == 0 {
			continue
		}
		titles = append(titles, a.Title)
	}
	return titles
}

func completion(_ js.Value, args []js.Value) any {
	source := args[0].String()
	line := args[1].Int()
	col := args[2].Int()

	doc, errs := parser.Parse([]byte(source))
	list := lsp.ComputeCompletion(doc, errs, source, protocol.Position{
		Line:      uint32(line),
		Character: uint32(col),
	})
	if list == nil {
		list = &protocol.CompletionList{Items: []protocol.CompletionItem{}}
	}

	data, _ := json.Marshal(list)
	return js.Global().Get("JSON").Call("parse", string(data))
}

func codeActions(_ js.Value, args []js.Value) any {
	source := args[0].String()
	line := args[1].Int()
	col := args[2].Int()

	var codeFilter map[string]struct{}
	if len(args) > 3 && isJSArray(args[3]) {
		codeFilter = make(map[string]struct{})
		length := args[3].Length()
		for i := 0; i < length; i++ {
			codeFilter[args[3].Index(i).String()] = struct{}{}
		}
	}

	doc, errs := parser.Parse([]byte(source))
	allDiags := lsp.ComputeDiagnostics(doc, errs)

	var ctxDiags []protocol.Diagnostic
	for _, d := range allDiags {
		if int(d.Range.Start.Line) != line {
			continue
		}
		if col < int(d.Range.Start.Character) || col > int(d.Range.End.Character) {
			continue
		}
		if codeFilter != nil {
			code, _ := d.Code.(string)
			if _, ok := codeFilter[code]; !ok {
				continue
			}
		}
		ctxDiags = append(ctxDiags, d)
	}

	actions := lsp.ComputeCodeActions(doc, source, codeActionsURI, ctxDiags, allDiags)
	if actions == nil {
		actions = []protocol.CodeAction{}
	}

	data, _ := json.Marshal(map[string]any{
		"uri":     string(codeActionsURI),
		"actions": actions,
	})
	return js.Global().Get("JSON").Call("parse", string(data))
}

func upgradeV1(_ js.Value, args []js.Value) any {
	source := args[0].String()
	upgraded, changed := migrate.UpgradeV1ToV2(source)

	data, _ := json.Marshal(map[string]any{
		"source":  upgraded,
		"changed": changed,
	})
	return js.Global().Get("JSON").Call("parse", string(data))
}

func renderHTML(_ js.Value, args []js.Value) any {
	source := args[0].String()
	doc, errs := parser.Parse([]byte(source))
	if hasV1ParseError(errs) {
		return ""
	}

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
	doc, errs := parser.Parse([]byte(source))
	if hasV1ParseError(errs) {
		return js.Null()
	}

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

func isJSArray(v js.Value) bool {
	if v.Type() != js.TypeObject {
		return false
	}
	return js.Global().Get("Array").Call("isArray", v).Bool()
}

func hasV1ParseError(errs []*parser.ParseError) bool {
	for _, e := range errs {
		if e == nil {
			continue
		}
		msg := e.Message
		if strings.Contains(msg, "document-level metadata is a V1 pattern") ||
			strings.Contains(msg, "top-level Dramatis Personae is a V1 pattern") {
			return true
		}
	}
	return false
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
