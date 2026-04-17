package desktop

import (
	"github.com/jscaltreto/downstage/internal/lsp"
	"github.com/jscaltreto/downstage/internal/migrate"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/stats"
	"go.lsp.dev/protocol"
)

type parseErrorJSON struct {
	Message string `json:"message"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	EndLine int    `json:"endLine"`
	EndCol  int    `json:"endCol"`
}

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

func (a *App) Parse(source string) []parseErrorJSON {
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
	return out
}

func (a *App) Diagnostics(source string) []diagnosticJSON {
	doc, errs := parser.Parse([]byte(source))
	diags := lsp.ComputeDiagnostics(doc, errs)

	out := make([]diagnosticJSON, len(diags))
	for i, diag := range diags {
		uri := protocol.DocumentURI("file:///local.ds")
		actions := lsp.ComputeCodeActions(doc, source, uri, []protocol.Diagnostic{diag}, diags)
		var quickFixes []string
		for _, action := range actions {
			quickFixes = append(quickFixes, action.Title)
		}

		out[i] = diagnosticJSON{
			Message:    diag.Message,
			Severity:   diagnosticSeverity(diag.Severity),
			Line:       int(diag.Range.Start.Line),
			Col:        int(diag.Range.Start.Character),
			EndLine:    int(diag.Range.End.Line),
			EndCol:     int(diag.Range.End.Character),
			Code:       diagnosticCode(diag.Code),
			QuickFixes: quickFixes,
		}
	}
	return out
}

type spellcheckContextJSON struct {
	AllowWords    []string         `json:"allowWords"`
	IgnoredRanges []protocol.Range `json:"ignoredRanges"`
}

func (a *App) SpellcheckContext(source string) spellcheckContextJSON {
	doc, errs := parser.Parse([]byte(source))
	ctx := lsp.ComputeSpellcheckContext(doc, errs)
	return spellcheckContextJSON{
		AllowWords:    ctx.AllowWords,
		IgnoredRanges: ctx.IgnoredRanges,
	}
}

type upgradeResultJSON struct {
	Source  string `json:"source"`
	Changed bool   `json:"changed"`
}

func (a *App) UpgradeV1(source string) upgradeResultJSON {
	upgraded, changed := migrate.UpgradeV1ToV2(source)
	return upgradeResultJSON{
		Source:  upgraded,
		Changed: changed,
	}
}

func (a *App) Completion(source string, line int, col int) *protocol.CompletionList {
	doc, errs := parser.Parse([]byte(source))
	pos := protocol.Position{Line: uint32(line), Character: uint32(col)}
	return lsp.ComputeCompletion(doc, errs, source, pos)
}

type lspTextEditJSON struct {
	Range   protocol.Range `json:"range"`
	NewText string         `json:"newText"`
}

type lspWorkspaceEditJSON struct {
	Changes map[string][]lspTextEditJSON `json:"changes"`
}

type lspCodeActionJSON struct {
	Title       string                `json:"title"`
	Kind        string                `json:"kind,omitempty"`
	IsPreferred bool                  `json:"isPreferred,omitempty"`
	Edit        *lspWorkspaceEditJSON `json:"edit,omitempty"`
}

type codeActionsResultJSON struct {
	URI     string              `json:"uri"`
	Actions []lspCodeActionJSON `json:"actions"`
}

func (a *App) CodeActions(source string, line int, col int, codes []string) codeActionsResultJSON {
	doc, errs := parser.Parse([]byte(source))

	uri := protocol.DocumentURI("file:///local.ds")
	allDiags := lsp.ComputeDiagnostics(doc, errs)

	codeSet := make(map[string]struct{})
	for _, c := range codes {
		codeSet[c] = struct{}{}
	}

	var diags []protocol.Diagnostic
	for _, d := range allDiags {
		codeStr := diagnosticCode(d.Code)
		if _, ok := codeSet[codeStr]; ok {
			diags = append(diags, d)
		}
	}

	actions := lsp.ComputeCodeActions(doc, source, uri, diags, allDiags)

	outActions := make([]lspCodeActionJSON, 0, len(actions))
	for _, a := range actions {
		ca := lspCodeActionJSON{
			Title:       a.Title,
			Kind:        string(a.Kind),
			IsPreferred: a.IsPreferred,
		}
		if a.Edit != nil {
			ca.Edit = &lspWorkspaceEditJSON{
				Changes: make(map[string][]lspTextEditJSON),
			}
			for u, edits := range a.Edit.Changes {
				simpleEdits := make([]lspTextEditJSON, len(edits))
				for i, e := range edits {
					simpleEdits[i] = lspTextEditJSON{
						Range:   e.Range,
						NewText: e.NewText,
					}
				}
				ca.Edit.Changes[string(u)] = simpleEdits
			}
		}
		outActions = append(outActions, ca)
	}

	return codeActionsResultJSON{
		URI:     string(uri),
		Actions: outActions,
	}
}

type documentSymbolsResultJSON struct {
	Symbols []protocol.DocumentSymbol `json:"symbols"`
}

func (a *App) DocumentSymbols(source string) documentSymbolsResultJSON {
	doc, errs := parser.Parse([]byte(source))
	symbols := lsp.ComputeDocumentSymbols(doc, errs)
	if symbols == nil {
		symbols = []protocol.DocumentSymbol{}
	}
	return documentSymbolsResultJSON{
		Symbols: symbols,
	}
}

func (a *App) SemanticTokens(source string) []uint32 {
	doc, errs := parser.Parse([]byte(source))
	return lsp.ComputeSemanticTokens(doc, errs)
}

func (a *App) TokenTypeNames() []string {
	return lsp.SemanticTokenTypeNames
}

func (a *App) Stats(source string) stats.Stats {
	doc, _ := parser.Parse([]byte(source))
	return stats.Compute(doc, stats.RuntimeOptions{})
}
