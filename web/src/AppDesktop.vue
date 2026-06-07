<script setup lang="ts">
import { computed, provide, onMounted, onUnmounted, ref, watch, watchEffect } from 'vue';
import {
    FolderOpen, FolderSync, FileText, FolderPlus,
    BookOpen, Terminal, Sparkles, History, ExternalLink,
    RotateCcw, X, PanelLeftClose, Plus, GitCompare
} from 'lucide-vue-next';
import { Store } from './core/store';
import type { EditorEnv, ExportPdfOptions } from './core/types';
import type { DesktopCapabilities } from './desktop/types';
import { Workspace } from './desktop/workspace';
import { registerFlushSave } from './desktop/flush-save';
import { CommandDispatcher } from './desktop/command-dispatcher';
import { createCommandHandlers, type CommandContext } from './desktop/commands';
import { registerDispatcher } from './desktop/dispatcher-registry';
import type { WorkbenchTab } from './components/shared/workbench-tabs';
import type { SearchMode } from './core/engine';
import ToastManager from './components/shared/ToastManager.vue';
import Editor from './components/shared/Editor.vue';
import CommandPalette from './desktop/CommandPalette.vue';
import Settings from './desktop/Settings.vue';
import LibraryTree from './desktop/LibraryTree.vue';
import PromptModal from './desktop/PromptModal.vue';
import ConfirmModal from './desktop/ConfirmModal.vue';
import StatusBar from './desktop/StatusBar.vue';
import ReviewChangesModal from './desktop/ReviewChangesModal.vue';
import { displayFileName } from './desktop/naming';
import ExportPdfModal from './components/shared/ExportPdfModal.vue';
import RevisionDiffView from './desktop/RevisionDiffView.vue';
import VersionsPanel from './desktop/VersionsPanel.vue';
import { formatRevisionTimestamp } from './desktop/revision-format';

const props = defineProps<{
  env: DesktopCapabilities;
}>();

const store = new Store(props.env as EditorEnv);
provide('store', store);

const workspace = new Workspace(props.env);

const isLoaded = ref(false);
const activeContent = ref("");
const pageStyle = ref("standard");
const isV1Document = ref(false);

const cursor = ref<{ line: number; col: number }>({ line: 1, col: 1 });
const wordCount = ref(0);

function errorMessage(error: unknown): string {
  return String((error as { message?: unknown } | null)?.message ?? error);
}

let sidebarDragStartX = 0;
let sidebarDragStartWidth = 0;
function beginSidebarDrag(e: MouseEvent) {
  sidebarDragStartX = e.clientX;
  sidebarDragStartWidth = workspace.state.sidebarWidth;
  document.body.style.cursor = 'col-resize';
  window.addEventListener('mousemove', onSidebarDragMove);
  window.addEventListener('mouseup', onSidebarDragEnd, { once: true });
}
function onSidebarDragMove(e: MouseEvent) {
  const delta = e.clientX - sidebarDragStartX;
  workspace.setSidebarWidth(sidebarDragStartWidth + delta);
}
function onSidebarDragEnd() {
  window.removeEventListener('mousemove', onSidebarDragMove);
  document.body.style.cursor = '';
}
function resetSidebarWidth() {
  workspace.setSidebarWidth(256);
}

const toastManager = ref<InstanceType<typeof ToastManager> | null>(null);
const editorRef = ref<InstanceType<typeof Editor> | null>(null);

const drawerOpen = ref(false);
const drawerTab = ref<WorkbenchTab>('issues');
const searchRequest = ref<{ mode: SearchMode; nonce: number }>({ mode: 'find', nonce: 0 });

const paletteOpen = ref(false);
const paletteMode = ref<'command' | 'file'>('command');
const settingsOpen = ref(false);
const settingsTab = ref<'library' | 'appearance' | 'export' | 'spellcheck'>('library');

function openPalette(mode: 'command' | 'file' = 'command') {
  paletteMode.value = mode;
  paletteOpen.value = true;
}
function openSettings(tab: 'library' | 'appearance' | 'export' | 'spellcheck' = 'library') {
  settingsTab.value = tab;
  settingsOpen.value = true;
}

const exportDialogOpen = ref(false);
const exportInitialOptions = ref<ExportPdfOptions>({
  pageSize: 'letter',
  style: 'standard',
  layout: 'single',
  bookletGutter: '0.125in',
});

async function openExportDialog() {
  try {
    const prefs = await props.env.getExportPreferences();
    const style = prefs.style === 'condensed' ? 'condensed' : 'standard';
    const layout = style === 'standard' ? 'single' : prefs.layout;
    exportInitialOptions.value = {
      pageSize: prefs.pageSize,
      style,
      layout,
      bookletGutter: prefs.bookletGutter,
    };
  } catch {}
  exportDialogOpen.value = true;
}

async function handleExportConfirmed(opts: ExportPdfOptions) {
  exportDialogOpen.value = false;
  try {
    await props.env.setExportPreferences(opts);
  } catch {}

  await flushSave();
  const source = editorContent.value;
  const title = source.match(/^#\s+(.+)$/m)?.[1]?.trim() || 'untitled';
  const styleSlug = opts.style === 'condensed' ? 'acting-edition' : 'manuscript';
  const layoutSuffix = opts.layout === 'single' ? '' : `-${opts.layout}`;
  const filename = `${title.replace(/[^a-z0-9]+/gi, '-').toLowerCase()}-${styleSlug}${layoutSuffix}.pdf`;
  try {
    const pdfBytes = await props.env.renderPDF(source, opts);
    if (!pdfBytes || pdfBytes.byteLength === 0) {
      toastManager.value?.addToast(
        'PDF export failed. Check the export settings and try again.',
        'error',
        5000,
      );
      return;
    }
    await props.env.saveFile(filename, pdfBytes, [
      { displayName: 'PDF Files (*.pdf)', pattern: '*.pdf' },
    ]);
  } catch (error: unknown) {
    toastManager.value?.addToast(`Failed to export PDF: ${errorMessage(error)}`, 'error');
  }
}

const newFolderOpen = ref(false);
const newFolderParent = ref('');
const newFolderError = ref<string | null>(null);

const reviewChangesOpen = ref(false);
const reviewBusy = ref(false);

interface ConfirmConfig {
  title: string;
  message: string;
  confirmLabel: string;
  destructive: boolean;
  onConfirm: () => void | Promise<void>;
}
const confirmOpen = ref(false);
const confirmConfig = ref<ConfirmConfig | null>(null);

function askConfirm(config: ConfirmConfig) {
  confirmConfig.value = config;
  confirmOpen.value = true;
}

async function onConfirmAccepted() {
  const cfg = confirmConfig.value;
  confirmOpen.value = false;
  confirmConfig.value = null;
  if (cfg) await cfg.onConfirm();
}

function onConfirmClosed() {
  confirmOpen.value = false;
  confirmConfig.value = null;
}

function openReviewChanges() {
  if (!workspace.state.libraryPath) return;
  void workspace.refreshLibraryDirty();
  reviewChangesOpen.value = true;
}

async function onReviewCommit(paths: string[], message: string) {
  reviewBusy.value = true;
  try {
    await workspace.commitDirtyPaths(paths, message);
    toastManager.value?.addToast(`Committed ${paths.length} change${paths.length === 1 ? '' : 's'}`, 'success');
  } catch (error: unknown) {
    toastManager.value?.addToast(`Commit failed: ${errorMessage(error)}`, 'error');
  } finally {
    reviewBusy.value = false;
  }
}

async function onReviewDiscard(paths: string[]) {
  reviewBusy.value = true;
  try {
    await workspace.discardDirtyPaths(paths);
    toastManager.value?.addToast(`Discarded ${paths.length} change${paths.length === 1 ? '' : 's'}`, 'success');
  } catch (error: unknown) {
    toastManager.value?.addToast(`Discard failed: ${errorMessage(error)}`, 'error');
  } finally {
    reviewBusy.value = false;
  }
}

function requestDeleteFromTree(path: string) {
  const name = path.includes('/') ? path.slice(path.lastIndexOf('/') + 1) : path;
  askConfirm({
    title: `Delete ${displayFileName(name)}?`,
    message: `The file will move to the Deleted section. You can restore it from there, or permanently delete it later.`,
    confirmLabel: 'Delete',
    destructive: true,
    onConfirm: () => performDelete(path),
  });
}

function requestDeleteActiveFile() {
  const path = workspace.state.activeFile;
  if (!path) return;
  requestDeleteFromTree(path);
}

async function performDelete(path: string) {
  try {
    await flushSave();
    await workspace.deleteFile(path);
    if (workspace.state.activeFile === null) {
      // Was the active file. Open the next live one (or leave editor empty).
      const remaining = workspace.libraryFiles.value;
      if (remaining.length > 0) {
        await selectLibraryFile(remaining[0].path);
      } else {
        activeContent.value = '';
      }
    }
    // A tracked file lands in the Deleted section (worktree-status=Deleted).
    // An untracked file is gone for good — no HEAD blob to restore. Tailor
    // the toast so the restore promise only appears when restore is real.
    const wasTracked = (workspace.state.libraryDirty?.plays ?? [])
      .some((p) => p.path === path && p.kind === 'deleted');
    const display = displayFileName(path.includes('/') ? path.slice(path.lastIndexOf('/') + 1) : path);
    toastManager.value?.addToast(
      wasTracked
        ? `Deleted ${display} — restore from the Deleted section`
        : `Deleted ${display}`,
      'success',
    );
  } catch (error: unknown) {
    toastManager.value?.addToast(`Delete failed: ${errorMessage(error)}`, 'error');
  }
}

async function requestRestoreFromTree(path: string) {
  try {
    await workspace.restoreFile(path);
    const content = await workspace.selectFile(path);
    markProgrammaticLoad(path, content);
    activeContent.value = content;
    toastManager.value?.addToast(`Restored ${path}`, 'success');
  } catch (error: unknown) {
    toastManager.value?.addToast(`Restore failed: ${errorMessage(error)}`, 'error');
  }
}

function requestPermanentDeleteFromTree(path: string) {
  const name = path.includes('/') ? path.slice(path.lastIndexOf('/') + 1) : path;
  askConfirm({
    title: 'Permanently delete?',
    message: `Permanently remove ${path}.\n\nGit history is preserved but the file will no longer appear in your library.`,
    confirmLabel: 'Permanently delete',
    destructive: true,
    onConfirm: async () => {
      reviewBusy.value = true;
      try {
        await workspace.commitDirtyPaths([path], `Delete ${name}`);
        toastManager.value?.addToast(`Permanently deleted ${path}`, 'success');
      } catch (error: unknown) {
        toastManager.value?.addToast(`Permanent delete failed: ${errorMessage(error)}`, 'error');
      } finally {
        reviewBusy.value = false;
      }
    },
  });
}

function openNewFolderPrompt(parentPath = '') {
  if (!workspace.state.libraryPath) {
    toastManager.value?.addToast(
      'No library open — set one in Settings > Library',
      'error',
    );
    return;
  }
  newFolderParent.value = parentPath;
  newFolderError.value = null;
  newFolderOpen.value = true;
}

async function submitNewFolder(name: string) {
  if (name.includes('/') || name.includes('\\')) {
    newFolderError.value = 'Folder names cannot contain slashes';
    return;
  }
  const parent = newFolderParent.value;
  const relPath = parent ? `${parent}/${name}` : name;
  try {
    await workspace.createFolder(relPath);
    newFolderOpen.value = false;
    newFolderError.value = null;
    toastManager.value?.addToast(`Created folder "${name}"`, 'success');
  } catch (error: unknown) {
    newFolderError.value = `Failed to create folder: ${errorMessage(error)}`;
  }
}

const saveVersionOpen = ref(false);
const saveVersionInitial = ref('');
const saveVersionError = ref<string | null>(null);

function openSaveVersionPrompt() {
  if (!workspace.state.activeFile) return;
  const filename = workspace.state.activeFile.split(/[\\/]/).pop() || 'file';
  saveVersionInitial.value = `Snapshot ${displayFileName(filename)}`;
  saveVersionError.value = null;
  saveVersionOpen.value = true;
}

async function submitSaveVersion(name: string) {
  await flushSave();
  try {
    await workspace.snapshotFile(name);
    saveVersionOpen.value = false;
    saveVersionError.value = null;
    toastManager.value?.addToast('Version saved', 'success');
  } catch (error: unknown) {
    const message = errorMessage(error);
    if (message.includes('downstage: nothing-to-snapshot')) {
      saveVersionOpen.value = false;
      saveVersionError.value = null;
      toastManager.value?.addToast('No changes to snapshot', 'info');
    } else {
      saveVersionError.value = `Failed to save version: ${message}`;
    }
  }
}

let saveTimer: number | null = null;

let dispatcher: CommandDispatcher | null = null;

const libraryNameBase = computed(
  () => workspace.state.libraryPath?.split(/[\\/]/).pop() ?? '',
);
const activeFileBase = computed(() => {
  const path = workspace.state.activeFile;
  if (!path) return '';
  const base = path.split(/[\\/]/).pop() ?? '';
  return displayFileName(base);
});

const isViewingRevision = computed(
  () => workspace.state.viewingRevisionHash !== null,
);
const isViewingExternal = computed(
  () => workspace.state.externalFile !== null,
);
const isEditorReadOnly = computed(
  () => isViewingRevision.value || isViewingExternal.value,
);
const inCompareDiff = computed(
  () =>
    isViewingRevision.value &&
    workspace.state.revisionViewMode === 'compare',
);
const inCompareTwo = computed(
  () =>
    inCompareDiff.value &&
    workspace.state.compareSecondHash !== null,
);
const compareOriginalLabel = computed(() => {
  const meta = workspace.state.viewingRevisionMeta;
  if (!meta) {
    const hash = workspace.state.viewingRevisionHash ?? '';
    return hash ? `Saved ${hash.slice(0, 7)}` : 'Saved version';
  }
  return `Saved ${formatRevisionTimestamp(meta.timestamp)}`;
});
const compareSecondLabel = computed(() => {
  const meta = workspace.state.compareSecondMeta;
  if (!meta) {
    const hash = workspace.state.compareSecondHash ?? '';
    return hash ? `Saved ${hash.slice(0, 7)}` : 'Saved version';
  }
  return `Saved ${formatRevisionTimestamp(meta.timestamp)}`;
});

const editorContent = computed<string>({
  get: () => {
    if (isViewingExternal.value) {
      return workspace.state.externalFile?.content ?? "";
    }
    if (isViewingRevision.value) {
      return workspace.state.viewingRevisionContent ?? "";
    }
    return activeContent.value;
  },
  set: (value: string) => {
    if (isEditorReadOnly.value) return;
    activeContent.value = value;
  },
});

const editorDocumentKey = computed(() => {
  if (isViewingExternal.value) {
    return `external:${workspace.state.externalFile?.absPath ?? ""}`;
  }
  if (!workspace.state.activeFile) return null;
  return workspace.state.viewingRevisionHash
    ? `${workspace.state.activeFile}@${workspace.state.viewingRevisionHash}`
    : workspace.state.activeFile;
});


onMounted(async () => {
  await store.init();
  await workspace.init();

  if (workspace.state.libraryPath && workspace.libraryFiles.value.length > 0) {
    const lastFile = await props.env.getLastActiveFile();
    const exists = workspace.libraryFiles.value.some(f => f.path === lastFile);
    const path = lastFile && exists ? lastFile : workspace.libraryFiles.value[0].path;
    const content = await workspace.selectFile(path);
    markProgrammaticLoad(path, content);
    activeContent.value = content;
  }

  if (workspace.state.lastDrawerTab) {
    drawerTab.value = workspace.state.lastDrawerTab as WorkbenchTab;
  }
  watch(drawerTab, (next) => {
    workspace.setLastDrawerTab(next);
  });

  window.addEventListener('resize', scheduleBoundsSave);

  isLoaded.value = true;
  registerFlushSave(() => flushSave());

  dispatcher = new CommandDispatcher({
    setDisabledCommands: (ids) => props.env.setDisabledCommands(ids),
  });

  const ctx: CommandContext = {
    env: props.env,
    store,
    workspace,
    toast: {
      addToast: (message, kind, durationMs) =>
        toastManager.value?.addToast(message, kind, durationMs),
    },
    activeContent,
    editorContent,
    isV1Document,
    isViewingRevision,
    isInCompareTwo: inCompareTwo,
    isViewingExternal,
    flushSave,
    markProgrammaticLoad,
    editor: {
      applyFormat: (action: string) => editorRef.value?.applyFormat(action),
      undo: () => editorRef.value?.undo(),
      redo: () => editorRef.value?.redo(),
      cut: () => editorRef.value?.cut(),
      copy: () => editorRef.value?.copy(),
      paste: () => editorRef.value?.paste(),
      selectAll: () => editorRef.value?.selectAll(),
    },
    ui: {
      drawerOpen,
      drawerTab,
      searchRequest,
      openPalette,
      openSettings,
      openNewFolderPrompt,
      openExportDialog,
      openSaveVersionPrompt,
      openReviewChanges,
      requestDeleteActiveFile,
    },
  };
  for (const [id, entry] of createCommandHandlers(ctx)) {
    dispatcher.register(id, entry);
  }
  registerDispatcher(dispatcher);

  watchEffect(() => {
    void workspace.state.activeFile;
    void workspace.libraryFiles.value.length;
    void workspace.state.viewingRevisionHash;
    void workspace.state.revisionViewMode;
    void workspace.state.compareSecondHash;
    void workspace.state.externalFile;
    void workspace.state.libraryDirty?.count;
    void isV1Document.value;
    dispatcher?.scheduleRefresh();
  });

  // Library-wide dirty surface: poll on a long interval while focused,
  // and refresh immediately on every focus event so users tabbing back
  // from a terminal see the latest state without waiting for the tick.
  if (typeof window !== 'undefined') {
    void workspace.refreshLibraryDirty();
    workspace.startLibraryDirtyPolling();
    window.addEventListener('focus', onWindowFocus);
    window.addEventListener('blur', onWindowBlur);
  }
});

function onWindowFocus() {
  void workspace.refreshLibraryDirty();
  workspace.startLibraryDirtyPolling();
}

function onWindowBlur() {
  workspace.stopLibraryDirtyPolling();
}

onUnmounted(() => {
  registerFlushSave(null);
  registerDispatcher(null);
  window.removeEventListener('resize', scheduleBoundsSave);
  if (typeof window !== 'undefined') {
    window.removeEventListener('focus', onWindowFocus);
    window.removeEventListener('blur', onWindowBlur);
  }
  workspace.stopLibraryDirtyPolling();
  if (boundsSaveTimer !== null) {
    clearTimeout(boundsSaveTimer);
    boundsSaveTimer = null;
  }
  void flushSave();
  void props.env.flushPreferences();
});

watch(
  () => workspace.state.sidebarCollapsed,
  (collapsed) => {
    if (collapsed && workspace.state.pickingSecondForCompare) {
      workspace.cancelPickSecond();
    }
  },
);

let boundsSaveTimer: ReturnType<typeof setTimeout> | null = null;
function scheduleBoundsSave() {
  if (boundsSaveTimer !== null) clearTimeout(boundsSaveTimer);
  boundsSaveTimer = setTimeout(() => {
    boundsSaveTimer = null;
    void props.env.saveWindowBoundsIfNormal();
  }, 500);
}

async function flushSave(): Promise<void> {
  if (!saveTimer) return;
  clearTimeout(saveTimer);
  saveTimer = null;
  if (workspace.state.activeFile) {
    await workspace.saveFile(activeContent.value);
  }
}

function handleNewPlay() {
  void dispatcher?.dispatch('file.newPlay');
}

async function handleRevealLibrary() {
  try {
    await props.env.revealLibraryInExplorer();
  } catch (error: unknown) {
    toastManager.value?.addToast(`Reveal failed: ${errorMessage(error)}`, 'error');
  }
}

const externalFileBasename = computed(() => {
  const abs = workspace.state.externalFile?.absPath ?? '';
  return abs.split(/[\\/]/).pop() || abs;
});

async function handleAddExternalFileToLibrary() {
  try {
    const rel = await workspace.addExternalFileToLibrary("");
    toastManager.value?.addToast(`Added ${rel} to your library`, "success");
  } catch (error: unknown) {
    toastManager.value?.addToast(`Failed to add file to library: ${errorMessage(error)}`, "error");
  }
}

function handleCloseExternalFile() {
  workspace.closeExternalFile();
  activeContent.value = "";
}

async function handleChangeLibraryLocation() {
  await flushSave();
  const path = await workspace.changeLibraryLocation();
  if (!path) return;
  activeContent.value = "";
  if (workspace.libraryFiles.value.length > 0) {
    const firstPath = workspace.libraryFiles.value[0].path;
    const content = await workspace.selectFile(firstPath);
    markProgrammaticLoad(firstPath, content);
    activeContent.value = content;
  }
  toastManager.value?.addToast(`Opened library: ${path.split(/[\\/]/).pop()}`, "success");
}

async function selectLibraryFile(path: string) {
  await flushSave();
  try {
    const content = await workspace.selectFile(path);
    markProgrammaticLoad(path, content);
    activeContent.value = content;
  } catch (error: unknown) {
    const name = path.split(/[\\/]/).pop() ?? path;
    toastManager.value?.addToast(`Failed to open ${name}: ${errorMessage(error)}`, "error");
  }
}

async function handleViewRevision(hash: string) {
  await flushSave();
  try {
    await workspace.viewRevision(hash);
  } catch (error: unknown) {
    toastManager.value?.addToast(`Failed to load version: ${errorMessage(error)}`, "error");
  }
}

async function handleCompareToCurrent(hash: string) {
  if (workspace.state.viewingRevisionHash !== hash) {
    await handleViewRevision(hash);
  }
  workspace.stopCompareTwo();
  if (workspace.state.revisionViewMode !== 'compare') {
    workspace.toggleRevisionCompare();
  }
}

function handleExitRevisionView() {
  workspace.clearRevisionView();
}

async function handleRestoreRevision() {
  const hash = workspace.state.viewingRevisionHash;
  if (!hash) return;
  try {
    const restored = await workspace.restoreRevision(hash, activeContent.value);
    // restoreRevision wrote `restored` to disk; the autosave watcher
    // would re-write the same bytes a second later if we didn't mark
    // this as a programmatic load.
    if (workspace.state.activeFile) {
      markProgrammaticLoad(workspace.state.activeFile, restored);
    }
    activeContent.value = restored;
    toastManager.value?.addToast("Version restored", "success");
  } catch (error: unknown) {
    toastManager.value?.addToast(`Failed to restore version: ${errorMessage(error)}`, "error");
  }
}

async function addSpellAllowlistWord(word: string) {
  return workspace.addAllowlistWord(word);
}

async function removeSpellAllowlistWord(word: string) {
  return workspace.removeAllowlistWord(word);
}

// Programmatic-load sentinel for the autosave watcher. Set right
// before any host-driven `activeContent.value = …` assignment so the
// watcher knows the next tick is a load, not a user edit. The sentinel
// is scoped to (file, content): an external-file load (activeFile=null)
// can't poison a later library-file save with the same bytes, and the
// scoped match clears itself on first consumption so a subsequent
// matching keystroke isn't swallowed.
const lastLoaded = ref<{ file: string; content: string } | null>(null);
function markProgrammaticLoad(file: string, content: string) {
  lastLoaded.value = { file, content };
}

watch(activeContent, (newContent) => {
  if (!workspace.state.activeFile) return;
  if (
    lastLoaded.value &&
    lastLoaded.value.file === workspace.state.activeFile &&
    lastLoaded.value.content === newContent
  ) {
    lastLoaded.value = null;
    return;
  }
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = window.setTimeout(() => {
    saveTimer = null;
    workspace.saveFile(newContent);
  }, 1000);
});
</script>

<template>
  <div class="h-screen flex flex-col bg-page-glow dark:bg-page-glow text-text-main overflow-hidden font-sans transition-colors duration-300">
    <div v-if="!isLoaded" class="flex-1 flex items-center justify-center text-text-muted italic bg-[var(--color-page-bg)]">
      Loading Downstage editor...
    </div>

    <div v-else-if="!workspace.state.libraryPath" class="flex-1 flex items-center justify-center bg-page-glow p-8">
        <div class="max-w-2xl w-full text-center">
            <div class="mb-12 inline-flex items-center justify-center w-20 h-20 rounded-3xl bg-brass-500/10 text-brass-500 shadow-inner border border-brass-500/20">
                <BookOpen class="w-10 h-10" />
            </div>

            <h2 class="text-4xl font-serif font-bold text-text-main mb-4 tracking-tight">Library unavailable</h2>
            <p class="text-lg text-text-muted mb-12 max-w-lg mx-auto leading-relaxed">
                Downstage Write couldn't find your library. Open Settings to point it at a folder where your plays live.
            </p>

            <div class="max-w-xs mx-auto">
                <button
                    @click="() => dispatcher?.dispatch('file.settings')"
                    class="flex flex-col items-center gap-4 p-8 rounded-2xl bg-[var(--color-page-surface)] border border-border hover:border-brass-500/50 hover:bg-black/5 dark:hover:bg-white/5 transition-all group text-left w-full"
                >
                    <div class="w-12 h-12 rounded-xl bg-brass-500/10 text-brass-500 flex items-center justify-center group-hover:scale-110 transition-transform">
                        <FolderOpen class="w-6 h-6" />
                    </div>
                    <div>
                        <h3 class="font-bold text-text-main text-lg mb-1">Open Settings</h3>
                        <p class="text-sm text-text-muted">Set your library location.</p>
                    </div>
                </button>
            </div>

            <div class="mt-16 flex items-center justify-center gap-8 text-xs font-bold uppercase tracking-widest text-text-muted opacity-40">
                <div class="flex items-center gap-2"><Terminal class="w-4 h-4" /> CLI Compatible</div>
                <div class="flex items-center gap-2"><Sparkles class="w-4 h-4" /> Live Preview</div>
                <div class="flex items-center gap-2"><FolderSync class="w-4 h-4" /> Auto-Save</div>
            </div>
        </div>
    </div>

    <main v-else class="flex-1 overflow-hidden flex relative">
      <aside
        v-if="!workspace.state.sidebarCollapsed && workspace.state.libraryPath"
        :style="{ width: workspace.state.sidebarWidth + 'px' }"
        class="border-r border-border bg-[var(--color-page-surface)] flex flex-col shrink-0"
      >
        <div class="p-4 border-b border-border flex justify-between items-start bg-black/[0.02] dark:bg-white/[0.02]">
          <div class="min-w-0">
            <h3 class="text-[10px] uppercase tracking-[0.2em] text-brass-500 font-bold">Library</h3>
            <p class="text-[10px] text-text-muted truncate mt-1 italic" :title="workspace.state.libraryPath">{{ workspace.state.libraryPath }}</p>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <button
              @click="workspace.toggleSidebar()"
              class="p-1 rounded text-text-muted hover:text-brass-500 hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
              title="Collapse sidebar"
            >
              <PanelLeftClose class="w-4 h-4" />
            </button>
          </div>
        </div>
        <LibraryTree
          :workspace="workspace"
          @select-file="selectLibraryFile"
          @error="(message) => toastManager?.addToast(message, 'error')"
          @info="(message) => toastManager?.addToast(message, 'info')"
          @request-new-folder="openNewFolderPrompt"
          @request-delete-file="requestDeleteFromTree"
          @request-restore-file="requestRestoreFromTree"
          @request-permanent-delete="requestPermanentDeleteFromTree"
        />

        <VersionsPanel
          :workspace="workspace"
          :is-viewing-revision="isViewingRevision"
          @view-revision="handleViewRevision"
          @exit-revision-view="handleExitRevisionView"
          @compare-to-current="handleCompareToCurrent"
          @error="(message) => toastManager?.addToast(message, 'error')"
          @info="(message) => toastManager?.addToast(message, 'success')"
        />
      </aside>

      <div
        v-if="!workspace.state.sidebarCollapsed && workspace.state.libraryPath"
        class="sidebar-resize-handle shrink-0"
        role="separator"
        aria-orientation="vertical"
        :aria-valuenow="workspace.state.sidebarWidth"
        title="Drag to resize sidebar — double-click to reset"
        @mousedown.prevent="beginSidebarDrag"
        @dblclick="resetSidebarWidth"
      ></div>

      <div class="flex-1 relative flex flex-col overflow-hidden bg-[var(--color-page-bg)]">
        <div
          v-if="workspace.state.isLoadingFile"
          class="absolute inset-0 z-20 flex items-center justify-center bg-[var(--color-page-bg)]/70 text-text-muted italic text-sm"
        >
          Loading file…
        </div>
        <div
            v-if="isViewingExternal && workspace.state.externalFile"
            class="flex items-center justify-between gap-3 px-4 py-2.5 bg-amber-500/10 border-b border-amber-500/30 text-ember-950 dark:text-amber-100 shadow-inner z-10"
        >
            <div class="flex items-center gap-3 min-w-0">
                <ExternalLink class="w-4 h-4 shrink-0 text-amber-600" />
                <div class="min-w-0">
                    <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-700 dark:text-amber-300">Viewing a file outside your library</p>
                    <p class="text-xs truncate opacity-80" :title="workspace.state.externalFile.absPath">
                        {{ externalFileBasename }} — read-only. Add to your library to keep editing.
                    </p>
                </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
                <button
                    type="button"
                    @click="handleAddExternalFileToLibrary"
                    class="inline-flex items-center gap-1.5 rounded-lg bg-brass-500 px-3 py-1.5 text-xs font-bold text-ember-950 transition-colors hover:bg-brass-400"
                    title="Copy this file into your library and open it for editing."
                >
                    <FolderPlus class="w-3.5 h-3.5" />
                    Add to Library
                </button>
                <button
                    type="button"
                    @click="handleCloseExternalFile"
                    class="inline-flex items-center gap-1.5 rounded-lg border border-ember-950/10 dark:border-amber-100/20 px-2.5 py-1.5 text-xs font-bold text-ember-950/80 dark:text-amber-100/80 transition-colors hover:bg-ember-950/5 dark:hover:bg-amber-100/10"
                    title="Close this file and return to the library"
                >
                    <X class="w-3.5 h-3.5" />
                    Close
                </button>
            </div>
        </div>
        <div
            v-if="isViewingRevision && !inCompareTwo && workspace.state.viewingRevisionMeta"
            class="flex items-center justify-between gap-3 px-4 py-2.5 bg-amber-500/10 border-b border-amber-500/30 text-ember-950 dark:text-amber-100 shadow-inner z-10"
        >
            <div class="flex items-center gap-3 min-w-0">
                <History class="w-4 h-4 shrink-0 text-amber-600" />
                <div class="min-w-0">
                    <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-700 dark:text-amber-300">Viewing older version</p>
                    <p class="text-xs truncate opacity-80">
                        "{{ workspace.state.viewingRevisionMeta.message }}" — saved {{ formatRevisionTimestamp(workspace.state.viewingRevisionMeta.timestamp) }}
                    </p>
                </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
                <button
                    type="button"
                    @click="workspace.toggleRevisionCompare()"
                    class="inline-flex items-center gap-1.5 rounded-lg border border-ember-950/10 dark:border-amber-100/20 px-2.5 py-1.5 text-xs font-bold text-ember-950/80 dark:text-amber-100/80 transition-colors hover:bg-ember-950/5 dark:hover:bg-amber-100/10"
                    :title="workspace.state.revisionViewMode === 'compare' ? 'Return to single-pane revision preview' : 'Show changes between this version and the current buffer'"
                >
                    <GitCompare class="w-3.5 h-3.5" />
                    {{ workspace.state.revisionViewMode === 'compare' ? 'Hide compare' : 'Compare to current' }}
                </button>
                <button
                    type="button"
                    @click="handleRestoreRevision"
                    class="inline-flex items-center gap-1.5 rounded-lg bg-brass-500 px-3 py-1.5 text-xs font-bold text-ember-950 transition-colors hover:bg-brass-400"
                    title="Replace the current version with this one. A backup snapshot of the current version is saved first."
                >
                    <RotateCcw class="w-3.5 h-3.5" />
                    Restore this version
                </button>
                <button
                    type="button"
                    @click="handleExitRevisionView"
                    class="inline-flex items-center gap-1.5 rounded-lg border border-ember-950/10 dark:border-amber-100/20 px-2.5 py-1.5 text-xs font-bold text-ember-950/80 dark:text-amber-100/80 transition-colors hover:bg-ember-950/5 dark:hover:bg-amber-100/10"
                    title="Return to the current editable version"
                >
                    <X class="w-3.5 h-3.5" />
                    Return to current
                </button>
            </div>
        </div>
        <div
            v-if="inCompareTwo && workspace.state.viewingRevisionMeta && workspace.state.compareSecondMeta"
            class="flex items-center justify-between gap-3 px-4 py-2.5 bg-amber-500/10 border-b border-amber-500/30 text-ember-950 dark:text-amber-100 shadow-inner z-10"
        >
            <div class="flex items-center gap-3 min-w-0">
                <GitCompare class="w-4 h-4 shrink-0 text-amber-600" />
                <div class="min-w-0">
                    <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-700 dark:text-amber-300">Comparing versions</p>
                    <p class="text-xs truncate opacity-80">
                        "{{ workspace.state.viewingRevisionMeta.message }}"
                        ({{ formatRevisionTimestamp(workspace.state.viewingRevisionMeta.timestamp) }})
                        vs
                        "{{ workspace.state.compareSecondMeta.message }}"
                        ({{ formatRevisionTimestamp(workspace.state.compareSecondMeta.timestamp) }})
                    </p>
                </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
                <button
                    type="button"
                    @click="handleExitRevisionView"
                    class="inline-flex items-center gap-1.5 rounded-lg border border-ember-950/10 dark:border-amber-100/20 px-2.5 py-1.5 text-xs font-bold text-ember-950/80 dark:text-amber-100/80 transition-colors hover:bg-ember-950/5 dark:hover:bg-amber-100/10"
                    title="Stop comparing and return to the current version"
                >
                    <X class="w-3.5 h-3.5" />
                    Stop comparing
                </button>
            </div>
        </div>
        <div
            v-if="workspace.state.activeFile || isViewingExternal"
            v-show="!inCompareDiff"
            class="flex-1 flex flex-col overflow-hidden min-h-0"
        >
          <Editor
              ref="editorRef"
              :env="env as EditorEnv"
              :document-key="editorDocumentKey"
              :read-only="isEditorReadOnly"
              v-model:content="editorContent"
              v-model:style="pageStyle"
              v-model:preview-hidden="store.state.previewHidden"
              v-model:spellcheck-disabled="store.state.spellcheckDisabled"
              v-model:drawer-open="drawerOpen"
              v-model:drawer-tab="drawerTab"
              v-model:search-request="searchRequest"
              :external-spellcheck="true"
              :get-spell-allowlist="() => workspace.state.spellAllowlist"
              :add-spell-allowlist-word="addSpellAllowlistWord"
              :remove-spell-allowlist-word="removeSpellAllowlistWord"
              :drawer-dock="workspace.state.drawerDock"
              :drawer-right-width="workspace.state.drawerRightWidth"
              @migration-state-change="isV1Document = $event"
              @open-spellcheck-settings="() => dispatcher?.dispatch('file.settings.spellcheck')"
              @update:cursor="cursor = $event"
              @update:wordCount="wordCount = $event"
              @update:drawerDock="workspace.setDrawerDock($event)"
              @update:drawerRightWidth="workspace.setDrawerRightWidth($event)"
          >
            <template #leadingActions>
              <button
                v-if="workspace.state.libraryPath"
                type="button"
                @click="workspace.toggleSidebar()"
                class="p-1.5 rounded-md hover:bg-black/5 dark:hover:bg-white/5 text-text-muted transition-colors"
                :title="workspace.state.sidebarCollapsed ? 'Open Library' : 'Close Library'"
              >
                <FolderOpen class="w-4 h-4" />
              </button>
            </template>
          </Editor>
        </div>
        <RevisionDiffView
            v-if="inCompareDiff && workspace.state.viewingRevisionContent !== null"
            class="flex-1 flex flex-col overflow-hidden min-h-0"
            :original="workspace.state.viewingRevisionContent"
            :modified="inCompareTwo
                ? (workspace.state.compareSecondContent ?? '')
                : activeContent"
            :original-label="compareOriginalLabel"
            :modified-label="inCompareTwo ? compareSecondLabel : 'Current'"
            :is-dark="store.state.isDark"
            :env="env as EditorEnv"
        />
        <div
            v-if="!(workspace.state.activeFile || isViewingExternal)"
            class="flex-1 flex flex-col items-center justify-center text-text-muted p-12 text-center"
        >
            <div class="w-16 h-16 rounded-full bg-black/5 dark:bg-white/5 flex items-center justify-center mb-4 text-brass-500">
                <BookOpen class="w-8 h-8 opacity-40" />
            </div>
            <h3 class="text-lg font-serif font-bold text-text-main mb-2">Open a script</h3>
            <p class="text-sm max-w-xs mx-auto mb-6">
              {{
                workspace.state.sidebarCollapsed
                  ? 'Show the file list to pick an existing play, or start a new one.'
                  : 'Select a file from the sidebar to start writing, or create a new manuscript.'
              }}
            </p>
            <div class="flex gap-2">
              <button
                v-if="workspace.state.sidebarCollapsed && workspace.state.libraryPath"
                @click="workspace.toggleSidebar()"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl border border-border text-text-main font-bold text-sm hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
              >
                <FolderOpen class="w-4 h-4" />
                Show Files
              </button>
              <button
                @click="handleNewPlay"
                class="px-5 py-2.5 rounded-xl bg-brass-500 text-ember-950 font-bold text-sm shadow-lg hover:bg-brass-400 transition-all transform hover:scale-105 active:scale-95"
              >
                New Play
              </button>
            </div>
        </div>
      </div>
    </main>

    <StatusBar
      :library-name="libraryNameBase"
      :active-file="activeFileBase"
      :cursor="cursor"
      :word-count="wordCount"
      :git-status="workspace.state.gitStatus"
      :has-library="!!workspace.state.libraryPath"
      :has-active-file="!!workspace.state.activeFile"
      :library-dirty-count="workspace.state.libraryDirty?.count ?? 0"
      @reveal-library="handleRevealLibrary"
      @review-library-changes="openReviewChanges"
    />
    <ReviewChangesModal
      :open="reviewChangesOpen"
      :dirty="workspace.state.libraryDirty"
      :busy="reviewBusy"
      @close="reviewChangesOpen = false"
      @commit="onReviewCommit"
      @discard="onReviewDiscard"
      @refresh="() => workspace.refreshLibraryDirty()"
    />
    <ToastManager ref="toastManager" />
    <CommandPalette
      :open="paletteOpen"
      :mode="paletteMode"
      :env="env"
      :library-files="workspace.libraryFiles.value"
      :disabled-ids="dispatcher?.disabledIds() ?? []"
      @close="paletteOpen = false"
      @select-file="async (path: string) => { paletteOpen = false; await selectLibraryFile(path); }"
    />
    <Settings
      :open="settingsOpen"
      :tab="settingsTab"
      :store="store"
      :workspace="workspace"
      :env="env"
      @close="settingsOpen = false"
      @change-library="handleChangeLibraryLocation"
    />
    <PromptModal
      :open="newFolderOpen"
      title="New Folder"
      :label="newFolderParent ? `Folder name in ${newFolderParent}` : 'Folder name'"
      placeholder="Act One"
      submit-label="Create folder"
      :error="newFolderError"
      @close="() => { newFolderOpen = false; newFolderError = null; }"
      @submit="submitNewFolder"
    />
    <PromptModal
      :open="saveVersionOpen"
      title="Save Version"
      label="Version name"
      placeholder="Snapshot"
      :initial-value="saveVersionInitial"
      submit-label="Save"
      :error="saveVersionError"
      @close="() => { saveVersionOpen = false; saveVersionError = null; }"
      @submit="submitSaveVersion"
    />
    <ExportPdfModal
      :open="exportDialogOpen"
      :initial-options="exportInitialOptions"
      hide-page-size
      @close="exportDialogOpen = false"
      @confirm="handleExportConfirmed"
    />
    <ConfirmModal
      v-if="confirmConfig"
      :open="confirmOpen"
      :title="confirmConfig.title"
      :message="confirmConfig.message"
      :confirm-label="confirmConfig.confirmLabel"
      :destructive="confirmConfig.destructive"
      @close="onConfirmClosed"
      @confirm="onConfirmAccepted"
    />
  </div>
</template>

<style>
.sidebar-resize-handle {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background-color 0.12s ease-out;
}
.sidebar-resize-handle:hover,
.sidebar-resize-handle:active {
  background: rgba(227, 168, 87, 0.25); /* brass-500 @ ~25% */
}

.custom-scrollbar::-webkit-scrollbar { width: 6px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.1); border-radius: 10px; }
.dark .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); }
.custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(0, 0, 0, 0.2); }
.dark .custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(255, 255, 255, 0.2); }
</style>
