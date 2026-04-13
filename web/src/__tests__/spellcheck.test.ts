import { describe, expect, it } from "vitest";
import { Text } from "@codemirror/state";
import { getSpellDiagnostics, matchCase } from "../spellcheck";
import type { SpellcheckContext } from "../core/types";

function spellContext(overrides?: Partial<SpellcheckContext>): SpellcheckContext {
  return {
    allowWords: [],
    ignoredRanges: [],
    ...overrides,
  };
}

// typo-js dictionary parse runs once on first call and is cached; bump the
// default 5s timeout to accommodate the cold load.
describe("getSpellDiagnostics", { timeout: 30000 }, () => {
  it("produces a lazy diagnostic with only the add-word action up front", async () => {
    const doc = Text.of(["mispelled"]);
    const diagnostics = await getSpellDiagnostics(doc, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });

    expect(diagnostics).toHaveLength(1);
    const [diag] = diagnostics;
    expect(diag.source).toBe("spellcheck");
    expect(diag.severity).toBe("warning");
    expect(diag.markClass).toBe(" cm-spellcheckRange");
    // Replacement suggestions are rendered lazily via renderMessage, so the
    // eager action list only holds the add-word action.
    expect(diag.actions).toHaveLength(1);
    expect(diag.actions?.[0]?.name).toBe('Add "mispelled" to this script dictionary');
    expect(typeof diag.renderMessage).toBe("function");
  });

  it("suppresses diagnostics for ignored ranges and merged allowlists", async () => {
    const doc = Text.of(["HAMLET mispelled bespoke"]);
    const diagnostics = await getSpellDiagnostics(doc, spellContext({
      allowWords: ["HAMLET"],
      ignoredRanges: [{
        start: { line: 0, character: 7 },
        end: { line: 0, character: 16 },
      }],
    }), {
      getUserAllowlist: () => ["bespoke"],
      addWord: async () => true,
    });

    expect(diagnostics).toHaveLength(0);
  });

  it("treats null allowWords and ignoredRanges as empty arrays", async () => {
    const doc = Text.of(["hello"]);
    const diagnostics = await getSpellDiagnostics(doc, {
      allowWords: null,
      ignoredRanges: null,
    } as any, {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });

    expect(diagnostics).toHaveLength(0);
  });

  it("does not flag KEEPCASE proper nouns or capitalized contractions", async () => {
    // Liz, Michael, English are all in en_US Hunspell with KEEPCASE; "I've"
    // is the canonical capitalized contraction. All previously failed because
    // we lowercased before check(), tripping KEEPCASE.
    const doc = Text.of(["Hello Liz and Michael, I've been reading English."]);
    const diagnostics = await getSpellDiagnostics(doc, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });
    expect(diagnostics).toHaveLength(0);
  });

  it("skips letter-plural constructs like \"M's\" and short contractions", async () => {
    // "M's" and "H's" are letter-plurals from prose like "two M's"; "I'd"
    // and "I'm" are short contractions. None should flag.
    const doc = Text.of(["Two M's and three H's, I'd say I'm sure."]);
    const diagnostics = await getSpellDiagnostics(doc, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });
    expect(diagnostics).toHaveLength(0);
  });

  it("accepts possessives of dictionary words like \"else's\" and \"someone's\"", async () => {
    // These constructions don't have inflected forms in Hunspell en_US, but
    // the bare words do. The engine retries the bare form on a 's-suffix
    // failure rather than flagging "somebody else's".
    const doc = Text.of(["This is somebody else's problem and someone's mistake."]);
    const diagnostics = await getSpellDiagnostics(doc, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });
    expect(diagnostics).toHaveLength(0);
  });

  it("treats a possessive of an allowlisted name as allowlisted", async () => {
    // Stage direction: "ALICE crosses to ALICE's chair." The character cue
    // allowlist contains ALICE but not ALICE's; we should still skip the
    // possessive form rather than flagging it with an Alice's "fix".
    const doc = Text.of(["ALICE crosses to ALICE's chair."]);
    const diagnostics = await getSpellDiagnostics(doc, spellContext({
      allowWords: ["ALICE"],
    }), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });
    expect(diagnostics).toHaveLength(0);
  });

  it("skips mid-sentence Title Case words but still checks them at sentence start", async () => {
    // Esmerelda is not in en_US dict. Mid-sentence: skipped (assumed proper
    // noun). Sentence-initial: still checked, so a typo of "Teh" is flagged.
    const midSentence = Text.of(["I greeted Esmerelda warmly."]);
    const midDiags = await getSpellDiagnostics(midSentence, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });
    expect(midDiags).toHaveLength(0);

    const sentenceStart = Text.of(["Teh quick brown fox."]);
    const startDiags = await getSpellDiagnostics(sentenceStart, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });
    expect(startDiags.some((d) => d.message.includes("Teh"))).toBe(true);
  });

});

describe("matchCase", () => {
  it("preserves lowercase suggestions for lowercase originals", () => {
    expect(matchCase("teh", "the")).toBe("the");
  });

  it("upper-cases the first letter for Title Case originals", () => {
    expect(matchCase("Teh", "the")).toBe("The");
  });

  it("upper-cases the entire suggestion for ALL-CAPS originals", () => {
    expect(matchCase("TEH", "the")).toBe("THE");
  });

  it("treats single uppercase letters as Title Case, not ALL-CAPS", () => {
    // A single-letter uppercase word like "I" shouldn't trigger the ALL-CAPS
    // path (which only kicks in for 2+ chars); the Title Case rule should
    // capitalize the first letter of the suggestion.
    expect(matchCase("I", "in")).toBe("In");
  });

  it("leaves mixed-case originals (like \"iPhone\") untouched", () => {
    expect(matchCase("iPhone", "iphone")).toBe("iphone");
  });

  it("returns empty suggestions unchanged", () => {
    expect(matchCase("Teh", "")).toBe("");
  });
});

describe("getSpellDiagnostics (extended)", { timeout: 30000 }, () => {
  it("continues scanning past the first hundred candidate words", async () => {
    const filler = Array.from({ length: 130 }, () => "hello").join(" ");
    // Use a lowercase typo so the Title Case proper-noun heuristic doesn't
    // skip it on its own.
    const doc = Text.of([`${filler} ascenddsion`]);

    const diagnostics = await getSpellDiagnostics(doc, spellContext(), {
      getUserAllowlist: () => [],
      addWord: async () => true,
    });

    expect(diagnostics.some((diagnostic) => diagnostic.message.includes("ascenddsion"))).toBe(true);
  });
});
