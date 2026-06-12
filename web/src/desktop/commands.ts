import type { Ref } from "vue";
import type { Store } from "../core/store";
import type { Workspace } from "./workspace";
import type { DesktopCapabilities } from "./types";
import type { HandlerEntry } from "./command-dispatcher";
import type { WorkbenchTab } from "../components/shared/workbench-tabs";
import { helpLinks } from "../core/help-links";
import type { SearchMode } from "../core/engine";

export interface CommandContext {
  env: DesktopCapabilities;
  store: Store;
  workspace: Workspace;
  toast: {
    addToast: (message: string, kind: "success" | "error" | "info", durationMs?: number) => void;
  };
  activeContent: Ref<string>;
  editorContent: Ref<string>;
  isV1Document: Ref<boolean>;
  isViewingRevision: Ref<boolean>;
  isInCompareTwo: Ref<boolean>;
  isViewingExternal: Ref<boolean>;
  flushSave: () => Promise<void>;
  // Host-owned callback to suppress the autosave watcher's next tick
  // when commands.ts assigns activeContent.value after a programmatic
  // load. The host gates its autosave-watcher on a (file, content)
  // sentinel; this is the wire to set it.
  markProgrammaticLoad: (file: string, content: string) => void;
  editor: {
    applyFormat: (action: string) => void;
    undo: () => void;
    redo: () => void;
    cut: () => void;
    copy: () => void;
    paste: () => void;
    selectAll: () => void;
  };
  ui: {
    drawerOpen: Ref<boolean>;
    drawerTab: Ref<WorkbenchTab>;
    searchRequest: Ref<{ mode: SearchMode; nonce: number }>;
    openPalette: (mode?: "command" | "file") => void;
    openSettings: (tab?: "library" | "appearance" | "export" | "spellcheck") => void;
    openNewFolderPrompt: (parentPath?: string) => void;
    openExportDialog: () => void;
    openSaveVersionPrompt: () => void;
    openReviewChanges: () => void;
    // Confirm-and-delete the active library file. Implemented in the host
    // so the confirm-prompt UX (BaseModal vs native confirm) stays in one
    // place; commands.ts just opens it.
    requestDeleteActiveFile: () => void;
  };
}

const newPlayTemplate = () =>
  `# Untitled Play\nSubtitle: A Play in One Act\nAuthor: Your Name\nDate: ${new Date().getFullYear()}\nDraft: First\n\n## Dramatis Personae\n\nPROTAGONIST - Add your cast here\n\n## ACT I\n\n### SCENE 1\n\n> Describe the setting here.\n\nPROTAGONIST\nWrite your opening lines here.\n`;

function errorMessage(error: unknown): string {
  return String((error as { message?: unknown } | null)?.message ?? error);
}

export function createCommandHandlers(ctx: CommandContext): Array<[string, HandlerEntry]> {
  const {
    env, store, workspace, toast, activeContent, editorContent,
    isV1Document, isViewingRevision, isInCompareTwo, isViewingExternal,
    flushSave, markProgrammaticLoad, editor, ui,
  } = ctx;

  const hasActiveFile = () => !!workspace.state.activeFile;
  const hasActiveFileEditable = () => hasActiveFile() && !isViewingRevision.value;
  const canExport = () => hasActiveFile() && !isV1Document.value && !isInCompareTwo.value;
  const canCopyAll = () => hasActiveFile() && !isInCompareTwo.value;
  const canExportDs = () =>
    (hasActiveFile() || isViewingExternal.value) && !isInCompareTwo.value;

  const toggleDrawerTab = (tab: WorkbenchTab) => {
    if (ui.drawerOpen.value && ui.drawerTab.value === tab) {
      ui.drawerOpen.value = false;
      return;
    }
    ui.drawerTab.value = tab;
    ui.drawerOpen.value = true;
  };

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
      const content = await workspace.selectFile(path);
      markProgrammaticLoad(path, content);
      activeContent.value = content;
      toast.addToast("Created new play", "success");
    } catch (error: unknown) {
      toast.addToast(`Failed to create file: ${errorMessage(error)}`, "error");
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
    } catch (error: unknown) {
      toast.addToast(`Failed to open file: ${errorMessage(error)}`, "error");
    }
  }

  async function handleSave() {
    await flushSave();
  }

  function handleSaveVersion() {
    if (!workspace.state.activeFile) return;
    ui.openSaveVersionPrompt();
  }

  function handleExport() {
    ui.openExportDialog();
  }

  async function handleExportDs() {
    if (!workspace.state.activeFile && !workspace.state.externalFile) return;
    await flushSave();
    const source = editorContent.value;
    const baseName =
      workspace.state.activeFile?.split(/[\\/]/).pop()
      ?? workspace.state.externalFile?.absPath.split(/[\\/]/).pop()
      ?? "untitled.ds";
    try {
      await env.saveFile(baseName, source, [
        { displayName: "Downstage Files (*.ds)", pattern: "*.ds" },
      ]);
    } catch (error: unknown) {
      toast.addToast(`Failed to export file: ${errorMessage(error)}`, "error");
    }
  }

  async function handleQuit() {
    await flushSave();
    try {
      await env.quit();
    } catch (error: unknown) {
      toast.addToast(`Failed to quit: ${errorMessage(error)}`, "error");
    }
  }

  async function handleCopyAll() {
    await navigator.clipboard.writeText(editorContent.value);
    toast.addToast("Copied to clipboard", "success");
  }

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
      const content = await workspace.selectFile(nextPath);
      markProgrammaticLoad(nextPath, content);
      activeContent.value = content;
    })();
  }

  function handleNewFolder() {
    ui.openNewFolderPrompt();
  }

  return [
    ["file.newPlay", { handler: handleNewPlay }],
    ["file.open", { handler: handleOpen }],
    ["library.newFolder", { handler: handleNewFolder }],
    ["file.save", { handler: handleSave, isEnabled: hasActiveFileEditable }],
    ["file.saveVersion", { handler: handleSaveVersion, isEnabled: hasActiveFileEditable }],
    ["file.exportPdf", { handler: handleExport, isEnabled: canExport }],
    ["file.exportDs", { handler: handleExportDs, isEnabled: canExportDs }],
    ["file.settings", { handler: () => ui.openSettings() }],
    ["file.settings.spellcheck", { handler: () => ui.openSettings("spellcheck") }],
    ["file.quit", { handler: handleQuit }],

    ["edit.undo", { handler: () => editor.undo(), isEnabled: hasActiveFileEditable }],
    ["edit.redo", { handler: () => editor.redo(), isEnabled: hasActiveFileEditable }],
    ["edit.cut", { handler: () => editor.cut(), isEnabled: hasActiveFileEditable }],
    ["edit.copy", { handler: () => editor.copy(), isEnabled: hasActiveFile }],
    ["edit.paste", { handler: () => editor.paste(), isEnabled: hasActiveFileEditable }],
    ["edit.selectAll", { handler: () => editor.selectAll(), isEnabled: hasActiveFile }],
    ["edit.find", { handler: () => openSearch("find"), isEnabled: hasActiveFile }],
    ["edit.findReplace", { handler: () => openSearch("replace"), isEnabled: hasActiveFile }],
    ["edit.copyAll", { handler: handleCopyAll, isEnabled: canCopyAll }],

    ["view.commandPalette", { handler: () => ui.openPalette("command") }],
    ["view.togglePreview", {
      handler: () => { store.state.previewHidden = !store.state.previewHidden; },
    }],
    ["view.toggleSidebar", { handler: () => workspace.toggleSidebar() }],
    ["view.toggleIssues", { handler: () => toggleDrawerTab("issues"), isEnabled: hasActiveFile }],
    ["view.toggleOutline", { handler: () => toggleDrawerTab("outline"), isEnabled: hasActiveFile }],
    ["view.toggleStats", { handler: () => toggleDrawerTab("stats"), isEnabled: hasActiveFile }],

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

    ["format.bold", { handler: () => editor.applyFormat("bold"), isEnabled: hasActiveFileEditable }],
    ["format.italic", { handler: () => editor.applyFormat("italic"), isEnabled: hasActiveFileEditable }],
    ["format.underline", { handler: () => editor.applyFormat("underline"), isEnabled: hasActiveFileEditable }],
    ["format.strikethrough", { handler: () => editor.applyFormat("strikethrough"), isEnabled: hasActiveFileEditable }],

    ["insert.cue", { handler: () => editor.applyFormat("cue"), isEnabled: hasActiveFileEditable }],
    ["insert.direction", { handler: () => editor.applyFormat("direction"), isEnabled: hasActiveFileEditable }],
    ["insert.act", { handler: () => editor.applyFormat("act"), isEnabled: hasActiveFileEditable }],
    ["insert.scene", { handler: () => editor.applyFormat("scene"), isEnabled: hasActiveFileEditable }],
    ["insert.song", { handler: () => editor.applyFormat("song"), isEnabled: hasActiveFileEditable }],
    ["insert.pageBreak", { handler: () => editor.applyFormat("page-break"), isEnabled: hasActiveFileEditable }],

    ["library.delete", {
      handler: () => ui.requestDeleteActiveFile(),
      isEnabled: hasActiveFileEditable,
    }],
    ["library.reviewChanges", {
      handler: () => ui.openReviewChanges(),
      isEnabled: () => (workspace.state.libraryDirty?.count ?? 0) > 0,
    }],

    ["help.toggle", { handler: () => toggleDrawerTab("help") }],
    ["help.github", { handler: () => env.openURL(helpLinks.github) }],
    ["help.docs", { handler: () => env.openURL(helpLinks.docs) }],
    ["help.about", { handler: () => env.showAboutDialog() }],
  ];
}
