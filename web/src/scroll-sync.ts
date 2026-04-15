import { ViewPlugin, type EditorView, type ViewUpdate } from "@codemirror/view";

function absoluteTop(el: HTMLElement): number {
  let top = 0;
  let current: HTMLElement | null = el;
  while (current) {
    top += current.offsetTop;
    current = current.offsetParent as HTMLElement | null;
  }
  return top;
}

export type SyncMode = 'center' | 'top';

export function createScrollSyncPlugin(iframe: HTMLIFrameElement) {
  let rafId: ReturnType<typeof requestAnimationFrame> | null = null;
  let lastSyncLine = -1;
  let lastSyncMode: SyncMode | null = null;
  let lastDocument: Document | null = null;

  function topLineAtScroll(view: EditorView): number {
    const topBlock = view.elementAtHeight(view.scrollDOM.scrollTop);
    return view.state.doc.lineAt(topBlock.from).number;
  }

  function syncToLine(view: EditorView, lineNumber: number, mode: SyncMode) {
    let idoc: Document | null = null;
    try {
      idoc = iframe.contentDocument;
    } catch {
      return;
    }
    if (!idoc?.body) return;

    if (idoc !== lastDocument) {
        lastDocument = idoc;
        lastSyncLine = -1;
        lastSyncMode = null;
    }

    if (lineNumber === lastSyncLine && mode === lastSyncMode) return;
    
    const els = Array.from(idoc.querySelectorAll("[data-source-line]"));
    if (els.length === 0) return;

    let bestIdx = -1;
    for (let i = 0; i < els.length; i++) {
      const el = els[i] as HTMLElement;
      const n = parseInt(el.getAttribute("data-source-line")!, 10);
      if (n <= lineNumber) {
        bestIdx = i;
      } else {
        break;
      }
    }

    if (bestIdx === -1) return;
    const bestEl = els[bestIdx] as HTMLElement;
    
    lastSyncLine = lineNumber;
    lastSyncMode = mode;

    const scrollEl = idoc.scrollingElement ?? idoc.documentElement;

    if (mode === 'center') {
        bestEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    } else {
        const bestTop = absoluteTop(bestEl);
        const bestLine = parseInt(bestEl.getAttribute("data-source-line")!, 10);
        const nextEl = els[bestIdx + 1] as HTMLElement | undefined;
        
        if (nextEl) {
            const nextLine = parseInt(nextEl.getAttribute("data-source-line")!, 10);
            const nextTop = absoluteTop(nextEl);
            if (nextLine > bestLine) {
                const t = Math.min((lineNumber - bestLine) / (nextLine - bestLine), 1);
                scrollEl.scrollTop = bestTop + (nextTop - bestTop) * t;
                return;
            }
        }
        scrollEl.scrollTop = bestTop;
    }
  }

  return ViewPlugin.fromClass(
    class {
      private scrollHandler: () => void;
      private scrollDOM: HTMLElement;
      private resizeObserver: ResizeObserver | null = null;

      constructor(private view: EditorView) {
        this.scrollDOM = view.scrollDOM;

        const resyncTop = () => {
          if (rafId) cancelAnimationFrame(rafId);
          rafId = requestAnimationFrame(() => {
            const topLine = topLineAtScroll(view);
            // Force a re-sync after a viewport resize so the preview's scrollTop is recomputed.
            lastSyncLine = -1;
            lastSyncMode = null;
            syncToLine(view, topLine, 'top');
            rafId = null;
          });
        };

        this.scrollHandler = () => {
          if (rafId) cancelAnimationFrame(rafId);
          rafId = requestAnimationFrame(() => {
            const topLine = topLineAtScroll(view);
            syncToLine(view, topLine, 'top');
            rafId = null;
          });
        };

        this.scrollDOM.addEventListener("scroll", this.scrollHandler, {
          passive: true,
        });

        if (typeof ResizeObserver !== 'undefined') {
          this.resizeObserver = new ResizeObserver(() => resyncTop());
          this.resizeObserver.observe(this.scrollDOM);
        }
      }

      update(update: ViewUpdate) {
        if (update.selectionSet || (update.docChanged && update.view.hasFocus)) {
          const head = update.state.selection.main.head;
          const line = update.state.doc.lineAt(head).number;
          syncToLine(this.view, line, 'center');
        }
      }

      destroy() {
        this.scrollDOM.removeEventListener("scroll", this.scrollHandler);
        this.resizeObserver?.disconnect();
        this.resizeObserver = null;
        if (rafId) cancelAnimationFrame(rafId);
      }
    },
  );
}
