import type { EditorView, ViewUpdate } from "@codemirror/view";
import { ViewPlugin } from "@codemirror/view";
import { renderHTML } from "./wasm";

export function renderPreview(
  iframe: HTMLIFrameElement,
  styleSelect: HTMLSelectElement,
  source: string,
) {
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

export function createPreviewPlugin(
  iframe: HTMLIFrameElement,
  styleSelect: HTMLSelectElement,
  getView: () => EditorView | null,
  isVisible: () => boolean,
) {
  let pending: ReturnType<typeof setTimeout> | null = null;

  styleSelect.addEventListener("change", () => {
    const view = getView();
    if (view && isVisible()) {
      renderPreview(iframe, styleSelect, view.state.doc.toString());
    }
  });

  return ViewPlugin.fromClass(
    class {
      constructor(view: EditorView) {
        if (isVisible()) {
          renderPreview(iframe, styleSelect, view.state.doc.toString());
        }
      }

      update(update: ViewUpdate) {
        if (!update.docChanged) return;
        if (pending) clearTimeout(pending);
        pending = setTimeout(() => {
          if (isVisible()) {
            renderPreview(iframe, styleSelect, update.view.state.doc.toString());
          }
          pending = null;
        }, 300);
      }

      destroy() {
        if (pending) clearTimeout(pending);
      }
    },
  );
}
