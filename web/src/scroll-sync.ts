import { ViewPlugin, type EditorView } from "@codemirror/view";

export function createScrollSyncPlugin(iframe: HTMLIFrameElement) {
  let rafId: ReturnType<typeof requestAnimationFrame> | null = null;

  function syncScroll(view: EditorView) {
    const idoc = iframe.contentDocument;
    if (!idoc?.body) return;

    const scrollEl = idoc.scrollingElement ?? idoc.documentElement;
    const maxEditorScroll =
      view.scrollDOM.scrollHeight - view.scrollDOM.clientHeight;

    if (maxEditorScroll <= 0) return;

    const ratio = view.scrollDOM.scrollTop / maxEditorScroll;

    // At extremes, just use proportional scroll to avoid jitter.
    if (ratio <= 0.02) {
      scrollEl.scrollTop = 0;
      return;
    }
    if (ratio >= 0.98) {
      scrollEl.scrollTop = scrollEl.scrollHeight;
      return;
    }

    // Find the source line at the top of the editor viewport.
    const topBlock = view.elementAtHeight(view.scrollDOM.scrollTop);
    const topLine = view.state.doc.lineAt(topBlock.from).number;

    // Walk the anchored elements to find the best match.
    const els = idoc.querySelectorAll("[data-source-line]");
    if (els.length === 0) return;

    let best: Element | null = null;
    let bestLine = 0;
    for (const el of els) {
      const n = parseInt(el.getAttribute("data-source-line")!, 10);
      if (isNaN(n)) continue;
      if (n <= topLine) {
        best = el;
        bestLine = n;
      } else {
        break;
      }
    }

    if (!best) {
      scrollEl.scrollTop = 0;
      return;
    }

    // Scroll so the matched element sits at the top of the iframe.
    // Use offsetTop which is relative to the document, not the viewport.
    const target = (best as HTMLElement).offsetTop;

    // If there's a next anchored element, interpolate between the two
    // based on how far between bestLine and the next anchor's line we are.
    let nextEl: Element | null = null;
    let nextLine = 0;
    for (const el of els) {
      const n = parseInt(el.getAttribute("data-source-line")!, 10);
      if (n > bestLine) {
        nextEl = el;
        nextLine = n;
        break;
      }
    }

    if (nextEl && nextLine > bestLine) {
      const nextTop = (nextEl as HTMLElement).offsetTop;
      const progress = (topLine - bestLine) / (nextLine - bestLine);
      scrollEl.scrollTop = target + (nextTop - target) * progress;
    } else {
      scrollEl.scrollTop = target;
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
