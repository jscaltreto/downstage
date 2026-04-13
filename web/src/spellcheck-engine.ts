type TypoConstructor = new (dictionary: string, affData?: string, wordsData?: string) => TypoInstance;

interface TypoInstance {
  check(word: string): boolean;
  suggest(word: string, limit?: number): string[];
}

export type SpellDictionary = TypoInstance;

const maxSuggestions = 3;

export async function loadSpellDictionary(): Promise<SpellDictionary> {
  const [typoModule, affModule, dicModule] = await Promise.all([
    import("typo-js"),
    import("typo-js/dictionaries/en_US/en_US.aff?raw"),
    import("typo-js/dictionaries/en_US/en_US.dic?raw"),
  ]);
  const Typo = typoModule.default as TypoConstructor;
  return new Typo("en_US", affModule.default, dicModule.default);
}

export function checkWords(dict: SpellDictionary, words: string[]): Record<string, boolean> {
  const out: Record<string, boolean> = {};
  for (const w of words) {
    if (dict.check(w)) {
      out[w] = true;
      continue;
    }
    // English possessive constructions like "else's", "someone's",
    // "everyone's" aren't in the Hunspell dictionary as inflected forms,
    // but the bare word is. Accept the possessive when the bare word
    // checks out — same shape as the cue-allowlist possessive handling
    // in spellcheck.ts, just for the dictionary.
    if (w.endsWith("'s") && dict.check(w.slice(0, -2))) {
      out[w] = true;
      continue;
    }
    out[w] = false;
  }
  return out;
}

export function suggestForWord(dict: SpellDictionary, word: string): string[] {
  return dict.suggest(word, maxSuggestions);
}
