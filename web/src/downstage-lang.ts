import {
  Decoration,
  type DecorationSet,
  type EditorView,
  ViewPlugin,
  type ViewUpdate,
} from "@codemirror/view";
import { RangeSetBuilder } from "@codemirror/state";
import { semanticTokens } from "./wasm";

const tokenClassMap: Record<string, string> = {
  namespace: "cm-ds-namespace",
  type: "cm-ds-type",
  comment: "cm-ds-comment",
  keyword: "cm-ds-keyword",
  property: "cm-ds-property",
  string: "cm-ds-string",
};

const tokenTypeNames = [
  "namespace",
  "type",
  "comment",
  "keyword",
  "property",
  "string",
];

function buildDecorations(view: EditorView): DecorationSet {
  const doc = view.state.doc;
  const source = doc.toString();
  const tokens = semanticTokens(source);

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

    const typeName = tokenTypeNames[tokenType];
    const className = tokenClassMap[typeName];
    if (!className) continue;

    // Convert 0-based line/col to CM absolute position.
    // Lines in CM are 1-based.
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

  // RangeSetBuilder requires sorted, non-overlapping ranges.
  decos.sort((a, b) => a.from - b.from || a.to - b.to);
  for (const d of decos) {
    builder.add(d.from, d.to, d.deco);
  }

  return builder.finish();
}

export const downstageHighlighter = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    pending: ReturnType<typeof setTimeout> | null = null;

    constructor(view: EditorView) {
      this.decorations = buildDecorations(view);
    }

    update(update: ViewUpdate) {
      if (!update.docChanged) return;

      if (this.pending) clearTimeout(this.pending);
      this.pending = setTimeout(() => {
        this.decorations = buildDecorations(update.view);
        update.view.dispatch({ effects: [] }); // trigger redraw
        this.pending = null;
      }, 150);
    }

    destroy() {
      if (this.pending) clearTimeout(this.pending);
    }
  },
  { decorations: (v) => v.decorations },
);
