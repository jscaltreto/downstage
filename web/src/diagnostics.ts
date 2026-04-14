import { StateEffect } from "@codemirror/state";
import { linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { EditorView } from "@codemirror/view";
import type { Text } from "@codemirror/state";
import type { EditorEnv, LSPTextEdit, SpellcheckContext, WasmDiagnostic } from "./core/types";
import { offsetFromLSP } from "./lsp-offsets";
import { getSpellDiagnostics, type SpellcheckCallbacks } from "./spellcheck";

export const spellcheckRefreshEffect = StateEffect.define<null>();

export function applyLSPEdits(view: EditorView, edits: LSPTextEdit[]) {
  if (!edits.length) return;
  const changes = edits.map((edit) => ({
    from: offsetFromLSP(view.state.doc, edit.range.start.line, edit.range.start.character),
    to: offsetFromLSP(view.state.doc, edit.range.end.line, edit.range.end.character),
    insert: edit.newText,
  }));
  view.dispatch({ changes, scrollIntoView: true });
}

function buildQuickFixActions(
  env: EditorEnv,
  code: string,
  titles: string[],
): Action[] {
  const codes = [code];
  return titles.map((title) => ({
    name: title,
    apply: async (view: EditorView, from: number) => {
      const doc = view.state.doc;
      const line = doc.lineAt(from);
      const lspLine = line.number - 1;
      const character = from - line.from;

      let result;
      try {
        result = await env.codeActions(doc.toString(), lspLine, character, codes);
      } catch {
        return;
      }

      const match = result?.actions?.find((a) => a.title === title);
      const edits = match?.edit?.changes?.[result.uri];
      if (!edits) return;
      applyLSPEdits(view, edits);
    },
  }));
}

export function toDiagnostics(
  env: EditorEnv,
  doc: Text,
  sourceDiagnostics: WasmDiagnostic[],
): Diagnostic[] {
  const result: Diagnostic[] = [];
  for (const d of sourceDiagnostics) {
    const startLine = d.line + 1;
    const endLine = d.endLine + 1;
    if (startLine > doc.lines || endLine > doc.lines) continue;

    const from = doc.line(startLine).from + d.col;
    let to = doc.line(endLine).from + d.endCol;
    if (to <= from) to = from + 1;

    const titles = d.code && d.quickFixes ? d.quickFixes : [];
    const actions = titles.length > 0 ? buildQuickFixActions(env, d.code!, titles) : undefined;

    result.push({
      from,
      to: Math.min(to, doc.line(endLine).to),
      severity: d.severity,
      message: d.message,
      source: "downstage",
      ...(d.code ? { code: d.code } : {}),
      actions,
    });
  }
  return result;
}

export function createDownstageLinter(
  env: EditorEnv,
  spellcheckEnabled: () => boolean,
  callbacks: SpellcheckCallbacks,
) {
  return linter(
    async (view) => {
      const source = view.state.doc.toString();
      const diagnosticsPromise = env.diagnostics(source);
      const spellContextPromise = spellcheckEnabled()
        ? env.spellcheckContext(source)
        : Promise.resolve<SpellcheckContext>({ allowWords: [], ignoredRanges: [] });
      const [{ diagnostics: sourceDiagnostics }, spellContext] = await Promise.all([
        diagnosticsPromise,
        spellContextPromise,
      ]);
      const diagnostics = toDiagnostics(env, view.state.doc, sourceDiagnostics);

      if (!spellcheckEnabled()) {
        return diagnostics;
      }

      const spellDiagnostics = await getSpellDiagnostics(view.state.doc, spellContext, callbacks);
      return diagnostics.concat(spellDiagnostics);
    },
    {
      delay: 300,
      needsRefresh: (update) => update.transactions.some((tr) =>
        tr.effects.some((effect) => effect.is(spellcheckRefreshEffect)),
      ),
    },
  );
}

export function refreshDiagnostics(view: EditorView) {
  view.dispatch({ effects: spellcheckRefreshEffect.of(null) });
}
