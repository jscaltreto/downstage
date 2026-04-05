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

    // Save the current scroll position before replacing content.
    const scrollEl =
      iframe.contentDocument?.scrollingElement ??
      iframe.contentDocument?.documentElement;
    const savedScroll = scrollEl?.scrollTop ?? 0;

    iframe.srcdoc = html;

    // Restore scroll position once the new content loads.
    iframe.addEventListener(
      "load",
      () => {
        const el =
          iframe.contentDocument?.scrollingElement ??
          iframe.contentDocument?.documentElement;
        if (el) el.scrollTop = savedScroll;
      },
      { once: true },
    );
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
