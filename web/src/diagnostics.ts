import { linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { EditorView } from "@codemirror/view";
import type { Text } from "@codemirror/state";
import type { EditorEnv, LSPCodeAction, LSPTextEdit, WasmDiagnostic } from "./core/types";
import { offsetFromLSP } from "./lsp-offsets";

function applyWorkspaceEdit(view: EditorView, uri: string, action: LSPCodeAction) {
  const edits = action.edit?.changes?.[uri];
  if (!edits || edits.length === 0) return;

  const changes = edits.map((edit: LSPTextEdit) => ({
    from: offsetFromLSP(view.state.doc, edit.range.start.line, edit.range.start.character),
    to: offsetFromLSP(view.state.doc, edit.range.end.line, edit.range.end.character),
    insert: edit.newText,
  }));

  view.dispatch({ changes, scrollIntoView: true });
}

async function actionsFor(
  env: EditorEnv,
  source: string,
  diagnostic: WasmDiagnostic,
): Promise<Action[]> {
  if (!diagnostic.code) return [];

  let result;
  try {
    result = await env.codeActions(source, diagnostic.line, diagnostic.col, [diagnostic.code]);
  } catch {
    return [];
  }

  if (!result || !result.actions) return [];

  return result.actions
    .filter((action) => action.edit?.changes?.[result.uri]?.length)
    .map((action) => ({
      name: action.title,
      apply: (view: EditorView) => applyWorkspaceEdit(view, result.uri, action),
    }));
}

async function toDiagnostics(
  env: EditorEnv,
  source: string,
  doc: Text,
  sourceDiagnostics: WasmDiagnostic[],
): Promise<Diagnostic[]> {
  const actionLists = await Promise.all(
    sourceDiagnostics.map((d) => actionsFor(env, source, d)),
  );

  const result: Diagnostic[] = [];
  for (let i = 0; i < sourceDiagnostics.length; i++) {
    const diagnostic = sourceDiagnostics[i];
    const startLine = diagnostic.line + 1; // 0-based → 1-based
    const endLine = diagnostic.endLine + 1;
    if (startLine > doc.lines || endLine > doc.lines) continue;

    const from = doc.line(startLine).from + diagnostic.col;
    let to = doc.line(endLine).from + diagnostic.endCol;
    if (to <= from) to = from + 1;

    const actions = actionLists[i];

    result.push({
      from,
      to: Math.min(to, doc.line(endLine).to),
      severity: diagnostic.severity,
      message: diagnostic.message,
      source: "downstage",
      actions: actions.length ? actions : undefined,
    });
  }
  return result;
}

export function createDownstageLinter(env: EditorEnv) {
  return linter(
    async (view) => {
      const source = view.state.doc.toString();
      const { diagnostics: sourceDiagnostics } = await env.diagnostics(source);
      return toDiagnostics(env, source, view.state.doc, sourceDiagnostics);
    },
    { delay: 300 },
  );
}
