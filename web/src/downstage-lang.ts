import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from "@codemirror/view";
import { RangeSetBuilder, StateEffect, StateField } from "@codemirror/state";
import { semanticTokens, tokenTypeNames } from "./wasm";

const tokenClassMap: Record<string, string> = {
  namespace: "cm-ds-namespace",
  type: "cm-ds-type",
  comment: "cm-ds-comment",
  keyword: "cm-ds-keyword",
  property: "cm-ds-property",
  string: "cm-ds-string",
};

// Cached at init time from the WASM bridge.
let cachedTypeNames: string[] | null = null;
function getTypeNames(): string[] {
  if (!cachedTypeNames) cachedTypeNames = tokenTypeNames();
  return cachedTypeNames;
}

function buildDecorations(view: EditorView): DecorationSet {
  const doc = view.state.doc;
  const source = doc.toString();
  const tokens = semanticTokens(source);
  const typeNames = getTypeNames();

  const builder = new RangeSetBuilder<Decoration>();
  const decos: { from: number; to: number; deco: Decoration }[] = [];

  let line = 0;
  let col = 0;

  for (let i = 0; i < tokens.length; i += 5) {
    const deltaLine = tokens[i];
    const deltaCol = tokens[i + 1];
    const length = tokens[i + 2];
    const tokenType = tokens[i + 3];

    line += deltaLine;
    if (deltaLine > 0) {
      col = deltaCol;
    } else {
      col += deltaCol;
    }

    const typeName = typeNames[tokenType];
    const className = tokenClassMap[typeName];
    if (!className) continue;

    if (line + 1 > doc.lines) continue;
    const lineObj = doc.line(line + 1);
    const from = lineObj.from + col;
    const to = Math.min(from + length, lineObj.to);

    if (from >= to || from < lineObj.from) continue;

    decos.push({
      from,
      to,
      deco: Decoration.mark({ class: className }),
    });
  }

  decos.sort((a, b) => a.from - b.from || a.to - b.to);
  for (const d of decos) {
    builder.add(d.from, d.to, d.deco);
  }

  return builder.finish();
}

const refreshHighlights = StateEffect.define<DecorationSet>();

const highlightField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(value, tr) {
    for (const e of tr.effects) {
      if (e.is(refreshHighlights)) return e.value;
    }
    return tr.docChanged ? Decoration.none : value;
  },
  provide: (f) => EditorView.decorations.from(f),
});

const highlightPlugin = ViewPlugin.fromClass(
  class {
    pending: ReturnType<typeof setTimeout> | null = null;

    constructor(view: EditorView) {
      this.scheduleUpdate(view, 0);
    }

    update(update: ViewUpdate) {
      if (!update.docChanged) return;
      this.scheduleUpdate(update.view, 150);
    }

    scheduleUpdate(view: EditorView, delay: number) {
      if (this.pending) clearTimeout(this.pending);
      this.pending = setTimeout(() => {
        const decos = buildDecorations(view);
        view.dispatch({ effects: refreshHighlights.of(decos) });
        this.pending = null;
      }, delay);
    }

    destroy() {
      if (this.pending) clearTimeout(this.pending);
    }
  },
);

export const downstageHighlighter = [highlightField, highlightPlugin];
