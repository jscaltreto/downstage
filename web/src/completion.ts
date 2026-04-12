import {
  autocompletion,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from "@codemirror/autocomplete";
import type { EditorView } from "@codemirror/view";
import type { EditorEnv, LSPCompletionItem } from "./core/types";
import { offsetFromLSP } from "./lsp-offsets";

// LSP CompletionItemKind values we map onto CodeMirror completion types.
const kindToType: Record<number, string> = {
  6: "variable",
  14: "keyword",
};

function toCompletion(item: LSPCompletionItem): Completion {
  const label = item.label;
  return {
    label,
    detail: item.detail,
    type: item.kind !== undefined ? kindToType[item.kind] : undefined,
    boost: 0,
    apply: (view: EditorView) => {
      const edit = item.textEdit;
      if (edit) {
        const from = offsetFromLSP(view.state.doc, edit.range.start.line, edit.range.start.character);
        const to = offsetFromLSP(view.state.doc, edit.range.end.line, edit.range.end.character);
        view.dispatch({
          changes: { from, to, insert: edit.newText },
          selection: { anchor: from + edit.newText.length },
          scrollIntoView: true,
        });
        return;
      }
      const insert = item.insertText ?? label;
      const { from, to } = view.state.selection.main;
      view.dispatch({
        changes: { from, to, insert },
        selection: { anchor: from + insert.length },
        scrollIntoView: true,
      });
    },
  };
}

export function createDownstageCompletion(env: EditorEnv) {
  async function source(context: CompletionContext): Promise<CompletionResult | null> {
    const { state, pos } = context;
    const line = state.doc.lineAt(pos);
    const lspLine = line.number - 1;
    const character = pos - line.from;

    let list;
    try {
      list = await env.completion(state.doc.toString(), lspLine, character);
    } catch {
      return null;
    }

    if (!list || !list.items || list.items.length === 0) {
      return null;
    }

    const items = list.items.slice().sort((a, b) => {
      const sa = a.sortText ?? a.label;
      const sb = b.sortText ?? b.label;
      return sa.localeCompare(sb);
    });

    return {
      from: line.from,
      to: pos,
      options: items.map(toCompletion),
      filter: false,
    };
  }

  return autocompletion({
    activateOnTyping: true,
    override: [source],
  });
}
