import { describe, expect, it } from "vitest";
import { Text } from "@codemirror/state";
import { offsetFromLSP } from "../lsp-offsets";

describe("offsetFromLSP", () => {
  it("converts line/character to flat offset on ASCII content", () => {
    const doc = Text.of(["first", "second", "third"]);
    expect(offsetFromLSP(doc, 0, 0)).toBe(0);
    expect(offsetFromLSP(doc, 1, 0)).toBe(6);
    expect(offsetFromLSP(doc, 1, 3)).toBe(9);
    expect(offsetFromLSP(doc, 2, 5)).toBe(18);
  });

  it("clamps characters past the end of a line to the line's end", () => {
    const doc = Text.of(["short"]);
    expect(offsetFromLSP(doc, 0, 99)).toBe(5);
  });

  it("returns 0 for negative lines and end-of-doc for lines past the end", () => {
    const doc = Text.of(["a", "bb"]);
    expect(offsetFromLSP(doc, -1, 0)).toBe(0);
    expect(offsetFromLSP(doc, 99, 99)).toBe(4);
  });

  it("treats characters as UTF-16 code units (astral chars take 2 units)", () => {
    // "a𝐀b" where 𝐀 (U+1D400) is encoded as a surrogate pair (2 UTF-16 units).
    // LSP character for 'b' is 3 (1 + 2 surrogate units).
    const doc = Text.of(["a\uD835\uDC00b"]);
    expect(offsetFromLSP(doc, 0, 0)).toBe(0);
    expect(offsetFromLSP(doc, 0, 1)).toBe(1); // after 'a'
    expect(offsetFromLSP(doc, 0, 3)).toBe(3); // after 𝐀
    expect(offsetFromLSP(doc, 0, 4)).toBe(4); // after 'b'
  });
});
