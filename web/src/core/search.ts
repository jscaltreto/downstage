import { StateEffect, StateField, type Extension, type Text, type Transaction } from "@codemirror/state";
import { Decoration, EditorView, type DecorationSet } from "@codemirror/view";
import { SearchCursor, RegExpCursor } from "@codemirror/search";

export interface SearchOptions {
  query: string;
  caseSensitive: boolean;
  wholeWord: boolean;
  regex: boolean;
}

export interface SearchMatch {
  from: number;
  to: number;
  line: number;
  col: number;
  lineText: string;
}

export type SearchResult =
  | { ok: true; matches: SearchMatch[] }
  | { ok: false; error: string };

export interface SearchState {
  opts: SearchOptions | null;
  matches: SearchMatch[];
  currentIndex: number;
  regexError: string | null;
}

const wordCharRe = /\w/;

function isWordBoundary(doc: Text, from: number, to: number): boolean {
  const before = from > 0 ? doc.sliceString(from - 1, from) : "";
  const after = to < doc.length ? doc.sliceString(to, to + 1) : "";
  const first = doc.sliceString(from, from + 1);
  const last = doc.sliceString(to - 1, to);
  const leftOk = !before || !wordCharRe.test(before) || !wordCharRe.test(first);
  const rightOk = !after || !wordCharRe.test(after) || !wordCharRe.test(last);
  return leftOk && rightOk;
}

function toMatch(doc: Text, from: number, to: number): SearchMatch {
  const lineInfo = doc.lineAt(from);
  return {
    from,
    to,
    line: lineInfo.number,
    col: from - lineInfo.from + 1,
    lineText: lineInfo.text,
  };
}

export function findMatches(doc: Text, opts: SearchOptions): SearchResult {
  if (!opts.query) return { ok: true, matches: [] };

  const matches: SearchMatch[] = [];

  if (opts.regex) {
    let cursor: RegExpCursor;
    try {
      cursor = new RegExpCursor(doc, opts.query, {
        ignoreCase: !opts.caseSensitive,
      });
    } catch (e) {
      return { ok: false, error: e instanceof Error ? e.message : "Invalid regex" };
    }
    while (!cursor.next().done) {
      const { from, to } = cursor.value;
      if (from === to) continue;
      if (opts.wholeWord && !isWordBoundary(doc, from, to)) continue;
      matches.push(toMatch(doc, from, to));
    }
    return { ok: true, matches };
  }

  const normalize = opts.caseSensitive ? undefined : (s: string) => s.toLowerCase();
  const cursor = new SearchCursor(doc, opts.query, 0, doc.length, normalize);
  while (!cursor.next().done) {
    const { from, to } = cursor.value;
    if (opts.wholeWord && !isWordBoundary(doc, from, to)) continue;
    matches.push(toMatch(doc, from, to));
  }
  return { ok: true, matches };
}

export const setSearchEffect = StateEffect.define<SearchOptions | null>();
export const clearSearchEffect = StateEffect.define<null>();
export const selectMatchEffect = StateEffect.define<number>();

const matchMark = Decoration.mark({ class: "cm-search-match" });
const currentMatchMark = Decoration.mark({ class: "cm-search-match-current" });

function buildDecorations(state: SearchState): DecorationSet {
  if (state.matches.length === 0) return Decoration.none;
  const ranges = state.matches.map((m, i) =>
    (i === state.currentIndex ? currentMatchMark : matchMark).range(m.from, m.to),
  );
  return Decoration.set(ranges, true);
}

function reconcileIndex(
  prev: SearchState,
  nextMatches: SearchMatch[],
  tr: Transaction | null,
): number {
  if (nextMatches.length === 0) return -1;
  const prevMatch = prev.matches[prev.currentIndex];
  if (!prevMatch) return 0;
  const mappedFrom = tr ? tr.changes.mapPos(prevMatch.from) : prevMatch.from;
  const exact = nextMatches.findIndex((m) => m.from === mappedFrom);
  if (exact >= 0) return exact;
  const after = nextMatches.findIndex((m) => m.from >= mappedFrom);
  return after >= 0 ? after : nextMatches.length - 1;
}

export const searchStateField = StateField.define<SearchState>({
  create: () => ({ opts: null, matches: [], currentIndex: -1, regexError: null }),
  update(value, tr) {
    let next = value;
    for (const eff of tr.effects) {
      if (eff.is(setSearchEffect)) {
        const opts = eff.value;
        if (!opts || !opts.query) {
          next = { opts, matches: [], currentIndex: -1, regexError: null };
          continue;
        }
        const res = findMatches(tr.state.doc, opts);
        if (!res.ok) {
          next = { opts, matches: [], currentIndex: -1, regexError: res.error };
          continue;
        }
        next = {
          opts,
          matches: res.matches,
          currentIndex: reconcileIndex(value, res.matches, null),
          regexError: null,
        };
      } else if (eff.is(clearSearchEffect)) {
        next = { opts: null, matches: [], currentIndex: -1, regexError: null };
      } else if (eff.is(selectMatchEffect)) {
        if (eff.value >= 0 && eff.value < next.matches.length) {
          next = { ...next, currentIndex: eff.value };
        }
      }
    }

    if (next === value && tr.docChanged && value.opts && value.opts.query) {
      const res = findMatches(tr.state.doc, value.opts);
      if (!res.ok) {
        next = { ...value, matches: [], currentIndex: -1, regexError: res.error };
      } else {
        next = {
          opts: value.opts,
          matches: res.matches,
          currentIndex: reconcileIndex(value, res.matches, tr),
          regexError: null,
        };
      }
    }

    return next;
  },
  provide: (field) => EditorView.decorations.from(field, buildDecorations),
});

export function searchExtension(): Extension {
  return [searchStateField];
}

export function replaceAtMatch(view: EditorView, index: number, replacement: string): boolean {
  const state = view.state.field(searchStateField, false);
  if (!state) return false;
  const match = state.matches[index];
  if (!match) return false;
  view.dispatch({
    changes: { from: match.from, to: match.to, insert: replacement },
    selection: { anchor: match.from + replacement.length },
    userEvent: "input.replace",
    scrollIntoView: true,
  });
  return true;
}

export function replaceAllMatches(view: EditorView, replacement: string): number {
  const state = view.state.field(searchStateField, false);
  if (!state || state.matches.length === 0) return 0;
  const changes = state.matches.map((m) => ({ from: m.from, to: m.to, insert: replacement }));
  const count = state.matches.length;
  view.dispatch({
    changes,
    userEvent: "input.replace.all",
  });
  return count;
}
