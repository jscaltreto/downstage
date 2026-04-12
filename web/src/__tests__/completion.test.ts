import { describe, expect, it } from "vitest";
import { EditorState, type TransactionSpec } from "@codemirror/state";
import type { Completion } from "@codemirror/autocomplete";
import type { EditorView } from "@codemirror/view";
import { toCompletion } from "../completion";
import type { LSPCompletionItem } from "../core/types";

type ApplyFn = (view: EditorView, completion: Completion, from: number, to: number) => void;

function applyFn(completion: Completion): ApplyFn {
  if (typeof completion.apply !== "function") {
    throw new Error("expected completion.apply to be a function");
  }
  return completion.apply;
}

function fakeView(initialDoc: string, selectionAnchor?: number) {
  let state = EditorState.create({
    doc: initialDoc,
    selection: selectionAnchor !== undefined ? { anchor: selectionAnchor } : undefined,
  });
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

describe("toCompletion", () => {
  it("applies a TextEdit that replaces the line prefix with the chosen label", () => {
    // "B" on line 1; LSP range is (line=1, char=0)..(line=1, char=1), replacing with "BOB".
    const source = "\nB";
    const view = fakeView(source, source.length);
    const item: LSPCompletionItem = {
      label: "BOB",
      kind: 6,
      textEdit: {
        range: { start: { line: 1, character: 0 }, end: { line: 1, character: 1 } },
        newText: "BOB",
      },
    };

    const completion = toCompletion(item);
    applyFn(completion)(view as unknown as EditorView, completion, 0, 0);
    expect(view.state.doc.toString()).toBe("\nBOB");
  });

  it("falls back to an insert at the cursor when no TextEdit is provided", () => {
    const view = fakeView("hi ", 3);
    const item: LSPCompletionItem = { label: "there" };
    const completion = toCompletion(item);
    applyFn(completion)(view as unknown as EditorView, completion, 3, 3);
    expect(view.state.doc.toString()).toBe("hi there");
  });

  it("handles astral-character documents via UTF-16 offsets", () => {
    // "𝐀B" → 𝐀 is 2 UTF-16 units; replacing LSP range (0,2)..(0,3) swaps "B" for "BOB".
    const doc = "\uD835\uDC00B";
    const view = fakeView(doc, doc.length);
    const item: LSPCompletionItem = {
      label: "BOB",
      textEdit: {
        range: { start: { line: 0, character: 2 }, end: { line: 0, character: 3 } },
        newText: "BOB",
      },
    };
    const completion = toCompletion(item);
    applyFn(completion)(view as unknown as EditorView, completion, 0, 0);
    expect(view.state.doc.toString()).toBe("\uD835\uDC00BOB");
  });
});
