<script setup lang="ts">
import { computed, provide, onMounted, onUnmounted, ref, watch, watchEffect } from 'vue';
import {
    FolderOpen, FolderSync, FileText, FolderPlus,
    BookOpen, Terminal, Sparkles, History, ExternalLink,
    RotateCcw, X, PanelLeftClose, Plus
} from 'lucide-vue-next';
import { Store } from './core/store';
import type { EditorEnv } from './core/types';
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
import StatusBar from './desktop/StatusBar.vue';

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

// Status-bar-bound editor telemetry. Editor.vue emits `update:cursor`
// (1-based line/col) on every selection or doc change, and
// `update:wordCount` whenever the 500ms stats debounce resolves.
// Desktop host collects both into refs so the StatusBar can render.
const cursor = ref<{ line: number; col: number }>({ line: 1, col: 1 });
const wordCount = ref(0);

// Sidebar drag state. Mouse handlers are attached to `window` on drag
// start so the mouse can leave the handle hitbox and still resize; they
// detach on mouseup. `startX` and `startWidth` are captured at drag
// start so intermediate moves don't accumulate floating error.
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

// Host-owned drawer + search state (lifted out of Editor.vue so commands
// can open specific tabs / trigger search from the menu).
const drawerOpen = ref(false);
const drawerTab = ref<WorkbenchTab>('issues');
const searchRequest = ref<{ mode: SearchMode; nonce: number }>({ mode: 'find', nonce: 0 });

// Host-owned palette + settings dialog state.
const paletteOpen = ref(false);
const paletteMode = ref<'command' | 'file'>('command');
const settingsOpen = ref(false);
const settingsTab = ref<'library' | 'appearance' | 'spellcheck'>('library');

function openPalette(mode: 'command' | 'file' = 'command') {
  paletteMode.value = mode;
  paletteOpen.value = true;
}
function openSettings(tab: 'library' | 'appearance' | 'spellcheck' = 'library') {
  settingsTab.value = tab;
  settingsOpen.value = true;
}

let saveTimer: number | null = null;

// Dispatcher is instantiated on mount (needs env + refs ready) and is
// referenced by both the menu event listener (via dispatcher-registry)
// and by local UI handlers (sidebar buttons, welcome screen) that want
// to dispatch the same commands the menu does.
let dispatcher: CommandDispatcher | null = null;

// Basename helpers used by the status bar.
const libraryNameBase = computed(
  () => workspace.state.libraryPath?.split(/[\\/]/).pop() ?? '',
);
const activeFileBase = computed(
  () => workspace.state.activeFile?.split(/[\\/]/).pop() ?? '',
);

// `editorContent` is what the editor shows. Three branches:
//   - external-file view (read-only) → externalFile.content
//   - revision view (read-only) → viewingRevisionContent
//   - live buffer → activeContent
// The setter drops writes in both read-only modes; the editor is also
// read-only in those modes, so this is belt-and-suspenders.
const isViewingRevision = computed(
  () => workspace.state.viewingRevisionHash !== null,
);
const isViewingExternal = computed(
  () => workspace.state.externalFile !== null,
);
const isEditorReadOnly = computed(
  () => isViewingRevision.value || isViewingExternal.value,
);

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

// documentKey identifies the buffer shown in the editor. External and
// revision views get distinct keys so the shared editor resets transient
// state (diagnostics, search, stats, outline) when toggling modes.
const editorDocumentKey = computed(() => {
  if (isViewingExternal.value) {
    return `external:${workspace.state.externalFile?.absPath ?? ""}`;
  }
  if (!workspace.state.activeFile) return null;
  return workspace.state.viewingRevisionHash
    ? `${workspace.state.activeFile}@${workspace.state.viewingRevisionHash}`
    : workspace.state.activeFile;
});

function formatRevisionTimestamp(ts: string): string {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

onMounted(async () => {
  await store.init();
  await workspace.init();

  if (workspace.state.libraryPath && workspace.libraryFiles.value.length > 0) {
    const lastFile = await props.env.getLastActiveFile();
    const exists = workspace.libraryFiles.value.some(f => f.path === lastFile);
    if (lastFile && exists) {
      activeContent.value = await workspace.selectFile(lastFile);
    } else {
      activeContent.value = await workspace.selectFile(workspace.libraryFiles.value[0].path);
    }
  }

  // Drawer-tab restore. Workspace.init() hydrated `lastDrawerTab`; map
  // "" → 'issues' default. Then watch the ref and persist every
  // user-driven change so the next launch opens on the same tab.
  if (workspace.state.lastDrawerTab) {
    drawerTab.value = workspace.state.lastDrawerTab as WorkbenchTab;
  }
  watch(drawerTab, (next) => {
    workspace.setLastDrawerTab(next);
  });

  // Live-save window bounds on resize. Position is captured at quit via
  // BeforeClose (Wails has no cross-platform move event); size changes
  // fire here. Debounced so a drag doesn't hammer Go. The backend
  // refuses to overwrite normal bounds while maximized, so a maximized
  // resize is a no-op.
  window.addEventListener('resize', scheduleBoundsSave);

  isLoaded.value = true;
  registerFlushSave(() => flushSave());

  // Dispatcher setup: build the context, wire handlers, register the
  // dispatcher with the module-scope event subscriber, and kick off the
  // first disabled-set push so the menu renders with the right Disabled
  // flags on launch.
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
    pageStyle,
    isV1Document,
    isViewingRevision,
    flushSave,
    editor: {
      applyFormat: (action: string) => editorRef.value?.applyFormat(action),
    },
    ui: {
      drawerOpen,
      drawerTab,
      searchRequest,
      openPalette,
      openSettings,
    },
  };
  for (const [id, entry] of createCommandHandlers(ctx)) {
    dispatcher.register(id, entry);
  }
  registerDispatcher(dispatcher);

  // React to any state that an isEnabled predicate might read. Vue
  // batches these into one microtask; the dispatcher further diffs the
  // resulting set against the last one it sent to Go so quiet state
  // changes produce zero wire traffic.
  watchEffect(() => {
    // Touch the reactive fields the predicates care about so watchEffect
    // tracks them. Running dispatcher.scheduleRefresh inside the effect
    // is what actually kicks the microtask.
    void workspace.state.activeFile;
    void workspace.libraryFiles.value.length;
    void workspace.state.viewingRevisionHash;
    void isV1Document.value;
    dispatcher?.scheduleRefresh();
  });
});

onUnmounted(() => {
  registerFlushSave(null);
  registerDispatcher(null);
  window.removeEventListener('resize', scheduleBoundsSave);
  if (boundsSaveTimer !== null) {
    clearTimeout(boundsSaveTimer);
    boundsSaveTimer = null;
  }
  void flushSave();
  // Best-effort: drain any in-flight preference writes if the app tears
  // down through component unmount rather than through the Wails
  // before-close path (e.g. during tests, or SPA-style navigation).
  void props.env.flushPreferences();
});

// Debounce handle for the window-resize → saveWindowBoundsIfNormal
// pipeline. 500ms lets a user's drag settle before we hit Go, which
// is the window between "still dragging" and "final size reached"
// visible to most humans.
let boundsSaveTimer: ReturnType<typeof setTimeout> | null = null;
function scheduleBoundsSave() {
  if (boundsSaveTimer !== null) clearTimeout(boundsSaveTimer);
  boundsSaveTimer = setTimeout(() => {
    boundsSaveTimer = null;
    void props.env.saveWindowBoundsIfNormal();
  }, 500);
}

// flushSave resolves when any pending debounced write is durable on disk.
// It must be awaited before any state transition that could clobber
// `workspace.state.activeFile` or change the project root — otherwise the
// inner guard in `workspace.saveFile` drops the write silently.
async function flushSave(): Promise<void> {
  if (!saveTimer) return;
  clearTimeout(saveTimer);
  saveTimer = null;
  if (workspace.state.activeFile) {
    await workspace.saveFile(activeContent.value);
  }
}

// Welcome screen / empty-sidebar "New Play" button fires this.
function handleNewPlay() {
  void dispatcher?.dispatch('file.newPlay');
}

// Status-bar library label click: open the library folder in the host
// OS's file explorer. The library location itself is managed through
// Settings, so this click has a narrower job than "open folder".
function handleRevealLibrary() {
  void env.revealLibraryInExplorer();
}

// External-file banner helpers. openExternalFile is invoked via the
// file.open command; these handlers wire the banner buttons.
const externalFileBasename = computed(() => {
  const abs = workspace.state.externalFile?.absPath ?? '';
  return abs.split(/[\\/]/).pop() || abs;
});

async function handleAddExternalFileToLibrary() {
  try {
    const rel = await workspace.addExternalFileToLibrary("");
    toastManager.value?.addToast(`Added ${rel} to your library`, "success");
  } catch (e: any) {
    toastManager.value?.addToast(
      `Failed to add file to library: ${e?.message ?? e}`,
      "error",
    );
  }
}

function handleCloseExternalFile() {
  workspace.closeExternalFile();
  activeContent.value = "";
}

// Settings > Library "Change…" button emits this. Runs the full switch
// flow: flush the live buffer, change location via workspace, toast,
// re-select the first file in the new library if any. Kept here (not
// in commands.ts) because it's only invoked from Settings, not from
// the menu / palette dispatcher.
async function handleChangeLibraryLocation() {
  await flushSave();
  const path = await workspace.changeLibraryLocation();
  if (!path) return;
  activeContent.value = "";
  if (workspace.libraryFiles.value.length > 0) {
    activeContent.value = await workspace.selectFile(workspace.libraryFiles.value[0].path);
  }
  toastManager.value?.addToast(`Opened library: ${path.split(/[\\/]/).pop()}`, "success");
}

async function selectLibraryFile(path: string) {
  await flushSave();
  activeContent.value = await workspace.selectFile(path);
}

async function handleViewRevision(hash: string) {
  // Flush in-flight edits before switching buffers so unwritten changes
  // aren't lost when the user exits the preview.
  await flushSave();
  try {
    await workspace.viewRevision(hash);
  } catch (e: any) {
    toastManager.value?.addToast(
      `Failed to load version: ${e?.message ?? e}`,
      "error",
    );
  }
}

function handleExitRevisionView() {
  workspace.clearRevisionView();
}

async function handleRestoreRevision() {
  const hash = workspace.state.viewingRevisionHash;
  if (!hash) return;
  try {
    // Pass the live buffer (not the revision content) — restoreRevision
    // snapshots it before overwriting so the restore is itself reversible.
    const restored = await workspace.restoreRevision(hash, activeContent.value);
    activeContent.value = restored;
    toastManager.value?.addToast("Version restored", "success");
  } catch (e: any) {
    toastManager.value?.addToast(
      `Failed to restore version: ${e?.message ?? e}`,
      "error",
    );
  }
}

// File-level commands (new play, open folder, copy all, save version,
// export PDF) live in web/src/desktop/commands.ts. They're registered
// with the CommandDispatcher in onMounted below. Sidebar-contextual
// actions (file switch, revision view/restore) stay as local functions
// because they're not on the menu or palette.

async function addSpellAllowlistWord(word: string) {
  return workspace.addAllowlistWord(word);
}

async function removeSpellAllowlistWord(word: string) {
  return workspace.removeAllowlistWord(word);
}

watch(activeContent, (newContent) => {
  if (!workspace.state.activeFile) return;
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

    <!-- Welcome Screen. initApp auto-creates the default library at
         ~/Documents/Downstage Plays so this view is only reached in the
         degenerate case (deleted library, permissions issue). The
         remedy is Settings — offer a direct path there and nothing
         else. -->
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
      <!-- Project Sidebar -->
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
        />

        <!-- Revisions Section -->
        <div v-if="workspace.state.activeFile" class="h-1/3 flex flex-col bg-black/[0.01] dark:bg-white/[0.01]">
            <div class="p-3 border-b border-border bg-black/[0.02] dark:bg-white/[0.02]">
                <h3 class="text-[10px] uppercase tracking-[0.2em] text-text-muted font-bold flex items-center gap-2">
                    <History class="w-3.5 h-3.5 opacity-50" /> Versions
                </h3>
            </div>
            <div class="flex-1 overflow-y-auto custom-scrollbar p-2 space-y-1">
                <button
                    v-if="workspace.state.revisions.length > 0"
                    type="button"
                    @click="handleExitRevisionView"
                    class="w-full text-left p-2 rounded transition-colors border"
                    :class="!isViewingRevision
                        ? 'bg-brass-500/10 border-brass-500/20 text-brass-500 font-bold'
                        : 'border-transparent hover:bg-black/5 dark:hover:bg-white/5 text-text-main'"
                    :title="isViewingRevision ? 'Return to current version' : 'Current version'"
                >
                    <div class="text-[11px] font-bold truncate flex items-center gap-1.5">
                        <FolderSync v-if="!isViewingRevision" class="w-3 h-3" />
                        <span>Current (editing)</span>
                    </div>
                </button>
                <button
                    v-for="rev in workspace.state.revisions"
                    :key="rev.hash"
                    type="button"
                    @click="handleViewRevision(rev.hash)"
                    class="w-full text-left p-2 rounded transition-colors border"
                    :class="workspace.state.viewingRevisionHash === rev.hash
                        ? 'bg-brass-500/10 border-brass-500/20 text-brass-500'
                        : 'border-transparent hover:bg-black/5 dark:hover:bg-white/5 text-text-main'"
                    :title="`Preview this version (${formatRevisionTimestamp(rev.timestamp)})`"
                >
                    <div class="text-[11px] font-bold truncate">{{ rev.message }}</div>
                    <div class="flex justify-end items-center mt-1">
                        <span class="text-[9px] text-text-muted italic">{{ formatRevisionTimestamp(rev.timestamp) }}</span>
                    </div>
                </button>
                <div v-if="workspace.state.revisions.length === 0" class="p-4 text-center">
                    <p class="text-[10px] text-text-muted italic">No versions yet. Click "Save Version" to create one.</p>
                </div>
            </div>
        </div>
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
            v-if="isViewingRevision && workspace.state.viewingRevisionMeta"
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
        <Editor
            v-if="workspace.state.activeFile || isViewingExternal"
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
              :title="workspace.state.sidebarCollapsed ? 'Open Projects' : 'Close Projects'"
            >
              <FolderOpen class="w-4 h-4" />
            </button>
          </template>
        </Editor>
        <div v-else class="flex-1 flex flex-col items-center justify-center text-text-muted p-12 text-center">
            <div class="w-16 h-16 rounded-full bg-black/5 dark:bg-white/5 flex items-center justify-center mb-4 text-brass-500">
                <BookOpen class="w-8 h-8 opacity-40" />
            </div>
            <h3 class="text-lg font-serif font-bold text-text-main mb-2">Open a script</h3>
            <p class="text-sm max-w-xs mx-auto mb-6">Select a file from the sidebar to start writing, or create a new manuscript.</p>
            <button @click="handleNewPlay" class="px-5 py-2.5 rounded-xl bg-brass-500 text-ember-950 font-bold text-sm shadow-lg hover:bg-brass-400 transition-all transform hover:scale-105 active:scale-95">New Play</button>
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
      @reveal-library="handleRevealLibrary"
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
