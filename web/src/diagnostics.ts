import { linter, type Diagnostic } from "@codemirror/lint";
import { parse, type ParseError } from "./wasm";

function toDiagnostics(
  doc: { line(n: number): { from: number; to: number } ; lines: number },
  errors: ParseError[],
): Diagnostic[] {
  const result: Diagnostic[] = [];
  for (const err of errors) {
    const startLine = err.line + 1; // 0-based → 1-based
    const endLine = err.endLine + 1;
    if (startLine > doc.lines || endLine > doc.lines) continue;

    const from = doc.line(startLine).from + err.col;
    let to = doc.line(endLine).from + err.endCol;
    if (to <= from) to = from + 1;

    result.push({
      from,
      to: Math.min(to, doc.line(endLine).to),
      severity: "error",
      message: err.message,
    });
  }
  return result;
}

export const downstageLinter = linter(
  (view) => {
    const source = view.state.doc.toString();
    const { errors } = parse(source);
    return toDiagnostics(view.state.doc, errors);
  },
  { delay: 300 },
);
