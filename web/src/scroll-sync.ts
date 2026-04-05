import { ViewPlugin, type EditorView } from "@codemirror/view";

export function createScrollSyncPlugin(iframe: HTMLIFrameElement) {
  let rafId: ReturnType<typeof requestAnimationFrame> | null = null;

  function syncScroll(view: EditorView) {
    const doc = iframe.contentDocument;
    if (!doc || !doc.body) return;

    const scrollTop = view.scrollDOM.scrollTop;
    const scrollHeight = view.scrollDOM.scrollHeight - view.scrollDOM.clientHeight;

    // If the editor is near the very top or bottom, just match proportionally
    // as a fast path — avoids jitter at extremes.
    if (scrollHeight <= 0) return;
    const ratio = scrollTop / scrollHeight;

    if (ratio < 0.01) {
      doc.documentElement.scrollTop = 0;
      return;
    }
    if (ratio > 0.99) {
      doc.documentElement.scrollTop = doc.documentElement.scrollHeight;
      return;
    }

    // Find the source line at the top of the editor viewport.
    const topBlock = view.elementAtHeight(scrollTop);
    const topLine = view.state.doc.lineAt(topBlock.from).number;

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
      const rect = target.getBoundingClientRect();
      const currentScroll = doc.documentElement.scrollTop;
      doc.documentElement.scrollTop = currentScroll + rect.top;
    }
  }

  return ViewPlugin.fromClass(
    class {
      private handler: () => void;

      constructor(view: EditorView) {
        this.handler = () => {
          if (rafId) cancelAnimationFrame(rafId);
          rafId = requestAnimationFrame(() => {
            syncScroll(view);
            rafId = null;
          });
        };
        view.scrollDOM.addEventListener("scroll", this.handler, {
          passive: true,
        });
      }

      destroy() {
        if (rafId) cancelAnimationFrame(rafId);
      }
    },
  );
}
