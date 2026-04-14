import { describe, expect, it } from "vitest";
import { EditorState, Text } from "@codemirror/state";
import {
  findMatches,
  searchExtension,
  searchStateField,
  setSearchEffect,
} from "../core/search";

function doc(s: string): Text {
  return Text.of(s.split("\n"));
}

describe("findMatches", () => {
  const script = [
    "ALICE",
    "Alice in wonderland.",
    "An ALICE search for ALICE.",
    "palice palindrome",
  ].join("\n");

  it("returns [] for empty query", () => {
    const res = findMatches(doc(script), {
      query: "",
      caseSensitive: false,
      wholeWord: false,
      regex: false,
    });
    expect(res).toEqual({ ok: true, matches: [] });
  });

  it("matches case-insensitively by default", () => {
    const res = findMatches(doc(script), {
      query: "alice",
      caseSensitive: false,
      wholeWord: false,
      regex: false,
    });
    if (!res.ok) throw new Error("expected ok");
    expect(res.matches).toHaveLength(5);
    expect(res.matches[0]).toMatchObject({ line: 1, col: 1 });
  });

  it("respects matchCase", () => {
    const res = findMatches(doc(script), {
      query: "ALICE",
      caseSensitive: true,
      wholeWord: false,
      regex: false,
    });
    if (!res.ok) throw new Error("expected ok");
    expect(res.matches).toHaveLength(3);
    expect(res.matches.every((m) => m.lineText.includes("ALICE"))).toBe(true);
  });

  it("respects wholeWord boundaries", () => {
    const res = findMatches(doc(script), {
      query: "alice",
      caseSensitive: false,
      wholeWord: true,
      regex: false,
    });
    if (!res.ok) throw new Error("expected ok");
    expect(res.matches).toHaveLength(4);
    expect(res.matches.every((m) => !m.lineText.toLowerCase().includes("palice") || m.col !== 1)).toBe(true);
  });

  it("supports regex and returns a parse error when invalid", () => {
    const ok = findMatches(doc(script), {
      query: "A[li]+CE",
      caseSensitive: false,
      wholeWord: false,
      regex: true,
    });
    if (!ok.ok) throw new Error("expected ok");
    expect(ok.matches.length).toBeGreaterThan(0);

    const bad = findMatches(doc(script), {
      query: "A[li",
      caseSensitive: false,
      wholeWord: false,
      regex: true,
    });
    expect(bad.ok).toBe(false);
  });

  it("combines regex and wholeWord", () => {
    const res = findMatches(doc(script), {
      query: "ali.e",
      caseSensitive: false,
      wholeWord: true,
      regex: true,
    });
    if (!res.ok) throw new Error("expected ok");
    for (const m of res.matches) {
      const before = m.col === 1 ? "" : m.lineText[m.col - 2];
      expect(/\w/.test(before)).toBe(false);
    }
  });

  it("records line and col (1-based)", () => {
    const res = findMatches(doc(script), {
      query: "wonderland",
      caseSensitive: true,
      wholeWord: false,
      regex: false,
    });
    if (!res.ok) throw new Error("expected ok");
    expect(res.matches).toHaveLength(1);
    expect(res.matches[0].line).toBe(2);
    expect(res.matches[0].col).toBe(10);
  });

  it("resets currentIndex to 0 when the query changes but preserves it on re-runs", () => {
    const doc = [
      "first ALICE line",
      "second line",
      "third ALICE line",
      "fourth ALICE line",
    ].join("\n");
    let state = EditorState.create({ doc, extensions: [searchExtension()] });
    const opts = { query: "ALICE", caseSensitive: true, wholeWord: false, regex: false };

    state = state.update({ effects: setSearchEffect.of(opts) }).state;
    const initial = state.field(searchStateField);
    expect(initial.matches).toHaveLength(3);
    expect(initial.currentIndex).toBe(0);

    state = state.update({
      effects: setSearchEffect.of({ ...opts, query: "line" }),
    }).state;
    const afterQueryChange = state.field(searchStateField);
    expect(afterQueryChange.matches.length).toBeGreaterThan(0);
    expect(afterQueryChange.currentIndex).toBe(0);

    state = state.update({ effects: setSearchEffect.of({ ...opts, query: "line" }) }).state;
    const afterSameRun = state.field(searchStateField);
    expect(afterSameRun.currentIndex).toBe(0);
  });

  it("captures the line text for match context", () => {
    const res = findMatches(doc(script), {
      query: "ALICE",
      caseSensitive: true,
      wholeWord: true,
      regex: false,
    });
    if (!res.ok) throw new Error("expected ok");
    expect(res.matches[0].lineText).toBe("ALICE");
    expect(res.matches[1].lineText).toBe("An ALICE search for ALICE.");
  });
});
