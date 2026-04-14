import type { Diagnostic } from "@codemirror/lint";
import type { Text } from "@codemirror/state";
import type { EditorDiagnostic } from "./types";

type DiagnosticWithCode = Diagnostic & { code?: string };

export function projectDiagnostics(
  doc: Text,
  diagnostics: readonly Diagnostic[],
): EditorDiagnostic[] {
  const result: EditorDiagnostic[] = [];
  for (const d of diagnostics) {
    if (d.source === "spellcheck") continue;
    const line = doc.lineAt(d.from);
    const withCode = d as DiagnosticWithCode;
    result.push({
      from: d.from,
      to: d.to,
      line: line.number,
      col: d.from - line.from + 1,
      severity: d.severity,
      message: d.message,
      code: withCode.code,
    });
  }
  return result;
}

export interface IssuesSummary {
  errors: number;
  warnings: number;
  infos: number;
  hints: number;
  total: number;
}

export function summarizeIssues(items: readonly EditorDiagnostic[]): IssuesSummary {
  const summary: IssuesSummary = { errors: 0, warnings: 0, infos: 0, hints: 0, total: items.length };
  for (const d of items) {
    if (d.severity === "error") summary.errors++;
    else if (d.severity === "warning") summary.warnings++;
    else if (d.severity === "info") summary.infos++;
    else if (d.severity === "hint") summary.hints++;
  }
  return summary;
}

export type IssuesStatus = "clean" | "info" | "warning" | "error";

// FilterSeverity is the set of severities the IssuesDrawer pills can toggle.
// Info and hint diagnostics share a single pill (and therefore a single
// filter bucket), so there's no separate "hint" here.
export type FilterSeverity = "error" | "warning" | "info";

export function issuesStatus(summary: IssuesSummary): IssuesStatus {
  if (summary.errors > 0) return "error";
  if (summary.warnings > 0) return "warning";
  if (summary.infos > 0 || summary.hints > 0) return "info";
  return "clean";
}
