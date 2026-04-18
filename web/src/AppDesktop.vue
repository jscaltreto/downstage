<script setup lang="ts">
import { computed, provide, onMounted, onUnmounted, ref, watch } from 'vue';
import {
    Plus, FolderOpen, Copy, FolderSync, FileText, FileOutput, Sun, Moon,
    BookOpen, Terminal, Sparkles, PanelLeftClose, PanelLeft, History, Save,
    RotateCcw, X
} from 'lucide-vue-next';
import { Store } from './core/store';
import type { EditorEnv } from './core/types';
import type { DesktopCapabilities } from './desktop/types';
import { Workspace } from './desktop/workspace';
import { registerFlushSave } from './desktop/flush-save';
import ToolbarButton from './components/shared/ToolbarButton.vue';
import ToastManager from './components/shared/ToastManager.vue';
import Editor from './components/shared/Editor.vue';

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

const toastManager = ref<InstanceType<typeof ToastManager> | null>(null);

let saveTimer: number | null = null;

// `editorContent` is what the editor shows. While viewing an older
// revision, we route the revision's content into the editor and keep the
// live buffer in `activeContent` so exiting the view restores in-flight
// edits. `editorContent` is a two-way binding; the setter drops writes
// while viewing (editor is also read-only, so this is belt-and-suspenders).
const isViewingRevision = computed(
  () => workspace.state.viewingRevisionHash !== null,
);

const editorContent = computed<string>({
  get: () =>
    isViewingRevision.value
      ? workspace.state.viewingRevisionContent ?? ""
      : activeContent.value,
  set: (value: string) => {
    if (isViewingRevision.value) return;
    activeContent.value = value;
  },
});

// documentKey identifies the buffer shown in the editor. While viewing a
// revision, append the hash so the shared editor resets transient state
// (diagnostics, search, stats, outline) when toggling in and out of view.
const editorDocumentKey = computed(() => {
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

  if (workspace.state.projectPath && workspace.state.projectFiles.length > 0) {
    const lastFile = await props.env.getLastActiveFile();
    const exists = workspace.state.projectFiles.some(f => f.path === lastFile);
    if (lastFile && exists) {
      activeContent.value = await workspace.selectFile(lastFile);
    } else {
      activeContent.value = await workspace.selectFile(workspace.state.projectFiles[0].path);
    }
  }

  isLoaded.value = true;
  registerFlushSave(() => flushSave());
});

onUnmounted(() => {
  registerFlushSave(null);
  void flushSave();
});

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

async function handleOpenFolder() {
  // Flush the previous project's pending edit BEFORE the folder switch —
  // `workspace.openFolder` clears `state.activeFile` on success, which
  // would cause an unawaited flush to no-op.
  await flushSave();

  const path = await workspace.openFolder();
  if (!path) return;

  activeContent.value = "";

  if (workspace.state.projectFiles.length > 0) {
    activeContent.value = await workspace.selectFile(workspace.state.projectFiles[0].path);
  }
  toastManager.value?.addToast(`Opened project: ${path.split(/[\\/]/).pop()}`, "success");
}

async function selectProjectFile(path: string) {
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

async function handleNewPlay() {
  if (!workspace.state.projectPath) {
    await handleOpenFolder();
    if (!workspace.state.projectPath) return;
  }

  await flushSave();

  const template = `# Untitled Play\nSubtitle: A Play in One Act\nAuthor: Your Name\nDate: ${new Date().getFullYear()}\nDraft: First\n\n## Dramatis Personae\n\nPROTAGONIST - Add your cast here\n\n## ACT I\n\n### SCENE 1\n\n> Describe the setting here.\n\nPROTAGONIST\nWrite your opening lines here.\n`;

  try {
    const path = await workspace.createFile("Untitled Play", template);
    activeContent.value = await workspace.selectFile(path);
    toastManager.value?.addToast("Created new play", "success");
  } catch (e: any) {
    toastManager.value?.addToast(`Failed to create file: ${e.message}`, "error");
  }
}

async function handleCopy() {
  // Copy what's actually on screen. While previewing a revision, that's
  // the revision text, not the live buffer.
  await navigator.clipboard.writeText(editorContent.value);
  toastManager.value?.addToast("Copied to clipboard", "success");
}

const nothingToSnapshotPrefix = "downstage: nothing-to-snapshot";

async function handleSnapshot() {
  if (!workspace.state.activeFile) return;
  // Must await — otherwise the snapshot can race the pending write and
  // commit stale disk contents.
  await flushSave();
  const filename = workspace.state.activeFile.split(/[\\/]/).pop() || "file";
  try {
    await workspace.snapshotFile(`Snapshot ${filename}`);
    toastManager.value?.addToast("Version saved", "success");
  } catch (e: any) {
    const message = String(e?.message ?? e);
    if (message.includes(nothingToSnapshotPrefix)) {
      toastManager.value?.addToast("No changes to snapshot", "info");
    } else {
      toastManager.value?.addToast(`Failed to save version: ${message}`, "error");
    }
  }
}

async function handleExport() {
  if (isV1Document.value) {
    toastManager.value?.addToast("Upgrade this V1 document to V2 before exporting PDF", "error", 5000);
    return;
  }

  await flushSave();
  // Export whatever the user is looking at — live buffer or revision preview.
  const source = editorContent.value;
  const title = source.match(/^#\s+(.+)$/m)?.[1]?.trim() || "untitled";
  const styleSlug = pageStyle.value === "condensed" ? "acting-edition" : "manuscript";
  const filename = `${title.replace(/[^a-z0-9]+/gi, "-").toLowerCase()}-${styleSlug}.pdf`;

  try {
    const pdfBytes = await props.env.renderPDF(source, pageStyle.value);
    await props.env.saveFile(filename, pdfBytes, [
      { displayName: "PDF Files (*.pdf)", pattern: "*.pdf" },
    ]);
  } catch (e: any) {
    toastManager.value?.addToast(`Failed to export PDF: ${e?.message ?? e}`, "error");
  }
}

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
    <header v-if="isLoaded" class="flex items-center justify-between gap-5 px-5 py-3.5 bg-[var(--color-page-surface)] border-b border-border shadow-stage z-10">
      <div class="flex items-center gap-4">
        <button
            @click="workspace.toggleSidebar()"
            class="p-1.5 rounded-md hover:bg-black/5 dark:hover:bg-white/5 text-text-muted transition-colors"
            :title="workspace.state.sidebarCollapsed ? 'Expand Sidebar' : 'Collapse Sidebar'"
        >
            <PanelLeft v-if="workspace.state.sidebarCollapsed" class="w-5 h-5" />
            <PanelLeftClose v-else class="w-5 h-5" />
        </button>

        <h1 class="font-serif text-xl font-bold text-text-main tracking-tight cursor-default">Downstage Write</h1>

        <div v-if="workspace.state.projectPath" class="hidden lg:flex items-center gap-2 px-3 py-1 rounded-full bg-black/5 dark:bg-white/5 border border-black/5 dark:border-white/5 max-w-[300px]">
            <FolderSync class="w-3 h-3 text-brass-500 shrink-0" />
            <span class="text-[10px] font-bold text-text-muted truncate uppercase tracking-wider">{{ workspace.state.projectPath.split(/[\\/]/).pop() }}</span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <ToolbarButton @click="handleNewPlay" title="New Play"><template #icon><Plus class="w-4 h-4" /></template>New Play</ToolbarButton>
          <ToolbarButton v-if="workspace.state.activeFile" @click="handleCopy" title="Copy Content"><template #icon><Copy class="w-4 h-4" /></template>Copy</ToolbarButton>
          <ToolbarButton v-if="workspace.state.activeFile && !isViewingRevision" @click="handleSnapshot" title="Save Version"><template #icon><Save class="w-4 h-4" /></template>Save Version</ToolbarButton>
          <ToolbarButton
            v-if="workspace.state.activeFile"
            @click="handleExport"
            :disabled="isV1Document"
            :title="isV1Document ? 'Upgrade this V1 document before exporting PDF' : 'Export to PDF'"
          ><template #icon><FileOutput class="w-4 h-4" /></template>Export PDF</ToolbarButton>
          <button
            @click="store.toggleTheme()"
            class="p-2 rounded-md hover:bg-black/5 dark:hover:bg-white/5 text-text-muted transition-colors"
            :title="store.state.isDark ? 'Switch to Light Theme' : 'Switch to Dark Theme'"
          >
            <Sun v-if="store.state.isDark" class="w-4 h-4" />
            <Moon v-else class="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>

    <div v-if="!isLoaded" class="flex-1 flex items-center justify-center text-text-muted italic bg-[var(--color-page-bg)]">
      Loading Downstage editor...
    </div>

    <!-- Welcome Screen -->
    <div v-else-if="!workspace.state.projectPath" class="flex-1 flex items-center justify-center bg-page-glow p-8">
        <div class="max-w-2xl w-full text-center">
            <div class="mb-12 inline-flex items-center justify-center w-20 h-20 rounded-3xl bg-brass-500/10 text-brass-500 shadow-inner border border-brass-500/20">
                <BookOpen class="w-10 h-10" />
            </div>

            <h2 class="text-4xl font-serif font-bold text-text-main mb-4 tracking-tight">Ready to write?</h2>
            <p class="text-lg text-text-muted mb-12 max-w-lg mx-auto leading-relaxed">
                Downstage Write is a project-based editor. Open a folder to start writing your next masterpiece with local file access and Git versioning.
            </p>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-xl mx-auto">
                <button
                    @click="handleOpenFolder"
                    class="flex flex-col items-center gap-4 p-8 rounded-2xl bg-[var(--color-page-surface)] border border-border hover:border-brass-500/50 hover:bg-black/5 dark:hover:bg-white/5 transition-all group text-left"
                >
                    <div class="w-12 h-12 rounded-xl bg-brass-500/10 text-brass-500 flex items-center justify-center group-hover:scale-110 transition-transform">
                        <FolderOpen class="w-6 h-6" />
                    </div>
                    <div>
                        <h3 class="font-bold text-text-main text-lg mb-1">Open Folder</h3>
                        <p class="text-sm text-text-muted">Select an existing project or a fresh directory.</p>
                    </div>
                </button>

                <button
                    @click="handleNewPlay"
                    class="flex flex-col items-center gap-4 p-8 rounded-2xl bg-[var(--color-page-surface)] border border-border hover:border-brass-500/50 hover:bg-black/5 dark:hover:bg-white/5 transition-all group text-left"
                >
                    <div class="w-12 h-12 rounded-xl bg-ember-600/10 text-ember-600 flex items-center justify-center group-hover:scale-110 transition-transform">
                        <Plus class="w-6 h-6" />
                    </div>
                    <div>
                        <h3 class="font-bold text-text-main text-lg mb-1">New Play</h3>
                        <p class="text-sm text-text-muted">Create a new .ds file in a project folder.</p>
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
        v-if="!workspace.state.sidebarCollapsed && workspace.state.projectPath"
        class="w-64 border-r border-border bg-[var(--color-page-surface)] flex flex-col shrink-0"
      >
        <div class="p-4 border-b border-border flex justify-between items-start bg-black/[0.02] dark:bg-white/[0.02]">
          <div class="min-w-0">
            <h3 class="text-[10px] uppercase tracking-[0.2em] text-brass-500 font-bold">Project Files</h3>
            <p class="text-[10px] text-text-muted truncate mt-1 italic" :title="workspace.state.projectPath">{{ workspace.state.projectPath }}</p>
          </div>
          <button @click="handleOpenFolder" class="text-text-muted hover:text-brass-500 transition-colors" title="Change Project Folder">
            <FolderOpen class="w-4 h-4" />
          </button>
        </div>
        <nav v-if="workspace.state.projectFiles" class="flex-1 overflow-y-auto p-2 space-y-1 custom-scrollbar border-b border-border">
          <button
            v-for="file in workspace.state.projectFiles"
            :key="file.path"
            @click="selectProjectFile(file.path)"
            class="w-full text-left px-3 py-2 rounded text-sm hover:bg-black/5 dark:hover:bg-white/5 transition-colors flex items-center gap-2 group border border-transparent"
            :class="workspace.state.activeFile === file.path ? 'bg-brass-500/10 text-brass-500 font-bold border-brass-500/20 shadow-sm' : 'text-text-main'"
          >
            <FolderSync v-if="workspace.state.activeFile === file.path" class="w-4 h-4 text-brass-500" />
            <FileText v-else class="w-4 h-4 opacity-40 group-hover:opacity-100 transition-opacity text-text-muted" />
            <span class="truncate">{{ file.name }}</span>
          </button>

          <div v-if="workspace.state.projectFiles.length === 0" class="p-4 text-center">
            <p class="text-xs text-text-muted italic text-balance">This folder is empty. Create a new .ds file to get started.</p>
            <button @click="handleNewPlay" class="mt-3 px-3 py-1.5 rounded-lg bg-brass-500/10 text-brass-600 dark:text-brass-400 text-xs font-bold hover:bg-brass-500/20 transition-colors">Create Play</button>
          </div>
        </nav>

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

      <div class="flex-1 relative flex flex-col overflow-hidden bg-[var(--color-page-bg)]">
        <div
          v-if="workspace.state.isLoadingFile"
          class="absolute inset-0 z-20 flex items-center justify-center bg-[var(--color-page-bg)]/70 text-text-muted italic text-sm"
        >
          Loading file…
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
            v-if="workspace.state.activeFile"
            :env="env as EditorEnv"
            :document-key="editorDocumentKey"
            :read-only="isViewingRevision"
            v-model:content="editorContent"
            v-model:style="pageStyle"
            v-model:preview-hidden="store.state.previewHidden"
            v-model:spellcheck-disabled="store.state.spellcheckDisabled"
            :get-spell-allowlist="() => workspace.state.spellAllowlist"
            :add-spell-allowlist-word="addSpellAllowlistWord"
            :remove-spell-allowlist-word="removeSpellAllowlistWord"
            @migration-state-change="isV1Document = $event"
        />
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

    <ToastManager ref="toastManager" />
  </div>
</template>

<style>
.custom-scrollbar::-webkit-scrollbar { width: 6px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.1); border-radius: 10px; }
.dark .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); }
.custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(0, 0, 0, 0.2); }
.dark .custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(255, 255, 255, 0.2); }
</style>
