// Command handler map. The catalog (labels, IDs, accelerators, menu
// paths, categories) lives on the Go side in internal/desktop/commands.go
// so Wails can build the native menu. This file's only job is to map
// IDs to handler bodies (and optional enablement predicates).
//
// Adding a command: add it to the Go catalog, then add a handler entry
// here. Changing a label or accelerator: Go only. Keep this file
// metadata-free.

import type { Ref } from "vue";
import type { Store } from "../core/store";
import type { Workspace } from "./workspace";
import type { DesktopCapabilities } from "./types";
import type { HandlerEntry } from "./command-dispatcher";
import type { WorkbenchTab } from "../components/shared/workbench-tabs";
import type { SearchMode } from "../core/engine";

// Anything a command handler might need, injected by the host. Keeps the
// handlers decoupled from the Vue component they live next to.
export interface CommandContext {
  env: DesktopCapabilities;
  store: Store;
  workspace: Workspace;
  toast: {
    addToast: (message: string, kind: "success" | "error" | "info", durationMs?: number) => void;
  };
  // Reactive refs owned by AppDesktop.
  activeContent: Ref<string>;
  editorContent: Ref<string>;
  isV1Document: Ref<boolean>;
  isViewingRevision: Ref<boolean>;
  // Flushing the debounced file save. Save Version / Export / Open Folder
  // all need the live buffer durable on disk before they proceed.
  flushSave: () => Promise<void>;
  // Narrow imperative hooks on the shared editor. `applyFormat` stays
  // exposed; search/drawer/palette state are host-owned refs mutated
  // directly here (see `ui`).
  editor: {
    applyFormat: (action: string) => void;
  };
  // Host-owned UI state the host has lifted out of Editor.vue. Commands
  // mutate these refs directly; Editor reacts through v-model props.
  ui: {
    drawerOpen: Ref<boolean>;
    drawerTab: Ref<WorkbenchTab>;
    searchRequest: Ref<{ mode: SearchMode; nonce: number }>;
    openPalette: (mode?: "command" | "file") => void;
    openSettings: (tab?: "library" | "appearance" | "export" | "spellcheck") => void;
    openNewFolderPrompt: (parentPath?: string) => void;
    // Opens the PDF export dialog. The host owns the dialog state and
    // the render/save pipeline; this is just the trigger. Fire-and-
    // forget: the host loads persisted prefs asynchronously before the
    // dialog appears.
    openExportDialog: () => void;
  };
}

const nothingToSnapshotPrefix = "downstage: nothing-to-snapshot";

// Template seed for the New Play command.
const newPlayTemplate = () =>
  `# Untitled Play\nSubtitle: A Play in One Act\nAuthor: Your Name\nDate: ${new Date().getFullYear()}\nDraft: First\n\n## Dramatis Personae\n\nPROTAGONIST - Add your cast here\n\n## ACT I\n\n### SCENE 1\n\n> Describe the setting here.\n\nPROTAGONIST\nWrite your opening lines here.\n`;

export function createCommandHandlers(ctx: CommandContext): Array<[string, HandlerEntry]> {
  const {
    env, store, workspace, toast, activeContent, editorContent,
    isV1Document, isViewingRevision, flushSave, editor, ui,
  } = ctx;

  // Common enablement predicates — hoisted so the ID→predicate mapping
  // is a clean lookup rather than inline repetition.
  const hasActiveFile = () => !!workspace.state.activeFile;
  const hasActiveFileEditable = () => hasActiveFile() && !isViewingRevision.value;
  const canExport = () => hasActiveFile() && !isV1Document.value;

  // Opens a workbench drawer tab, toggling if the tab is already open.
  const toggleDrawerTab = (tab: WorkbenchTab) => {
    if (ui.drawerOpen.value && ui.drawerTab.value === tab) {
      ui.drawerOpen.value = false;
      return;
    }
    ui.drawerTab.value = tab;
    ui.drawerOpen.value = true;
  };

  // Bumps the search-request nonce so Editor watches the change and
  // opens its find-drawer in the requested mode.
  const openSearch = (mode: SearchMode) => {
    ui.searchRequest.value = { mode, nonce: ui.searchRequest.value.nonce + 1 };
  };

  async function handleNewPlay() {
    if (!workspace.state.libraryPath) {
      toast.addToast("No library open — set one in Settings > Library", "error");
      return;
    }
    await flushSave();
    try {
      const path = await workspace.createFile("Untitled Play", newPlayTemplate());
      activeContent.value = await workspace.selectFile(path);
      toast.addToast("Created new play", "success");
    } catch (e: any) {
      toast.addToast(`Failed to create file: ${e?.message ?? e}`, "error");
    }
  }

  async function handleOpen() {
    await flushSave();
    const absPath = await env.openExternalFileDialog();
    if (!absPath) return;
    try {
      const content = await workspace.openExternalFile(absPath);
      activeContent.value = content;
      if (workspace.state.externalFile) {
        const name = absPath.split(/[\\/]/).pop() ?? "file";
        toast.addToast(`Opened ${name} read-only — use Add to Library to keep editing`, "info", 6000);
      }
    } catch (e: any) {
      toast.addToast(`Failed to open file: ${e?.message ?? e}`, "error");
    }
  }

  async function handleSaveVersion() {
    if (!workspace.state.activeFile) return;
    await flushSave();
    const filename = workspace.state.activeFile.split(/[\\/]/).pop() || "file";
    try {
      await workspace.snapshotFile(`Snapshot ${filename}`);
      toast.addToast("Version saved", "success");
    } catch (e: any) {
      const message = String(e?.message ?? e);
      if (message.includes(nothingToSnapshotPrefix)) {
        toast.addToast("No changes to snapshot", "info");
      } else {
        toast.addToast(`Failed to save version: ${message}`, "error");
      }
    }
  }

  // Export opens the PDF dialog. The host (AppDesktop) renders the
  // dialog, loads persisted defaults from env.getExportPreferences, and
  // handles render + save on confirm. Keeping the handler tiny keeps
  // all of the export pipeline's state in one place.
  function handleExport() {
    ui.openExportDialog();
  }

  async function handleCopyAll() {
    await navigator.clipboard.writeText(editorContent.value);
    toast.addToast("Copied to clipboard", "success");
  }

  // File navigation. Wraps modulo list length so Next at the end cycles
  // to the start; Prev at the start cycles to the end. Small niceness
  // over staying put, and more Finder-like.
  function navigateFile(direction: 1 | -1) {
    const files = workspace.libraryFiles.value;
    if (files.length === 0) return;
    const current = workspace.state.activeFile;
    const currentIdx = files.findIndex((f) => f.path === current);
    const nextIdx = currentIdx < 0
      ? 0
      : (currentIdx + direction + files.length) % files.length;
    const nextPath = files[nextIdx].path;
    void (async () => {
      await flushSave();
      activeContent.value = await workspace.selectFile(nextPath);
    })();
  }

  function handleNewFolder() {
    ui.openNewFolderPrompt();
  }

  return [
    // File
    ["file.newPlay", { handler: handleNewPlay }],
    ["file.open", { handler: handleOpen }],
    ["library.newFolder", { handler: handleNewFolder }],
    ["file.saveVersion", { handler: handleSaveVersion, isEnabled: hasActiveFileEditable }],
    ["file.exportPdf", { handler: handleExport, isEnabled: canExport }],
    ["file.settings", { handler: () => ui.openSettings() }],
    ["file.settings.spellcheck", { handler: () => ui.openSettings("spellcheck") }],

    // Edit
    ["edit.find", { handler: () => openSearch("find"), isEnabled: hasActiveFile }],
    ["edit.findReplace", { handler: () => openSearch("replace"), isEnabled: hasActiveFile }],
    ["edit.copyAll", { handler: handleCopyAll, isEnabled: hasActiveFile }],

    // View
    ["view.commandPalette", { handler: () => ui.openPalette("command") }],
    ["view.togglePreview", {
      handler: () => { store.state.previewHidden = !store.state.previewHidden; },
    }],
    ["view.toggleSidebar", { handler: () => workspace.toggleSidebar() }],
    ["view.toggleIssues", { handler: () => toggleDrawerTab("issues"), isEnabled: hasActiveFile }],
    ["view.toggleOutline", { handler: () => toggleDrawerTab("outline"), isEnabled: hasActiveFile }],
    ["view.toggleStats", { handler: () => toggleDrawerTab("stats"), isEnabled: hasActiveFile }],
    ["view.toggleDrawerDock", {
      handler: () => {
        const next = workspace.state.drawerDock === "right" ? "bottom" : "right";
        workspace.setDrawerDock(next);
      },
    }],

    // Navigate
    ["navigate.nextFile", {
      handler: () => navigateFile(1),
      isEnabled: () => workspace.libraryFiles.value.length > 1,
    }],
    ["navigate.prevFile", {
      handler: () => navigateFile(-1),
      isEnabled: () => workspace.libraryFiles.value.length > 1,
    }],
    ["navigate.goToFile", {
      handler: () => ui.openPalette("file"),
      isEnabled: () => workspace.libraryFiles.value.length > 0,
    }],

    // Format — all go through the editor's imperative applyFormat hook.
    // Menu accelerators own Cmd+B/I/U; the engine's custom keymap that
    // previously bound them has been removed.
    ["format.bold", { handler: () => editor.applyFormat("bold"), isEnabled: hasActiveFileEditable }],
    ["format.italic", { handler: () => editor.applyFormat("italic"), isEnabled: hasActiveFileEditable }],
    ["format.underline", { handler: () => editor.applyFormat("underline"), isEnabled: hasActiveFileEditable }],
    ["format.cue", { handler: () => editor.applyFormat("cue"), isEnabled: hasActiveFileEditable }],
    ["format.direction", { handler: () => editor.applyFormat("direction"), isEnabled: hasActiveFileEditable }],
    ["format.act", { handler: () => editor.applyFormat("act"), isEnabled: hasActiveFileEditable }],
    ["format.scene", { handler: () => editor.applyFormat("scene"), isEnabled: hasActiveFileEditable }],
    ["format.song", { handler: () => editor.applyFormat("song"), isEnabled: hasActiveFileEditable }],
    ["format.pageBreak", { handler: () => editor.applyFormat("page-break"), isEnabled: hasActiveFileEditable }],

    // Help
    ["help.toggle", { handler: () => toggleDrawerTab("help"), isEnabled: hasActiveFile }],
    ["help.github", { handler: () => env.openURL("https://github.com/jscaltreto/downstage") }],
    ["help.docs", { handler: () => env.openURL("https://getdownstage.com/docs") }],
    ["help.about", { handler: () => env.showAboutDialog() }],
  ];
}
