import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { EditorState, Compartment } from "@codemirror/state";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { completionKeymap } from "@codemirror/autocomplete";
import { oneDark } from "@codemirror/theme-one-dark";
import { createDownstageLinter } from "../diagnostics";
import { createDownstageCompletion } from "../completion";
import { createDownstageHighlighter } from "../downstage-lang";
import { createScrollSyncPlugin } from "../scroll-sync";
import type { EditorEnv } from "./types";

const themeCompartment = new Compartment();

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

  constructor(
    private parent: HTMLElement,
    private env: EditorEnv,
    private onDocChange: (content: string) => void,
    private iframe: HTMLIFrameElement,
  ) {}

  init(initialContent: string, isDark: boolean) {
    const customTheme = EditorView.theme({
      "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
        backgroundColor: "rgba(227, 168, 87, 0.4) !important",
      },
      ".cm-activeLine": {
        backgroundColor: "rgba(255, 255, 255, 0.05) !important",
      },
    });

    this.view = new EditorView({
      state: EditorState.create({
        doc: initialContent,
        extensions: [
          lineNumbers(),
          history(),
          keymap.of([...completionKeymap, ...defaultKeymap, ...historyKeymap]),
          themeCompartment.of(isDark ? oneDark : lightTheme),
          customTheme,
          createDownstageHighlighter(this.env),
          createDownstageCompletion(this.env),
          createDownstageLinter(this.env),
          createScrollSyncPlugin(this.iframe),
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              this.onDocChange(update.state.doc.toString());
            }
          }),
        ],
      }),
      parent: this.parent,
    });
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
    this.view?.destroy();
  }
}
