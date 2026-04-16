import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { EditorState, Compartment, Transaction } from "@codemirror/state";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { completionKeymap } from "@codemirror/autocomplete";
import { oneDark } from "@codemirror/theme-one-dark";
import { forEachDiagnostic, setDiagnosticsEffect, type Diagnostic } from "@codemirror/lint";
import { createDownstageLinter, refreshDiagnostics } from "../diagnostics";
import { createDownstageCompletion } from "../completion";
import { createDownstageHighlighter } from "../downstage-lang";
import { createScrollSyncPlugin } from "../scroll-sync";
import { warmSpellDictionary } from "../spellcheck";
import { projectDiagnostics } from "./issues";
import {
  clearSearchEffect,
  replaceAllMatches,
  replaceAtMatch,
  searchExtension,
  searchStateField,
  selectMatchEffect,
  setSearchEffect,
  type SearchMatch,
  type SearchOptions,
  type SearchState,
} from "./search";
import type { EditorDiagnostic, EditorEnv } from "./types";

export type SearchMode = "find" | "replace";

export interface SearchSummary {
  total: number;
  index: number;
  error: string | null;
}

const themeCompartment = new Compartment();
const lintCompartment = new Compartment();

// Simple light theme for CodeMirror
const lightTheme = EditorView.theme({
  "&": {
    color: "#1a1a1a",
    backgroundColor: "#fffdf9"
  },
  ".cm-content": {
    caretColor: "#000"
  },
  "&.cm-focused .cm-cursor": {
    borderLeftColor: "#000"
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
    backgroundColor: "rgba(227, 168, 87, 0.3) !important"
  },
  ".cm-gutters": {
    backgroundColor: "#fbf4eb",
    color: "#6e6e6e",
    border: "none"
  },
  ".cm-activeLine": {
    backgroundColor: "rgba(0, 0, 0, 0.02) !important"
  }
}, { dark: false });

export class Engine {
  private view: EditorView | null = null;
  private spellcheckEnabled = false;
  private spellcheckReady = false;
  private cancelSpellcheckWarmup: (() => void) | null = null;

  private lastEmittedSearch: SearchState | null = null;

  constructor(
    private parent: HTMLElement,
    private env: EditorEnv,
    private onDocChange: (content: string, info: { userInput: boolean }) => void,
    private iframe: HTMLIFrameElement,
    private getUserSpellAllowlist: () => string[],
    private addUserSpellAllowlistWord: (word: string) => Promise<boolean>,
    private onDiagnosticsChange: (diagnostics: EditorDiagnostic[]) => void = () => {},
    private onOpenSearch: (mode: SearchMode) => void = () => {},
    private onSearchChange: (summary: SearchSummary, matches: SearchMatch[]) => void = () => {},
    private onAction: (action: string) => void = () => {},
  ) {}

  init(initialContent: string, isDark: boolean, spellcheckEnabled = false) {
    this.spellcheckEnabled = spellcheckEnabled;
    this.spellcheckReady = !spellcheckEnabled;

    const customTheme = EditorView.theme({
      "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
        backgroundColor: "rgba(227, 168, 87, 0.4) !important",
      },
      ".cm-activeLine": {
        backgroundColor: "rgba(255, 255, 255, 0.05) !important",
      },
      ".cm-search-match": {
        backgroundColor: "rgba(227, 168, 87, 0.2)",
        outline: "1px solid rgba(227, 168, 87, 0.45)",
        borderRadius: "2px",
      },
      ".cm-search-match-current": {
        backgroundColor: "rgba(227, 168, 87, 0.55)",
        outline: "1px solid rgba(227, 168, 87, 0.9)",
        borderRadius: "2px",
      },
    });

    const editorKeymap = keymap.of([
      {
        key: "Mod-f",
        preventDefault: true,
        run: () => { this.onOpenSearch("find"); return true; },
      },
      {
        key: "Mod-h",
        preventDefault: true,
        run: () => { this.onOpenSearch("replace"); return true; },
      },
      {
        key: "Mod-Alt-f",
        preventDefault: true,
        run: () => { this.onOpenSearch("replace"); return true; },
      },
      {
        key: "Mod-b",
        preventDefault: true,
        run: () => { this.applyFormat("bold"); return true; },
      },
      {
        key: "Mod-i",
        preventDefault: true,
        run: () => { this.applyFormat("italic"); return true; },
      },
      {
        key: "Mod-u",
        preventDefault: true,
        run: () => { this.applyFormat("underline"); return true; },
      },
      {
        key: "Mod-Shift-p",
        preventDefault: true,
        run: () => { this.onAction("toggle-preview"); return true; },
      },
      {
        key: "Mod-Shift-/",
        preventDefault: true,
        run: () => { this.onAction("toggle-help"); return true; },
      },
    ]);

    this.view = new EditorView({
      state: EditorState.create({
        doc: initialContent,
        extensions: [
          lineNumbers(),
          history(),
          editorKeymap,
          keymap.of([...completionKeymap, ...defaultKeymap, ...historyKeymap]),
          themeCompartment.of(isDark ? oneDark : lightTheme),
          lintCompartment.of(this.createLintExtension()),
          customTheme,
          searchExtension(),
          createDownstageHighlighter(this.env),
          createDownstageCompletion(this.env),
          createScrollSyncPlugin(this.iframe),
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              const userInput = update.transactions.some((tr) => {
                const evt = tr.annotation(Transaction.userEvent);
                return typeof evt === "string" && (evt.startsWith("input") || evt.startsWith("delete"));
              });
              this.onDocChange(update.state.doc.toString(), { userInput });
            }
            const lintChanged = update.transactions.some((tr) =>
              tr.effects.some((effect) => effect.is(setDiagnosticsEffect)),
            );
            if (lintChanged) {
              this.emitDiagnostics();
            }
            this.maybeEmitSearch();
          }),
        ],
      }),
      parent: this.parent,
    });

    if (spellcheckEnabled) {
      this.scheduleSpellcheckWarmup();
    }
  }

  applyFormat(action: string) {
    if (!this.view) return;
    
    const applyWrap = (before: string, after: string, fallback: string) => {
        const selection = this.view!.state.selection.main;
        const from = selection.from;
        const to = selection.to;
        const selected = this.view!.state.sliceDoc(from, to);
        
        if (selected.startsWith(before) && selected.endsWith(after)) {
            this.view!.dispatch({
                changes: { 
                    from, 
                    to, 
                    insert: selected.substring(before.length, selected.length - after.length) 
                },
                selection: { anchor: from, head: from + selected.length - before.length - after.length },
                scrollIntoView: true,
            });
            return;
        }

        const beforeRange = this.view!.state.sliceDoc(from - before.length, from);
        const afterRange = this.view!.state.sliceDoc(to, to + after.length);
        if (beforeRange === before && afterRange === after) {
            this.view!.dispatch({
                changes: { 
                    from: from - before.length, 
                    to: to + after.length, 
                    insert: selected 
                },
                selection: { anchor: from - before.length, head: from - before.length + selected.length },
                scrollIntoView: true,
            });
            return;
        }

        const content = selected || fallback;
        this.view!.dispatch({
            changes: { from, to, insert: `${before}${content}${after}` },
            selection: { anchor: from + before.length, head: from + before.length + content.length },
            scrollIntoView: true,
        });
        this.view!.focus();
    };

    const applySnippet = (snippet: string, offset: number) => {
        const selection = this.view!.state.selection.main;
        this.view!.dispatch({
            changes: { from: selection.from, to: selection.to, insert: snippet },
            selection: { anchor: selection.from + offset },
            scrollIntoView: true,
        });
        this.view!.focus();
    };

    switch (action) {
      case "bold": applyWrap("**", "**", "bold text"); break;
      case "italic": applyWrap("*", "*", "italic text"); break;
      case "underline": applyWrap("_", "_", "underlined text"); break;
      case "cue": applySnippet("\nCHARACTER\nDialogue here.\n", 11); break;
      case "direction": applySnippet("\n> Stage direction.\n", 3); break;
      case "act": applySnippet("\n## ACT I\n", 4); break;
      case "scene": applySnippet("\n### SCENE 1\n", 5); break;
      case "song": applySnippet("\nSONG 1: Song Title\n\nCHARACTER\n  Lyric line one.\n\nSONG END\n", 9); break;
      case "page-break": applySnippet("\n===\n", 1); break;
    }
  }

  setTheme(isDark: boolean) {
    if (!this.view) return;
    this.view.dispatch({
      effects: themeCompartment.reconfigure(isDark ? oneDark : lightTheme),
    });
  }

  setSpellcheckEnabled(enabled: boolean) {
    this.spellcheckEnabled = enabled;
    if (!this.view) return;

    if (!enabled) {
      this.clearSpellcheckWarmup();
      this.spellcheckReady = false;
      this.refreshDiagnostics();
      return;
    }

    this.scheduleSpellcheckWarmup();
  }

  refreshDiagnostics() {
    if (!this.view) return;
    refreshDiagnostics(this.view);
  }

  getDiagnostics(): EditorDiagnostic[] {
    if (!this.view) return [];
    const raw: Diagnostic[] = [];
    forEachDiagnostic(this.view.state, (d) => {
      raw.push(d);
    });
    return projectDiagnostics(this.view.state.doc, raw);
  }

  revealDiagnostic(from: number, to: number) {
    if (!this.view) return;
    const docLength = this.view.state.doc.length;
    const clampedFrom = Math.max(0, Math.min(from, docLength));
    const clampedTo = Math.max(clampedFrom, Math.min(to, docLength));
    this.view.dispatch({
      selection: { anchor: clampedFrom, head: clampedTo },
      effects: EditorView.scrollIntoView(clampedFrom, { y: "center" }),
    });
    this.view.focus();
  }

  revealPosition(line: number, character: number) {
    if (!this.view) return;
    const doc = this.view.state.doc;
    const lineNumber = Math.max(1, Math.min(line + 1, doc.lines));
    const lineInfo = doc.line(lineNumber);
    const offset = Math.min(lineInfo.from + Math.max(0, character), lineInfo.to);
    this.view.dispatch({
      selection: { anchor: offset, head: offset },
      effects: EditorView.scrollIntoView(offset, { y: "center" }),
    });
    this.view.focus();
  }

  private emitDiagnostics() {
    this.onDiagnosticsChange(this.getDiagnostics());
  }

  private maybeEmitSearch() {
    if (!this.view) return;
    const state = this.view.state.field(searchStateField, false);
    if (!state) return;
    if (state === this.lastEmittedSearch) return;
    this.lastEmittedSearch = state;
    this.onSearchChange(
      { total: state.matches.length, index: state.currentIndex, error: state.regexError },
      state.matches,
    );
  }

  setSearch(opts: SearchOptions): SearchSummary {
    if (!this.view) return { total: 0, index: -1, error: null };
    this.view.dispatch({ effects: setSearchEffect.of(opts) });
    const state = this.view.state.field(searchStateField);
    if (state.matches.length > 0 && state.currentIndex >= 0) {
      const active = state.matches[state.currentIndex];
      this.view.dispatch({ effects: EditorView.scrollIntoView(active.from, { y: "nearest" }) });
    }
    return { total: state.matches.length, index: state.currentIndex, error: state.regexError };
  }

  findNext(): SearchSummary {
    return this.step(1);
  }

  findPrev(): SearchSummary {
    return this.step(-1);
  }

  private step(direction: 1 | -1): SearchSummary {
    if (!this.view) return { total: 0, index: -1, error: null };
    const state = this.view.state.field(searchStateField);
    if (state.matches.length === 0) {
      return { total: 0, index: -1, error: state.regexError };
    }
    const total = state.matches.length;
    const currentValid = state.currentIndex >= 0;
    const nextIndex = currentValid
      ? (state.currentIndex + direction + total) % total
      : direction === 1
        ? 0
        : total - 1;
    this.selectMatch(nextIndex);
    return { total, index: nextIndex, error: null };
  }

  selectMatch(index: number): SearchSummary {
    if (!this.view) return { total: 0, index: -1, error: null };
    const state = this.view.state.field(searchStateField);
    if (index < 0 || index >= state.matches.length) {
      return { total: state.matches.length, index: state.currentIndex, error: state.regexError };
    }
    const match = state.matches[index];
    this.view.dispatch({
      effects: [
        selectMatchEffect.of(index),
        EditorView.scrollIntoView(match.from, { y: "center" }),
      ],
      selection: { anchor: match.from, head: match.to },
    });
    return { total: state.matches.length, index, error: null };
  }

  replaceCurrent(replacement: string): SearchSummary {
    if (!this.view) return { total: 0, index: -1, error: null };
    const state = this.view.state.field(searchStateField);
    if (state.currentIndex < 0 || state.matches.length === 0) {
      return { total: state.matches.length, index: state.currentIndex, error: state.regexError };
    }
    replaceAtMatch(this.view, state.currentIndex, replacement);
    const after = this.view.state.field(searchStateField);
    if (after.matches.length > 0) {
      const targetIndex = Math.min(state.currentIndex, after.matches.length - 1);
      this.selectMatch(targetIndex);
      return { total: after.matches.length, index: targetIndex, error: null };
    }
    return { total: 0, index: -1, error: null };
  }

  replaceAll(replacement: string): number {
    if (!this.view) return 0;
    return replaceAllMatches(this.view, replacement);
  }

  clearSearch() {
    if (!this.view) return;
    this.view.dispatch({ effects: clearSearchEffect.of(null) });
  }

  getSelectionText(): string {
    if (!this.view) return "";
    const sel = this.view.state.selection.main;
    if (sel.empty) return "";
    return this.view.state.sliceDoc(sel.from, sel.to);
  }

  getSearchMatches(): SearchMatch[] {
    if (!this.view) return [];
    const state = this.view.state.field(searchStateField, false);
    return state ? state.matches : [];
  }

  setContent(content: string) {
    if (!this.view) return;
    if (this.getContent() === content) return;
    this.view.dispatch({
      changes: { from: 0, to: this.view.state.doc.length, insert: content },
      scrollIntoView: true,
    });
  }

  getContent(): string {
    return this.view?.state.doc.toString() || "";
  }

  focus() {
    this.view?.focus();
  }

  destroy() {
    this.clearSpellcheckWarmup();
    this.view?.destroy();
  }

  private createLintExtension() {
    return createDownstageLinter(this.env, () => this.spellcheckEnabled && this.spellcheckReady, {
      getUserAllowlist: this.getUserSpellAllowlist,
      addWord: async (word) => {
        const added = await this.addUserSpellAllowlistWord(word);
        if (added) {
          this.refreshDiagnostics();
        }
        return added;
      },
    });
  }

  private scheduleSpellcheckWarmup() {
    this.clearSpellcheckWarmup();
    this.spellcheckReady = false;

    let cancelled = false;
    const activate = async () => {
      this.cancelSpellcheckWarmup = null;
      if (cancelled || !this.view || !this.spellcheckEnabled) return;

      try {
        await warmSpellDictionary();
      } catch {
        return;
      }
      if (cancelled || !this.view || !this.spellcheckEnabled) return;

      this.spellcheckReady = true;
      this.refreshDiagnostics();
    };

    if (typeof window.requestIdleCallback === "function") {
      const callbackId = window.requestIdleCallback(() => { void activate(); }, { timeout: 500 });
      this.cancelSpellcheckWarmup = () => {
        cancelled = true;
        window.cancelIdleCallback(callbackId);
      };
      return;
    }

    const timerId = globalThis.setTimeout(() => { void activate(); }, 300);
    this.cancelSpellcheckWarmup = () => {
      cancelled = true;
      globalThis.clearTimeout(timerId);
    };
  }

  private clearSpellcheckWarmup() {
    this.cancelSpellcheckWarmup?.();
    this.cancelSpellcheckWarmup = null;
  }

}
