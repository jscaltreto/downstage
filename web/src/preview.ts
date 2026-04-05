import type { EditorView, ViewUpdate } from "@codemirror/view";
import { ViewPlugin } from "@codemirror/view";
import { renderHTML } from "./wasm";

export function createPreviewPlugin(
  iframe: HTMLIFrameElement,
  styleSelect: HTMLSelectElement,
  getView: () => EditorView | null,
) {
  let pending: ReturnType<typeof setTimeout> | null = null;

  function updatePreview(source: string) {
    const style = styleSelect.value;
    const html = renderHTML(source, style);
    iframe.srcdoc = html;
  }

  styleSelect.addEventListener("change", () => {
    const view = getView();
    if (view) updatePreview(view.state.doc.toString());
  });

  return ViewPlugin.fromClass(
    class {
      constructor(view: EditorView) {
        updatePreview(view.state.doc.toString());
      }

      update(update: ViewUpdate) {
        if (!update.docChanged) return;
        if (pending) clearTimeout(pending);
        pending = setTimeout(() => {
          updatePreview(update.view.state.doc.toString());
          pending = null;
        }, 300);
      }

      destroy() {
        if (pending) clearTimeout(pending);
      }
    },
  );
}
