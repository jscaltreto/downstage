import { type Action, type Diagnostic } from "@codemirror/lint";
import type { EditorView } from "@codemirror/view";
import type { Text } from "@codemirror/state";
import type { SpellcheckContext } from "./core/types";
import { offsetFromLSP } from "./lsp-offsets";
// Type-only import so the engine module (and its mnemonist + typo-js
// dependencies) doesn't land in the main bundle. The inline fallback below
// pulls them in via a dynamic import, kept separate so the worker bundle is
// the only place runtime spellcheck code ships in production.
import type { SpellDictionary } from "./spellcheck-engine";

interface SpellIssue {
  from: number;
  to: number;
  word: string;
}

export interface SpellcheckCallbacks {
  getUserAllowlist: () => string[];
  addWord: (word: string) => Promise<boolean>;
}

interface SpellBackend {
  warmup(): Promise<void>;
  check(words: string[]): Promise<Record<string, boolean>>;
  suggest(word: string): Promise<string[]>;
}

const minWordLength = 3;
const maxDiagnostics = 100;
const maxSuggestions = 3;

const checkCache = new Map<string, boolean>();
const suggestionsCache = new Map<string, string[]>();

let backend: SpellBackend | null = null;

// Apostrophe normalization without lowercasing — used for the value we hand
// to typo.check(), so KEEPCASE proper nouns ("Liz", "Michael") and
// capitalized contractions ("I've") aren't lowercased into a check failure.
function normalizeApostrophes(word: string) {
  return word.trim().replace(/’/g, "'");
}

// Lowercased form for case-insensitive lookups: allowlist matching and the
// SymSpell suggest index (which is built from lowercased dictionary stems).
function normalizeWord(word: string) {
  return normalizeApostrophes(word).toLocaleLowerCase();
}

function alphaLength(word: string) {
  let count = 0;
  for (let i = 0; i < word.length; i++) {
    const ch = word.charCodeAt(i);
    if ((ch >= 65 && ch <= 90) || (ch >= 97 && ch <= 122)) count++;
  }
  return count;
}

function shouldCheckWord(word: string) {
  // Count letters only — apostrophes shouldn't push tokens like "M's" or
  // "I'd" past the minimum length and into the checked set.
  if (alphaLength(word) < minWordLength) return false;
  if (/\d/.test(word)) return false;
  return true;
}

// Browser-spellchecker convention: skip mid-sentence Title Case words. The
// vast majority of capitalized words appearing inside a clause are proper
// nouns the user knows are correct; checking them produces noise far more
// often than catches a real typo. Sentence-initial Title Case still gets
// checked so genuine misspellings ("Teh document is...") aren't missed.
function isLikelyProperNoun(word: string, source: string, index: number) {
  // Title Case = leading uppercase, at least one trailing lowercase letter,
  // optional apostrophes. Excludes ALL-CAPS (cues, acronyms) and mixed-case
  // identifiers like "iPhone".
  if (!/^[A-Z][a-z'’]+$/.test(word)) return false;

  // Walk back through inline whitespace only. A line break, sentence-ender,
  // or Downstage structural marker (>, :) means the word is sentence-initial.
  let i = index - 1;
  while (i >= 0 && (source[i] === " " || source[i] === "\t")) i--;
  if (i < 0) return false;
  const prev = source[i];
  if (prev === "\n" || prev === "." || prev === "!" || prev === "?" || prev === ">" || prev === ":") {
    return false;
  }
  return true;
}

function rangeOverlaps(from: number, to: number, ignoredRanges: readonly { from: number; to: number }[]) {
  for (const range of ignoredRanges) {
    if (range.to <= from) continue;
    if (range.from >= to) break;
    return true;
  }
  return false;
}

function collectAllowlist(words: string[]) {
  return new Set(
    words
      .flatMap((word) => word.match(/\b[A-Za-z][A-Za-z'’]*\b/g) || [])
      .map((word) => normalizeWord(word)),
  );
}

// Match a word against the allowlist, also accepting the bare form of a
// possessive. Stage directions routinely use "ALICE's hand" — the cue
// allowlist has ALICE but not its possessive. Strip a trailing "'s" and
// retry so the bare character name covers both forms.
function isAllowlisted(word: string, allowlist: Set<string>) {
  const key = normalizeWord(word);
  if (allowlist.has(key)) return true;
  if (key.endsWith("'s") && allowlist.has(key.slice(0, -2))) return true;
  return false;
}

function collectSpellIssues(doc: Text, ignoredRanges: readonly { from: number; to: number }[]) {
  const issues: SpellIssue[] = [];
  const source = doc.toString();
  const wordPattern = /\b[A-Za-z][A-Za-z'’]*\b/g;

  for (const match of source.matchAll(wordPattern)) {
    const word = match[0];
    const index = match.index;
    if (index === undefined || !shouldCheckWord(word)) continue;
    if (isLikelyProperNoun(word, source, index)) continue;
    const from = index;
    const to = index + word.length;
    if (rangeOverlaps(from, to, ignoredRanges)) continue;
    issues.push({ from, to, word });
  }

  return issues;
}

function buildAddWordAction(word: string, callbacks: SpellcheckCallbacks): Action {
  return {
    name: `Add "${word}" to this script dictionary`,
    async apply() {
      await callbacks.addWord(word);
    },
  };
}

// Worker backend: runs typo-js entirely off the main thread. Dictionary
// construction no longer blocks the editor; the first lint pass awaits the
// worker's "ready" signal before producing spell diagnostics.
//
// Failure handling: a worker that fails before posting "ready" used to leave
// the ready promise unresolved forever (silently disabling spellcheck for
// the session). We now wire `error` and `messageerror` listeners in addition
// to the `init-error` message, and mark the backend "broken" so subsequent
// requests fail fast instead of queueing into the void.
function createWorkerBackend(): SpellBackend {
  const worker = new Worker(new URL("./spellcheck.worker.ts", import.meta.url), { type: "module" });

  let readyResolve!: () => void;
  let readyReject!: (err: Error) => void;
  let resolved = false;
  const ready = new Promise<void>((resolve, reject) => {
    readyResolve = () => { resolved = true; resolve(); };
    readyReject = (err) => { resolved = true; reject(err); };
  });

  type Pending = { resolve: (value: unknown) => void; reject: (err: Error) => void };
  const pending = new Map<number, Pending>();
  let broken = false;
  let nextId = 0;

  const fail = (err: Error) => {
    broken = true;
    if (!resolved) readyReject(err);
    for (const p of pending.values()) p.reject(err);
    pending.clear();
  };

  worker.addEventListener("message", (event: MessageEvent) => {
    const msg = event.data;
    if (msg.type === "ready") {
      readyResolve();
    } else if (msg.type === "init-error") {
      fail(new Error(msg.message));
    } else if (msg.type === "check") {
      pending.get(msg.id)?.resolve(msg.results);
      pending.delete(msg.id);
    } else if (msg.type === "suggest") {
      pending.get(msg.id)?.resolve(msg.suggestions);
      pending.delete(msg.id);
    }
  });

  worker.addEventListener("error", (event) => {
    event.preventDefault?.();
    fail(new Error(`spellcheck worker error: ${event.message || "unknown"}`));
  });

  worker.addEventListener("messageerror", () => {
    fail(new Error("spellcheck worker message deserialization failed"));
  });

  const request = <T>(payload: unknown): Promise<T> => {
    if (broken) return Promise.reject(new Error("spellcheck worker is unavailable"));
    const id = ++nextId;
    return new Promise<T>((resolve, reject) => {
      pending.set(id, { resolve: resolve as (v: unknown) => void, reject });
      worker.postMessage({ ...(payload as object), id });
    });
  };

  return {
    warmup: () => ready,
    check: async (words) => {
      await ready;
      return request<Record<string, boolean>>({ type: "check", words });
    },
    suggest: async (word) => {
      await ready;
      return request<string[]>({ type: "suggest", word });
    },
  };
}

// Inline backend: used when Worker is unavailable (node tests, old browsers).
// The engine module is dynamically imported so its mnemonist + typo-js
// dependencies are emitted as a separate chunk that only loads if this
// path runs — modern browsers go through the worker and never fetch it.
function createInlineBackend(): SpellBackend {
  type Engine = typeof import("./spellcheck-engine");
  let enginePromise: Promise<Engine> | null = null;
  let dictPromise: Promise<SpellDictionary> | null = null;

  const loadEngine = () => {
    if (!enginePromise) enginePromise = import("./spellcheck-engine");
    return enginePromise;
  };

  const load = () => {
    if (!dictPromise) {
      dictPromise = loadEngine()
        .then((engine) => engine.loadSpellDictionary())
        .catch((err) => {
          dictPromise = null;
          throw err;
        });
    }
    return dictPromise;
  };

  return {
    warmup: async () => { await load(); },
    check: async (words) => {
      const [engine, dict] = await Promise.all([loadEngine(), load()]);
      return engine.checkWords(dict, words);
    },
    suggest: async (word) => {
      const [engine, dict] = await Promise.all([loadEngine(), load()]);
      return engine.suggestForWord(dict, word);
    },
  };
}

function getBackend(): SpellBackend {
  if (backend) return backend;
  if (typeof Worker !== "undefined") {
    try {
      backend = createWorkerBackend();
      return backend;
    } catch (err) {
      console.warn("spellcheck worker unavailable, falling back to inline:", err);
    }
  }
  backend = createInlineBackend();
  return backend;
}

export async function warmSpellDictionary(): Promise<void> {
  await getBackend().warmup();
}

function ignoredRangesFromContext(doc: Text, context: SpellcheckContext) {
  return (context.ignoredRanges ?? [])
    .map((range) => ({
      from: offsetFromLSP(doc, range.start.line, range.start.character),
      to: offsetFromLSP(doc, range.end.line, range.end.character),
    }))
    .sort((a, b) => a.from - b.from || a.to - b.to);
}

function buildReplaceButton(view: EditorView, issue: SpellIssue, suggestion: string) {
  const btn = document.createElement("button");
  btn.type = "button";
  // Reuse the built-in action styling so lint.css + our Editor.vue theme apply.
  btn.className = "cm-diagnosticAction";
  btn.textContent = `Replace with "${suggestion}"`;
  btn.addEventListener("click", () => {
    view.dispatch({
      changes: { from: issue.from, to: issue.to, insert: suggestion },
      selection: { anchor: issue.from + suggestion.length },
      scrollIntoView: true,
    });
  });
  return btn;
}

function renderSuggestionButtons(
  list: HTMLElement,
  view: EditorView,
  issue: SpellIssue,
  suggestions: string[],
) {
  const deduped = Array.from(new Set(suggestions)).slice(0, maxSuggestions);
  list.replaceChildren();
  if (deduped.length === 0) {
    const empty = document.createElement("span");
    empty.className = "cm-spellEmpty";
    empty.textContent = "No suggestions.";
    list.appendChild(empty);
    return;
  }
  for (const suggestion of deduped) {
    list.appendChild(buildReplaceButton(view, issue, suggestion));
  }
}

// Project the case shape of `original` onto `suggestion` so a sentence-start
// typo like "Teh" gets "The" (not "the") and an ALL-CAPS shout like "TEH"
// gets "THE". typo-js suggestions come back lowercased; without this the
// replacement buttons silently drop capitalization the user typed.
export function matchCase(original: string, suggestion: string): string {
  if (suggestion.length === 0) return suggestion;
  const trimmed = original.trim();
  if (trimmed.length >= 2 && trimmed === trimmed.toLocaleUpperCase() && trimmed !== trimmed.toLocaleLowerCase()) {
    return suggestion.toLocaleUpperCase();
  }
  const first = trimmed[0];
  if (first && first === first.toLocaleUpperCase() && first !== first.toLocaleLowerCase()) {
    return suggestion[0].toLocaleUpperCase() + suggestion.slice(1);
  }
  return suggestion;
}

function casedSuggestions(original: string, suggestions: string[]): string[] {
  return suggestions.map((s) => matchCase(original, s));
}

async function populateSuggestions(list: HTMLElement, view: EditorView, issue: SpellIssue) {
  const key = normalizeWord(issue.word);
  let suggestions = suggestionsCache.get(key);
  if (!suggestions) {
    try {
      suggestions = await getBackend().suggest(key);
    } catch {
      suggestions = [];
    }
    suggestionsCache.set(key, suggestions);
  }
  renderSuggestionButtons(list, view, issue, casedSuggestions(issue.word, suggestions));
}

function renderSpellMessage(view: EditorView, issue: SpellIssue, message: string): Node {
  const root = document.createElement("div");
  root.className = "cm-spellMessage";

  const text = document.createElement("div");
  text.className = "cm-spellText";
  text.textContent = message;
  root.appendChild(text);

  const list = document.createElement("div");
  list.className = "cm-spellSuggestions";
  root.appendChild(list);

  const key = normalizeWord(issue.word);
  const cached = suggestionsCache.get(key);
  if (cached) {
    renderSuggestionButtons(list, view, issue, casedSuggestions(issue.word, cached));
    return root;
  }

  const loading = document.createElement("span");
  loading.className = "cm-spellLoading";
  loading.textContent = "Finding suggestions…";
  list.appendChild(loading);

  void populateSuggestions(list, view, issue);
  return root;
}

export async function getSpellDiagnostics(
  doc: Text,
  context: SpellcheckContext,
  callbacks: SpellcheckCallbacks,
): Promise<Diagnostic[]> {
  const userAllowlist = callbacks.getUserAllowlist() || [];
  const allowlist = collectAllowlist([
    ...(context.allowWords ?? []),
    ...userAllowlist,
  ]);
  const ignoredRanges = ignoredRangesFromContext(doc, context);
  const issues = collectSpellIssues(doc, ignoredRanges);

  // Drop allowlisted words before any RPC so we don't pay round-trip cost
  // for cues, user-added words, or structural tokens.
  const candidates = issues.filter((issue) => !isAllowlisted(issue.word, allowlist));

  // Hand original-case words to typo.check() so KEEPCASE entries (proper
  // nouns, capitalized contractions) match. Cache by the same checked
  // string so "Liz" and "liz" stay independent — they have different
  // correctness verdicts.
  const checkKeys = new Map<SpellIssue, string>();
  const needLookup = new Set<string>();
  for (const issue of candidates) {
    const key = normalizeApostrophes(issue.word);
    checkKeys.set(issue, key);
    if (!checkCache.has(key)) needLookup.add(key);
  }

  if (needLookup.size > 0) {
    try {
      const batch = Array.from(needLookup);
      const results = await getBackend().check(batch);
      for (const key of batch) {
        checkCache.set(key, results[key] ?? false);
      }
    } catch {
      // Backend unavailable for this pass. We deliberately do NOT poison
      // the cache: writing `true` for every un-checked word would treat
      // real misspellings as valid for the rest of the session even after
      // the backend recovers. Instead, leave them out of the cache so the
      // next lint pass retries, and treat them as "unknown verdict" below
      // (skip rather than flag — better to miss a typo than to scream
      // false positives at the user).
    }
  }

  const diagnostics: Diagnostic[] = [];
  for (const issue of candidates) {
    const key = checkKeys.get(issue)!;
    // Only flag words we have a definitive `false` for. Unknown verdicts
    // (cache miss after a failed backend call) are silently passed.
    if (checkCache.get(key) !== false) continue;

    const message = `"${issue.word}" is not in the English dictionary`;
    diagnostics.push({
      from: issue.from,
      to: issue.to,
      severity: "warning",
      // CodeMirror concatenates markClass onto the built-in lint class string.
      markClass: " cm-spellcheckRange",
      message,
      renderMessage: (view) => renderSpellMessage(view, issue, message),
      source: "spellcheck",
      actions: [buildAddWordAction(issue.word, callbacks)],
    });

    if (diagnostics.length >= maxDiagnostics) break;
  }

  return diagnostics;
}
