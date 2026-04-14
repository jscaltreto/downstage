import { describe, expect, it } from "vitest";
import { EditorState } from "@codemirror/state";
import type { Diagnostic } from "@codemirror/lint";
import { issuesStatus, projectDiagnostics, summarizeIssues } from "../core/issues";
import type { EditorDiagnostic } from "../core/types";

function docOf(source: string) {
  return EditorState.create({ doc: source }).doc;
}

describe("projectDiagnostics", () => {
  it("converts CodeMirror diagnostics to 1-based line/col", () => {
    const doc = docOf("first line\nsecond line\nthird line\n");
    const diagnostics: Diagnostic[] = [
      { from: 0, to: 5, severity: "error", message: "at start", source: "downstage" },
      { from: 11, to: 17, severity: "warning", message: "at second line", source: "downstage" },
    ];

    const projected = projectDiagnostics(doc, diagnostics);

    expect(projected).toHaveLength(2);
    expect(projected[0]).toMatchObject({ line: 1, col: 1, severity: "error" });
    expect(projected[1]).toMatchObject({ line: 2, col: 1, severity: "warning" });
  });

  it("filters out spellcheck diagnostics", () => {
    const doc = docOf("hello world\n");
    const diagnostics: Diagnostic[] = [
      { from: 0, to: 5, severity: "warning", message: "real issue", source: "downstage" },
      { from: 6, to: 11, severity: "warning", message: "typo", source: "spellcheck" },
    ];

    const projected = projectDiagnostics(doc, diagnostics);

    expect(projected).toHaveLength(1);
    expect(projected[0].message).toBe("real issue");
  });

  it("surfaces the custom code field when present", () => {
    const doc = docOf("something\n");
    const diagnostics = [
      {
        from: 0,
        to: 9,
        severity: "error" as const,
        message: "outdated",
        source: "downstage",
        code: "v1-document",
      },
    ];

    const projected = projectDiagnostics(doc, diagnostics as unknown as Diagnostic[]);

    expect(projected[0].code).toBe("v1-document");
  });

  it("computes col relative to the containing line", () => {
    const doc = docOf("abcdef\n123456\n");
    const diagnostics: Diagnostic[] = [
      { from: 9, to: 11, severity: "info", message: "mid-line", source: "downstage" },
    ];

    const [projected] = projectDiagnostics(doc, diagnostics);

    expect(projected.line).toBe(2);
    expect(projected.col).toBe(3);
  });
});

describe("summarizeIssues", () => {
  it("counts severities and total", () => {
    const items: EditorDiagnostic[] = [
      mk("error"),
      mk("error"),
      mk("warning"),
      mk("info"),
      mk("hint"),
    ];

    expect(summarizeIssues(items)).toEqual({
      errors: 2,
      warnings: 1,
      infos: 1,
      hints: 1,
      total: 5,
    });
  });

  it("returns zeros for an empty list", () => {
    expect(summarizeIssues([])).toEqual({
      errors: 0,
      warnings: 0,
      infos: 0,
      hints: 0,
      total: 0,
    });
  });
});

describe("issuesStatus", () => {
  it("reports error when any errors are present", () => {
    expect(issuesStatus({ errors: 1, warnings: 5, infos: 0, hints: 0, total: 6 })).toBe("error");
  });

  it("reports warning when only non-error diagnostics exist", () => {
    expect(issuesStatus({ errors: 0, warnings: 2, infos: 1, hints: 0, total: 3 })).toBe("warning");
  });

  it("reports clean when nothing is present", () => {
    expect(issuesStatus({ errors: 0, warnings: 0, infos: 0, hints: 0, total: 0 })).toBe("clean");
  });
});

function mk(severity: EditorDiagnostic["severity"]): EditorDiagnostic {
  return { from: 0, to: 1, line: 1, col: 1, severity, message: severity };
}
