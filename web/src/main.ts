import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { initWasm } from "./wasm";
import { downstageHighlighter } from "./downstage-lang";
import { downstageLinter } from "./diagnostics";
import { createPreviewPlugin } from "./preview";
import { setupPdfExport } from "./pdf-export";
import { createScrollSyncPlugin } from "./scroll-sync";

const defaultContent = `Title: The Example Play
Author: Your Name
Date: 2024
Draft: First

# Dramatis Personae

ALICE — A curious young woman
BOB — Her steadfast companion

# The Example Play

## ACT I

### SCENE 1

> A sunny park. Birds sing. A bench sits center stage.

ALICE
(excited)
Have you ever noticed how the world looks different
when you pay **attention** to the *small things*?

BOB
I suppose I haven't given it much thought.

> ALICE crosses to the bench and sits.

ALICE
That's precisely the problem, Bob.
Everyone rushes past the beauty right under their noses.

BOB
And what beauty have you found today?

ALICE
~Everything.~ This park. This bench. This conversation.

===

### SCENE 2

> Evening. The same park, now lit by streetlamps.

SONG 1: The Wanderer's Lament

ALICE
  O, the paths we walk alone
  through the gardens overgrown,
  every stone a stepping place
  to a new and unknown space.

SONG END

BOB
That was beautiful.

ALICE
It's just the beginning.
`;

function getInitialContent(): string {
  const params = new URLSearchParams(window.location.search);
  const encoded = params.get("content");
  if (encoded) {
    try {
      return decodeURIComponent(escape(atob(encoded)));
    } catch {
      // Fall through to default content.
    }
  }
  return defaultContent;
}

let editorView: EditorView | null = null;

async function main() {
  const loading = document.getElementById("loading")!;
  const workspace = document.getElementById("workspace")!;
  const editorPane = document.getElementById("editor-pane")!;
  const iframe = document.getElementById("preview") as HTMLIFrameElement;
  const styleSelect = document.getElementById(
    "style-select",
  ) as HTMLSelectElement;
  const exportBtn = document.getElementById(
    "export-pdf",
  ) as HTMLButtonElement;

  const copyBtn = document.getElementById(
    "copy-source",
  ) as HTMLButtonElement;
  const saveBtn = document.getElementById(
    "save-source",
  ) as HTMLButtonElement;

  await initWasm();

  loading.style.display = "none";
  workspace.classList.remove("hidden");
  exportBtn.disabled = false;
  copyBtn.disabled = false;
  saveBtn.disabled = false;

  const previewPlugin = createPreviewPlugin(
    iframe,
    styleSelect,
    () => editorView,
  );
  const scrollSyncPlugin = createScrollSyncPlugin(iframe);

  const state = EditorState.create({
    doc: getInitialContent(),
    extensions: [
      lineNumbers(),
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      oneDark,
      downstageHighlighter,
      downstageLinter,
      previewPlugin,
      scrollSyncPlugin,
      EditorView.lineWrapping,
    ],
  });

  editorView = new EditorView({
    state,
    parent: editorPane,
  });

  setupPdfExport(exportBtn, styleSelect, () => editorView);

  copyBtn.addEventListener("click", async () => {
    if (!editorView) return;
    const source = editorView.state.doc.toString();
    try {
      await navigator.clipboard.writeText(source);
      const prev = copyBtn.textContent;
      copyBtn.textContent = "Copied";
      setTimeout(() => { copyBtn.textContent = prev; }, 1500);
    } catch {
      copyBtn.textContent = "Failed";
      setTimeout(() => { copyBtn.textContent = "Copy"; }, 1500);
    }
  });

  saveBtn.addEventListener("click", () => {
    if (!editorView) return;
    const source = editorView.state.doc.toString();
    const blob = new Blob([source], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "untitled.ds";
    a.click();
    URL.revokeObjectURL(url);
  });
}

main();
