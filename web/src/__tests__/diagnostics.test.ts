import { describe, expect, it } from "vitest";
import { EditorState, type TransactionSpec } from "@codemirror/state";
import { applyLSPEdits } from "../diagnostics";
import type { LSPTextEdit } from "../core/types";

function fakeView(initialDoc: string) {
  let state = EditorState.create({ doc: initialDoc });
  const view = {
    get state() {
      return state;
    },
    dispatch(spec: TransactionSpec) {
      state = state.update(spec).state;
    },
  };
  return view;
}

describe("applyLSPEdits", () => {
  it("applies a single-line TextEdit in place", () => {
    const view = fakeView("FRED\n");
    const edits: LSPTextEdit[] = [
      { range: { start: { line: 0, character: 0 }, end: { line: 0, character: 4 } }, newText: "FREDDY" },
    ];
    applyLSPEdits(view as any, edits);
    expect(view.state.doc.toString()).toBe("FREDDY\n");
  });

  it("applies multiple non-overlapping edits as a bulk change (e.g. renumber all scenes)", () => {
    const source = [
      "### SCENE A",
      "CHAR",
      "line",
      "",
      "### SCENE B",
      "CHAR",
      "line",
      "",
      "### SCENE C",
    ].join("\n");

    const view = fakeView(source);
    const edits: LSPTextEdit[] = [
      { range: { start: { line: 0, character: 0 }, end: { line: 0, character: 11 } }, newText: "### SCENE 1" },
      { range: { start: { line: 4, character: 0 }, end: { line: 4, character: 11 } }, newText: "### SCENE 2" },
      { range: { start: { line: 8, character: 0 }, end: { line: 8, character: 11 } }, newText: "### SCENE 3" },
    ];
    applyLSPEdits(view as any, edits);

    const updated = view.state.doc.toString();
    expect(updated).toContain("### SCENE 1");
    expect(updated).toContain("### SCENE 2");
    expect(updated).toContain("### SCENE 3");
    expect(updated).not.toContain("SCENE A");
    expect(updated).not.toContain("SCENE B");
    expect(updated).not.toContain("SCENE C");
  });

  it("is a no-op for an empty edit list", () => {
    const view = fakeView("unchanged\n");
    applyLSPEdits(view as any, []);
    expect(view.state.doc.toString()).toBe("unchanged\n");
  });
});
