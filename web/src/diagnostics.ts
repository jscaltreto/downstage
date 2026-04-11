import { linter, type Diagnostic } from "@codemirror/lint";
import type { EditorEnv, WasmDiagnostic } from "./core/types";

function toDiagnostics(
  doc: { line(n: number): { from: number; to: number } ; lines: number },
  sourceDiagnostics: WasmDiagnostic[],
): Diagnostic[] {
  const result: Diagnostic[] = [];
  for (const diagnostic of sourceDiagnostics) {
    const startLine = diagnostic.line + 1; // 0-based → 1-based
    const endLine = diagnostic.endLine + 1;
    if (startLine > doc.lines || endLine > doc.lines) continue;

    const from = doc.line(startLine).from + diagnostic.col;
    let to = doc.line(endLine).from + diagnostic.endCol;
    if (to <= from) to = from + 1;

    result.push({
      from,
      to: Math.min(to, doc.line(endLine).to),
      severity: diagnostic.severity,
      message: diagnostic.message,
      source: "downstage",
    });
  }
  return result;
}

export function createDownstageLinter(env: EditorEnv) {
  return linter(
    async (view) => {
      const source = view.state.doc.toString();
      const { diagnostics: sourceDiagnostics } = await env.diagnostics(source);
      return toDiagnostics(view.state.doc, sourceDiagnostics);
    },
    { delay: 300 },
  );
}
