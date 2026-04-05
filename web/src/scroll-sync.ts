import { ViewPlugin, type EditorView } from "@codemirror/view";

function absoluteTop(el: HTMLElement): number {
  let top = 0;
  let current: HTMLElement | null = el;
  while (current) {
    top += current.offsetTop;
    current = current.offsetParent as HTMLElement | null;
  }
  return top;
}

export function createScrollSyncPlugin(iframe: HTMLIFrameElement) {
  let rafId: ReturnType<typeof requestAnimationFrame> | null = null;

  function syncScroll(view: EditorView) {
    let idoc: Document | null = null;
    try {
      idoc = iframe.contentDocument;
    } catch {
      return;
    }
    if (!idoc?.body) return;

    const scrollEl = idoc.scrollingElement ?? idoc.documentElement;
    const maxEditorScroll =
      view.scrollDOM.scrollHeight - view.scrollDOM.clientHeight;
    if (maxEditorScroll <= 0) return;

    const ratio = view.scrollDOM.scrollTop / maxEditorScroll;

    if (ratio <= 0.01) {
      scrollEl.scrollTop = 0;
      return;
    }
    if (ratio >= 0.99) {
      scrollEl.scrollTop = scrollEl.scrollHeight;
      return;
    }

    // Find the 1-based source line at the top of the editor viewport.
    const topBlock = view.elementAtHeight(view.scrollDOM.scrollTop);
    const topLine = view.state.doc.lineAt(topBlock.from).number;

    // Walk anchored elements to find the bracket [best, next).
    const els = Array.from(idoc.querySelectorAll("[data-source-line]"));
    if (els.length === 0) return;

    let bestIdx = 0;
    for (let i = 0; i < els.length; i++) {
      const n = parseInt(els[i].getAttribute("data-source-line")!, 10);
      if (n <= topLine) {
        bestIdx = i;
      } else {
        break;
      }
    }

    const bestEl = els[bestIdx] as HTMLElement;
    const bestLine = parseInt(bestEl.getAttribute("data-source-line")!, 10);
    const bestTop = absoluteTop(bestEl);

    // Interpolate toward the next anchor for smoother tracking.
    const nextEl = els[bestIdx + 1] as HTMLElement | undefined;
    if (nextEl) {
      const nextLine = parseInt(nextEl.getAttribute("data-source-line")!, 10);
      const nextTop = absoluteTop(nextEl);
      if (nextLine > bestLine) {
        const t = Math.min((topLine - bestLine) / (nextLine - bestLine), 1);
        scrollEl.scrollTop = bestTop + (nextTop - bestTop) * t;
        return;
      }
    }

    scrollEl.scrollTop = bestTop;
  }

  return ViewPlugin.fromClass(
    class {
      private handler: () => void;
      private scrollDOM: HTMLElement;

      constructor(view: EditorView) {
        this.scrollDOM = view.scrollDOM;
        this.handler = () => {
          if (rafId) cancelAnimationFrame(rafId);
          rafId = requestAnimationFrame(() => {
            syncScroll(view);
            rafId = null;
          });
        };
        this.scrollDOM.addEventListener("scroll", this.handler, {
          passive: true,
        });
      }

      destroy() {
        this.scrollDOM.removeEventListener("scroll", this.handler);
        if (rafId) cancelAnimationFrame(rafId);
      }
    },
  );
}
