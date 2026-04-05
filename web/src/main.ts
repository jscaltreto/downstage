import "../style.css";
import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { initWasm } from "./wasm";
import { downstageHighlighter } from "./downstage-lang";
import { downstageLinter } from "./diagnostics";
import { createPreviewPlugin, renderPreview } from "./preview";
import { setupPdfExport } from "./pdf-export";
import { createScrollSyncPlugin } from "./scroll-sync";

declare const __APP_VERSION__: string;

const cheatSheetStorageKey = "downstage-editor-cheat-sheet-hidden";
const previewStorageKey = "downstage-editor-preview-hidden";
const draftsStorageKey = "downstage-editor-drafts";
const activeDraftStorageKey = "downstage-editor-active-draft";

const exampleContent = `Title: The Example Play
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

type ToolbarAction =
  | "bold"
  | "italic"
  | "underline"
  | "cue"
  | "direction"
  | "act"
  | "scene"
  | "song"
  | "page-break";

interface SavedDraft {
  id: string;
  title: string;
  content: string;
  updatedAt: string;
}

const octiconChevronRight = `<svg viewBox="0 0 16 16" aria-hidden="true" focusable="false"><path d="M5.22 3.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06L6.28 12.78a.75.75 0 1 1-1.06-1.06L8.94 8 5.22 4.28a.75.75 0 0 1 0-1.06Z"></path></svg>`;
const octiconTrash = `<svg viewBox="0 0 16 16" aria-hidden="true" focusable="false"><path d="M6.5 1.75a.75.75 0 0 1 .75-.75h1.5a.75.75 0 0 1 .75.75V3h3.25a.75.75 0 0 1 0 1.5h-.538l-.613 8.177A1.75 1.75 0 0 1 9.859 14.5H6.14a1.75 1.75 0 0 1-1.745-1.823L3.782 4.5H3.25a.75.75 0 0 1 0-1.5H6.5V1.75Zm1.5.75v.5h1v-.5h-1Zm-2.103 2 .594 7.927a.25.25 0 0 0 .249.068H9.86a.25.25 0 0 0 .249-.233L10.703 4.5H5.897ZM6.75 6a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 6.75 6Zm2.5 0a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 9.25 6Z"></path></svg>`;
const octiconSidebarCollapse = `<svg viewBox="0 0 16 16" aria-hidden="true" focusable="false"><path d="M9.78 3.22a.75.75 0 0 1 0 1.06L6.06 8l3.72 3.72a.75.75 0 1 1-1.06 1.06L4.47 8.53a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z"></path></svg>`;
const octiconSidebarExpand = `<svg viewBox="0 0 16 16" aria-hidden="true" focusable="false"><path d="M6.22 3.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 1 1-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 0 1 0-1.06Z"></path></svg>`;

function buildNewPlayTemplate(): string {
  const currentYear = new Date().getFullYear();

  return `Title: Untitled Play
Author: Your Name
Date: ${currentYear}
Draft: First

# Dramatis Personae

PROTAGONIST — Add your cast here

# Untitled Play

## ACT I

### SCENE 1

> Describe the setting here.

PROTAGONIST
Write your opening lines here.
`;
}

function getUrlContent(): string | null {
  const params = new URLSearchParams(window.location.search);
  const encoded = params.get("content");
  if (encoded) {
    try {
      return decodeURIComponent(escape(atob(encoded)));
    } catch {
      return null;
    }
  }
  return null;
}

function extractDraftTitle(content: string): string {
  const match = content.match(/^Title:\s*(.+)$/m);
  const title = match?.[1]?.trim();
  return title && title.length > 0 ? title : "Untitled Play";
}

function generateDraftId(): string {
  return `draft-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function buildSavedDraft(content: string): SavedDraft {
  return {
    id: generateDraftId(),
    title: extractDraftTitle(content),
    content,
    updatedAt: new Date().toISOString(),
  };
}

function readSavedDrafts(): SavedDraft[] {
  try {
    const raw = localStorage.getItem(draftsStorageKey);
    if (!raw) {
      return [];
    }

    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.filter((draft): draft is SavedDraft => (
      typeof draft?.id === "string"
      && typeof draft?.title === "string"
      && typeof draft?.content === "string"
      && typeof draft?.updatedAt === "string"
    ));
  } catch {
    return [];
  }
}

function writeSavedDrafts(drafts: SavedDraft[]) {
  try {
    localStorage.setItem(draftsStorageKey, JSON.stringify(drafts));
  } catch {
    // Ignore local storage failures so editing still works.
  }
}

function readActiveDraftId(): string | null {
  try {
    return localStorage.getItem(activeDraftStorageKey);
  } catch {
    return null;
  }
}

function writeActiveDraftId(draftId: string) {
  try {
    localStorage.setItem(activeDraftStorageKey, draftId);
  } catch {
    // Ignore local storage failures so editing still works.
  }
}

function sortDrafts(drafts: SavedDraft[]): SavedDraft[] {
  return [...drafts].sort((left, right) => right.updatedAt.localeCompare(left.updatedAt));
}

function initializeDrafts(): { drafts: SavedDraft[]; activeDraftId: string } {
  let drafts = readSavedDrafts();
  const urlContent = getUrlContent();

  if (urlContent) {
    const existingDraft = drafts.find((draft) => draft.content === urlContent);
    if (existingDraft) {
      writeActiveDraftId(existingDraft.id);
      return { drafts, activeDraftId: existingDraft.id };
    }

    const importedDraft = buildSavedDraft(urlContent);
    drafts = [importedDraft, ...drafts];
    writeSavedDrafts(drafts);
    writeActiveDraftId(importedDraft.id);
    return { drafts, activeDraftId: importedDraft.id };
  }

  if (drafts.length === 0) {
    const firstDraft = buildSavedDraft(exampleContent);
    drafts = [firstDraft];
    writeSavedDrafts(drafts);
    writeActiveDraftId(firstDraft.id);
    return { drafts, activeDraftId: firstDraft.id };
  }

  const activeDraftId = readActiveDraftId();
  const activeDraft = drafts.find((draft) => draft.id === activeDraftId);
  if (activeDraft) {
    return { drafts, activeDraftId: activeDraft.id };
  }

  const newestDraft = sortDrafts(drafts)[0];
  writeActiveDraftId(newestDraft.id);
  return { drafts, activeDraftId: newestDraft.id };
}

let editorView: EditorView | null = null;

function getEditorContent(): string {
  return editorView?.state.doc.toString() ?? "";
}

function setEditorContent(content: string) {
  if (!editorView) {
    return;
  }

  editorView.dispatch({
    changes: {
      from: 0,
      to: editorView.state.doc.length,
      insert: content,
    },
    selection: { anchor: 0 },
    scrollIntoView: true,
  });
  editorView.focus();
}

async function confirmAction(
  dialog: HTMLDialogElement,
  title: HTMLElement,
  message: HTMLElement,
  acceptButton: HTMLButtonElement,
  nextTitle: string,
  nextMessage: string,
  acceptLabel: string,
): Promise<boolean> {
  title.textContent = nextTitle;
  message.textContent = nextMessage;
  acceptButton.textContent = acceptLabel;

  dialog.showModal();

  return new Promise((resolve) => {
    const handleClose = () => {
      dialog.removeEventListener("close", handleClose);
      resolve(dialog.returnValue === "confirm");
    };

    dialog.addEventListener("close", handleClose, { once: true });
  });
}

function applyWrap(before: string, after: string, fallback: string) {
  if (!editorView) {
    return;
  }

  const selection = editorView.state.selection.main;
  const selected = editorView.state.sliceDoc(selection.from, selection.to);

  if (!selection.empty) {
    const wrappedStart = selection.from - before.length;
    const wrappedEnd = selection.to + after.length;

    if (wrappedStart >= 0) {
      const surrounding = editorView.state.sliceDoc(wrappedStart, wrappedEnd);
      const expected = `${before}${selected}${after}`;

      if (surrounding === expected) {
        editorView.dispatch({
          changes: [
            { from: selection.to, to: wrappedEnd, insert: "" },
            { from: wrappedStart, to: selection.from, insert: "" },
          ],
          selection: {
            anchor: wrappedStart,
            head: wrappedStart + selected.length,
          },
          scrollIntoView: true,
        });
        editorView.focus();
        return;
      }
    }
  }

  const content = selected || fallback;
  const insert = `${before}${content}${after}`;
  const cursorStart = selection.from + before.length;
  const cursorEnd = cursorStart + content.length;

  editorView.dispatch({
    changes: { from: selection.from, to: selection.to, insert },
    selection: { anchor: cursorStart, head: cursorEnd },
    scrollIntoView: true,
  });
  editorView.focus();
}

function applySnippet(snippet: string, cursorOffset = snippet.length) {
  if (!editorView) {
    return;
  }

  const selection = editorView.state.selection.main;
  const insert = selection.empty
    ? snippet
    : `${snippet}${editorView.state.sliceDoc(selection.from, selection.to)}`;
  const anchor = selection.from + cursorOffset;

  editorView.dispatch({
    changes: { from: selection.from, to: selection.to, insert },
    selection: { anchor },
    scrollIntoView: true,
  });
  editorView.focus();
}

function applyToolbarAction(action: ToolbarAction) {
  switch (action) {
    case "bold":
      applyWrap("**", "**", "bold text");
      break;
    case "italic":
      applyWrap("*", "*", "italic text");
      break;
    case "underline":
      applyWrap("_", "_", "underlined text");
      break;
    case "cue":
      applySnippet("\nCHARACTER\nDialogue here.\n", "\nCHARACTER\n".length);
      break;
    case "direction":
      applySnippet("\n> Stage direction.\n", "\n> ".length);
      break;
    case "act":
      applySnippet("\n## ACT I\n", "\n## ".length);
      break;
    case "scene":
      applySnippet("\n### SCENE 1\n", "\n### ".length);
      break;
    case "song":
      applySnippet(
        "\nSONG 1: Song Title\n\nCHARACTER\n  Lyric line one.\n  Lyric line two.\n\nSONG END\n",
        "\nSONG 1: ".length,
      );
      break;
    case "page-break":
      applySnippet("\n===\n", "\n".length);
      break;
  }
}

function reopenDialog(dialog: HTMLDialogElement) {
  requestAnimationFrame(() => {
    if (!dialog.open) {
      dialog.showModal();
    }
  });
}

async function main() {
  const loading = document.getElementById("loading")!;
  const workspace = document.getElementById("workspace")!;
  const editorPane = document.getElementById("editor-pane")!;
  const iframe = document.getElementById("preview") as HTMLIFrameElement;
  const workspaceEl = document.getElementById("workspace") as HTMLElement;
  const styleSelect = document.getElementById(
    "style-select",
  ) as HTMLSelectElement;
  const exportBtn = document.getElementById(
    "export-pdf",
  ) as HTMLButtonElement;
  const newPlayBtn = document.getElementById("new-play") as HTMLButtonElement;
  const openDraftsBtn = document.getElementById("open-drafts") as HTMLButtonElement;
  const loadExampleBtn = document.getElementById(
    "load-example",
  ) as HTMLButtonElement;
  const copyBtn = document.getElementById(
    "copy-source",
  ) as HTMLButtonElement;
  const saveBtn = document.getElementById(
    "save-source",
  ) as HTMLButtonElement;
  const toggleCheatSheetBtn = document.getElementById(
    "toggle-cheat-sheet",
  ) as HTMLButtonElement;
  const togglePreviewBtn = document.getElementById(
    "toggle-preview",
  ) as HTMLButtonElement;
  const dismissCheatSheetBtn = document.getElementById(
    "dismiss-cheat-sheet",
  ) as HTMLButtonElement;
  const editorVersion = document.getElementById("editor-version") as HTMLElement;
  const cheatSheetVersion = document.getElementById("cheat-sheet-version") as HTMLElement;
  const confirmDialog = document.getElementById(
    "confirm-dialog",
  ) as HTMLDialogElement;
  const confirmTitle = document.getElementById("confirm-title") as HTMLElement;
  const confirmMessage = document.getElementById("confirm-message") as HTMLElement;
  const confirmAccept = document.getElementById(
    "confirm-accept",
  ) as HTMLButtonElement;
  const draftsDialog = document.getElementById("drafts-dialog") as HTMLDialogElement;
  const importDsFileBtn = document.getElementById("import-ds-file") as HTMLButtonElement;
  const importDsInput = document.getElementById("import-ds-input") as HTMLInputElement;
  const draftsList = document.getElementById("drafts-list") as HTMLElement;
  const cheatSheet = document.getElementById("cheat-sheet") as HTMLElement;
  const formatToolbar = document.getElementById("format-toolbar") as HTMLElement;
  const formatButtons = Array.from(
    document.querySelectorAll<HTMLButtonElement>(".format-action"),
  );
  let { drafts, activeDraftId } = initializeDrafts();
  let persistTimer: number | null = null;

  const getActiveDraft = (): SavedDraft => {
    const draft = drafts.find((candidate) => candidate.id === activeDraftId);
    if (!draft) {
      const fallback = sortDrafts(drafts)[0] ?? buildSavedDraft(exampleContent);
      if (!drafts.some((candidate) => candidate.id === fallback.id)) {
        drafts = [fallback, ...drafts];
      }
      activeDraftId = fallback.id;
      writeSavedDrafts(drafts);
      writeActiveDraftId(activeDraftId);
      return fallback;
    }
    return draft;
  };

  const persistDraftState = () => {
    writeSavedDrafts(drafts);
    writeActiveDraftId(activeDraftId);
  };

  const flushDraftPersistence = () => {
    if (persistTimer !== null) {
      window.clearTimeout(persistTimer);
      persistTimer = null;
    }

    persistDraftState();
    if (draftsDialog.open) {
      renderDraftList();
    }
  };

  const scheduleDraftPersistence = () => {
    if (persistTimer !== null) {
      window.clearTimeout(persistTimer);
    }

    persistTimer = window.setTimeout(() => {
      persistTimer = null;
      persistDraftState();
      if (draftsDialog.open) {
        renderDraftList();
      }
    }, 250);
  };

  const renderDraftList = () => {
    draftsList.replaceChildren();
    const orderedDrafts = sortDrafts(drafts);

    if (orderedDrafts.length === 0) {
      const emptyState = document.createElement("p");
      emptyState.className = "drafts-empty";
      emptyState.textContent = "No saved drafts yet.";
      draftsList.append(emptyState);
      return;
    }

    for (const draft of orderedDrafts) {
      const row = document.createElement("article");
      row.className = "draft-row";

      const openButton = document.createElement("button");
      openButton.type = "button";
      openButton.className = "draft-open";
      openButton.dataset.draftId = draft.id;

      const title = document.createElement("span");
      title.className = "draft-open-title";
      title.textContent = draft.title;

      const meta = document.createElement("span");
      meta.className = "draft-open-meta";
      meta.textContent = new Date(draft.updatedAt).toLocaleString();

      const arrow = document.createElement("span");
      arrow.className = "draft-open-arrow";
      arrow.setAttribute("aria-hidden", "true");
      arrow.innerHTML = octiconChevronRight;

      if (draft.id === activeDraftId) {
        const badge = document.createElement("span");
        badge.className = "draft-open-badge";
        badge.textContent = "Open now";
        title.append(" ");
        title.append(badge);
      }

      const text = document.createElement("span");
      text.className = "draft-open-text";
      text.append(title, meta);

      openButton.append(text, arrow);

      const deleteButton = document.createElement("button");
      deleteButton.type = "button";
      deleteButton.className = "draft-delete";
      deleteButton.dataset.deleteDraftId = draft.id;
      deleteButton.setAttribute("aria-label", `Delete ${draft.title}`);
      deleteButton.title = `Delete ${draft.title}`;
      deleteButton.innerHTML = octiconTrash;

      row.append(openButton, deleteButton);
      draftsList.append(row);
    }
  };

  const switchToDraft = (draftId: string) => {
    flushDraftPersistence();

    const draft = drafts.find((candidate) => candidate.id === draftId);
    if (!draft) {
      return;
    }

    activeDraftId = draft.id;
    persistDraftState();
    renderDraftList();
    setEditorContent(draft.content);
  };

  const createDraft = (content: string) => {
    flushDraftPersistence();

    const draft = buildSavedDraft(content);
    drafts = [draft, ...drafts];
    activeDraftId = draft.id;
    persistDraftState();
    renderDraftList();
    setEditorContent(draft.content);
  };

  const setCheatSheetVisibility = (visible: boolean) => {
    cheatSheet.hidden = !visible;
    toggleCheatSheetBtn.setAttribute("aria-expanded", String(visible));

    if (visible) {
      localStorage.removeItem(cheatSheetStorageKey);
      return;
    }

    localStorage.setItem(cheatSheetStorageKey, "true");
  };

  const setPreviewVisibility = (visible: boolean) => {
    workspaceEl.classList.toggle("preview-hidden", !visible);
    togglePreviewBtn.setAttribute("aria-pressed", String(!visible));
    togglePreviewBtn.setAttribute("aria-label", visible ? "Hide preview" : "Show preview");
    togglePreviewBtn.title = visible ? "Hide preview" : "Show preview";
    togglePreviewBtn.innerHTML = visible ? octiconSidebarCollapse : octiconSidebarExpand;

    if (visible) {
      localStorage.removeItem(previewStorageKey);
      return;
    }

    localStorage.setItem(previewStorageKey, "true");
  };

  try {
    await initWasm();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    loading.textContent = `Failed to load Downstage editor: ${message}`;
    return;
  }

  loading.style.display = "none";
  workspace.classList.remove("hidden");
  editorVersion.textContent = __APP_VERSION__;
  cheatSheetVersion.textContent = `Editor ${__APP_VERSION__}`;
  exportBtn.disabled = false;
  newPlayBtn.disabled = false;
  openDraftsBtn.disabled = false;
  loadExampleBtn.disabled = false;
  copyBtn.disabled = false;
  saveBtn.disabled = false;
  togglePreviewBtn.disabled = false;
  toggleCheatSheetBtn.disabled = false;
  dismissCheatSheetBtn.disabled = false;
  importDsFileBtn.disabled = false;
  for (const button of formatButtons) {
    button.disabled = false;
  }

  const previewPlugin = createPreviewPlugin(
    iframe,
    styleSelect,
    () => editorView,
    () => !workspaceEl.classList.contains("preview-hidden"),
  );
  const scrollSyncPlugin = createScrollSyncPlugin(iframe);

  const state = EditorState.create({
    doc: getActiveDraft().content,
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
      EditorView.updateListener.of((update) => {
        if (!update.docChanged) {
          return;
        }

        const currentDraft = drafts.find((candidate) => candidate.id === activeDraftId);
        if (!currentDraft) {
          return;
        }

        currentDraft.content = update.state.doc.toString();
        currentDraft.title = extractDraftTitle(currentDraft.content);
        currentDraft.updatedAt = new Date().toISOString();
        scheduleDraftPersistence();
      }),
    ],
  });

  editorView = new EditorView({
    state,
    parent: editorPane,
  });

  setupPdfExport(exportBtn, styleSelect, () => editorView);
  setPreviewVisibility(localStorage.getItem(previewStorageKey) !== "true");
  setCheatSheetVisibility(localStorage.getItem(cheatSheetStorageKey) !== "true");
  renderDraftList();
  window.addEventListener("pagehide", flushDraftPersistence);

  newPlayBtn.addEventListener("click", () => {
    createDraft(buildNewPlayTemplate());
  });

  openDraftsBtn.addEventListener("click", () => {
    flushDraftPersistence();
    renderDraftList();
    draftsDialog.showModal();
  });

  importDsFileBtn.addEventListener("click", () => {
    importDsInput.click();
  });

  importDsInput.addEventListener("change", async () => {
    const [file] = Array.from(importDsInput.files ?? []);
    if (!file) {
      return;
    }

    const content = await file.text();
    createDraft(content);
    draftsDialog.close();
    importDsInput.value = "";
  });

  loadExampleBtn.addEventListener("click", () => {
    createDraft(exampleContent);
  });

  toggleCheatSheetBtn.addEventListener("click", () => {
    setCheatSheetVisibility(cheatSheet.hidden);
  });

  togglePreviewBtn.addEventListener("click", () => {
    const showPreview = workspaceEl.classList.contains("preview-hidden");
    setPreviewVisibility(showPreview);
    if (showPreview && editorView) {
      renderPreview(iframe, styleSelect, editorView.state.doc.toString());
    }
  });

  dismissCheatSheetBtn.addEventListener("click", () => {
    setCheatSheetVisibility(false);
  });

  formatToolbar.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) {
      return;
    }

    const button = target.closest<HTMLButtonElement>("[data-action]");
    if (!button) {
      return;
    }

    const action = button.dataset.action as ToolbarAction | undefined;
    if (!action) {
      return;
    }

    applyToolbarAction(action);
  });

  draftsList.addEventListener("click", async (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) {
      return;
    }

    const deleteButton = target.closest<HTMLButtonElement>("[data-delete-draft-id]");
    if (deleteButton) {
      const draftId = deleteButton.dataset.deleteDraftId;
      const draft = drafts.find((candidate) => candidate.id === draftId);
      if (!draft) {
        return;
      }

      draftsDialog.close();
      const confirmed = await confirmAction(
        confirmDialog,
        confirmTitle,
        confirmMessage,
        confirmAccept,
        `Delete "${draft.title}"?`,
        "This permanently deletes the draft from this browser. This cannot be undone.",
        "Delete Draft",
      );
      if (!confirmed) {
        renderDraftList();
        reopenDialog(draftsDialog);
        return;
      }

      drafts = drafts.filter((candidate) => candidate.id !== draft.id);
      if (draft.id === activeDraftId) {
        if (drafts.length === 0) {
          const replacement = buildSavedDraft(buildNewPlayTemplate());
          drafts = [replacement];
        }
        activeDraftId = sortDrafts(drafts)[0].id;
        persistDraftState();
        renderDraftList();
        setEditorContent(getActiveDraft().content);
      } else {
        persistDraftState();
        renderDraftList();
      }
      reopenDialog(draftsDialog);
      return;
    }

    const openButton = target.closest<HTMLButtonElement>("[data-draft-id]");
    if (!openButton) {
      return;
    }

    const draftId = openButton.dataset.draftId;
    if (!draftId) {
      return;
    }

    switchToDraft(draftId);
    draftsDialog.close();
  });

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
    const filename = `${extractDraftTitle(source).replace(/[^a-z0-9]+/gi, "-").replace(/^-+|-+$/g, "").toLowerCase() || "untitled"}.ds`;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  });
}

main();
