import { ViewPlugin, type EditorView, type ViewUpdate } from "@codemirror/view";

export function createScrollSyncPlugin(iframe: HTMLIFrameElement) {
  let pending: ReturnType<typeof requestAnimationFrame> | null = null;

  function syncScroll(view: EditorView) {
    const doc = iframe.contentDocument;
    if (!doc) return;

    // Get the 1-based line number at the top of the visible editor area.
    const topPos = view.elementAtHeight(view.scrollDOM.scrollTop);
    const topLine = view.state.doc.lineAt(topPos.from).number;

    const els = doc.querySelectorAll("[data-source-line]");
    let target: Element | null = null;
    for (const el of els) {
      const sourceLine = parseInt(el.getAttribute("data-source-line")!, 10);
      if (sourceLine <= topLine) {
        target = el;
      } else {
        break;
      }
    }

    if (target) {
      target.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  return ViewPlugin.fromClass(
    class {
      constructor(_view: EditorView) {}

      update(update: ViewUpdate) {
        if (!update.geometryChanged) return;
        if (pending) cancelAnimationFrame(pending);
        pending = requestAnimationFrame(() => {
          syncScroll(update.view);
          pending = null;
        });
      }

      destroy() {
        if (pending) cancelAnimationFrame(pending);
      }
    },
  );
}
