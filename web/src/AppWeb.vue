<script setup lang="ts">
import { computed, provide, onMounted, ref, watch, onUnmounted } from 'vue';
import {
    Plus, FolderOpen, FileText, Download, Copy, ExternalLink, Trash2, FileOutput, Upload, AlertTriangle
} from 'lucide-vue-next';
import { Store } from './core/store';
import type { EditorEnv, ExportPdfOptions, PdfLayout, PdfPageSize, SavedDraft } from './core/types';
import ToolbarButton from './components/shared/ToolbarButton.vue';
import BaseModal from './components/shared/BaseModal.vue';
import DeleteConfirmationModal from './components/shared/DeleteConfirmationModal.vue';
import ExportPdfModal from './components/shared/ExportPdfModal.vue';
import ToastManager from './components/shared/ToastManager.vue';
import Editor from './components/shared/Editor.vue';
import WelcomeModal from './components/shared/WelcomeModal.vue';

const props = defineProps<{
  env: EditorEnv;
}>();

const store = new Store(props.env);
provide('store', store);

const welcomeStorageKey = "downstage-editor-welcome-dismissed";
const pageSizeStorageKey = "downstage-editor-export-page-size";
const layoutStorageKey = "downstage-editor-export-layout";
const gutterStorageKey = "downstage-editor-export-booklet-gutter";
const letterRegions = new Set(["CA", "MX", "PH", "US"]);

function guessDefaultPageSize(): PdfPageSize {
  if (typeof navigator === "undefined") return "a4";
  const locales = navigator.languages && navigator.languages.length > 0
    ? navigator.languages
    : [navigator.language ?? ""];
  for (const locale of locales) {
    const region = locale.match(/-([A-Z]{2})$/u)?.[1];
    if (region && letterRegions.has(region)) {
      return "letter";
    }
  }
  return "a4";
}

function readStoredPageSize(): PdfPageSize {
  try {
    const stored = localStorage.getItem(pageSizeStorageKey);
    if (stored === "letter" || stored === "a4") {
      return stored;
    }
  } catch {
    // ignore storage errors
  }
  return guessDefaultPageSize();
}

function readStoredLayout(): PdfLayout {
  try {
    const stored = localStorage.getItem(layoutStorageKey);
    if (stored === "single" || stored === "2up" || stored === "booklet") {
      return stored;
    }
  } catch {
    // ignore storage errors
  }
  return "single";
}

function readStoredGutter(): string {
  try {
    const stored = localStorage.getItem(gutterStorageKey);
    if (stored && /^-?[\d.]+\s*(in|mm)$/i.test(stored)) {
      return stored;
    }
  } catch {
    // ignore storage errors
  }
  return "0.125in";
}

const isLoaded = ref(false);
const showDrafts = ref(false);
const showWelcome = ref(false);
const showNewPlayConfirm = ref(false);
const showExportDialog = ref(false);
const exportPageSize = ref<PdfPageSize>(readStoredPageSize());
const exportLayout = ref<PdfLayout>(readStoredLayout());
const exportGutter = ref<string>(readStoredGutter());
const activeContent = ref("");
const pageStyle = ref("standard");
const isV1Document = ref(false);

// Placeholder content shown before the user has made their first edit. Not
// persisted to the drafts list until that first change promotes it to a
// real draft.
const pendingDraft = ref<{ title: string; content: string } | null>(null);

// Toast and Delete state
const toastManager = ref<InstanceType<typeof ToastManager> | null>(null);
const draftToDelete = ref<SavedDraft | null>(null);

let persistTimer: number | null = null;

const newPlayTemplate = `# Untitled Play\nSubtitle: A Play in One Act\nAuthor: Your Name\nDate: ${new Date().getFullYear()}\nDraft: First\n\n## Dramatis Personae\n\nPROTAGONIST - Add your cast here\n\n## ACT I\n\n### SCENE 1\n\n> Describe the setting here.\n\nPROTAGONIST\nWrite your opening lines here.\n`;

const activeSavedDraft = computed(() =>
    store.state.drafts.find(d => d.id === store.state.activeDraftId) || null,
);

function extractDocumentTitle(content: string) {
  return content.match(/^#\s+(.+)$/m)?.[1]?.trim() || null;
}

const exampleContent = `# The Example Play
Subtitle: A Play in One Act
Author: Your Name
Date: 2024
Draft: First

## Dramatis Personae

ALICE - A curious young woman
BOB - Her steadfast companion

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

onMounted(async () => {
  await store.init();

  // Restore URL content if present (handling UTF-8 correctly). Two modes:
  //  - ?content=<b64>  shared/received link, promote to a saved draft
  //    right away so a reload before first edit doesn't drop the share.
  //  - ?try=<b64>      the syntax guide's "Try it" button; user is just
  //    exploring a snippet, so keep it as a pending placeholder until the
  //    first edit, and leave the URL alone so reload re-hydrates it.
  const params = new URLSearchParams(window.location.search);
  const sharedContent = params.get("content");
  const trySnippet = params.get("try");
  if (sharedContent) {
    try {
        const decoded = decodeURIComponent(escape(atob(sharedContent)));
        await createDraft("Imported Play", decoded);
        toastManager.value?.addToast("Imported content from URL", "success");
        window.history.replaceState({}, '', window.location.pathname);
    } catch (e) {
        console.error("Failed to decode shared URL content", e);
    }
  } else if (trySnippet) {
    try {
        const decoded = decodeURIComponent(escape(atob(trySnippet)));
        showPendingPlaceholder("Snippet", decoded);
    } catch (e) {
        console.error("Failed to decode snippet URL content", e);
    }
  } else if (store.state.activeDraftId) {
    const draft = store.state.drafts.find(d => d.id === store.state.activeDraftId);
    if (draft) {
        activeContent.value = draft.content;
    } else if (store.state.drafts.length > 0) {
        await activateDraft(store.state.drafts[0].id);
    }
  }

  // First-run: show the example script as a pending placeholder. It only
  // becomes a saved draft if the user actually edits it.
  if (store.state.drafts.length === 0 && !activeContent.value) {
    showPendingPlaceholder("The Example Play", exampleContent);
  }

  showWelcome.value = localStorage.getItem(welcomeStorageKey) !== "true";
  isLoaded.value = true;

  window.addEventListener("pagehide", flushDrafts);
});

onUnmounted(() => {
    window.removeEventListener("pagehide", flushDrafts);
    flushDrafts();
});

async function createDraft(title: string, content: string) {
  flushDrafts();
  pendingDraft.value = null;
  const id = `draft-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const newDraft: SavedDraft = {
      id,
      title,
      content,
      updatedAt: new Date().toISOString(),
      spellAllowlist: [],
  };
  store.state.drafts.unshift(newDraft);
  await activateDraft(id);
  await props.env.saveDrafts(store.state.drafts);
}

async function addSpellAllowlistWord(word: string) {
    const trimmed = word.trim();
    if (!trimmed) return false;

    if (!store.activeDraft() && pendingDraft.value) {
        const title = extractDocumentTitle(activeContent.value) || pendingDraft.value.title;
        await createDraft(title, activeContent.value);
    }

    return store.addSpellAllowlistWord(trimmed);
}

async function removeSpellAllowlistWord(word: string) {
    return store.removeSpellAllowlistWord(word);
}

function showPendingPlaceholder(title: string, content: string) {
    // Runtime-only; intentionally do not touch the stored activeDraftId so
    // that a returning visitor still lands on their last saved draft if they
    // reload without the ?content= param.
    flushDrafts();
    store.state.activeDraftId = null;
    pendingDraft.value = { title, content };
    activeContent.value = content;
}

async function activateDraft(id: string) {
    const draft = store.state.drafts.find(d => d.id === id);
    if (draft) {
        pendingDraft.value = null;
        store.state.activeDraftId = id;
        activeContent.value = draft.content;
        await props.env.saveActiveDraftId(id);
    }
}

function flushDrafts() {
    if (persistTimer) {
        clearTimeout(persistTimer);
        persistTimer = null;
    }
    props.env.saveDrafts(store.state.drafts);
}

function dismissWelcome() {
  showWelcome.value = false;
  localStorage.setItem(welcomeStorageKey, "true");
}

function handleNewPlay() {
  if (activeSavedDraft.value) {
    showNewPlayConfirm.value = true;
    return;
  }
  applyNewPlay();
}

function applyNewPlay() {
  showNewPlayConfirm.value = false;
  showPendingPlaceholder("Untitled Play", newPlayTemplate);
  toastManager.value?.addToast("Started a new play", "success");
}

async function handleImport() {
    const imported = await props.env.importLocalFile();
    if (imported) {
        const title = extractDocumentTitle(imported.content) || imported.name.replace(/\.ds$/, "");
        await createDraft(title, imported.content);
        toastManager.value?.addToast(`Imported "${title}"`, "success");
        showDrafts.value = false;
    }
}

async function handleCopy() {
    await navigator.clipboard.writeText(activeContent.value);
    toastManager.value?.addToast("Copied to clipboard", "success");
}

async function handleSave() {
    const title = extractDocumentTitle(activeContent.value) || "untitled";
    const filename = `${title.replace(/[^a-z0-9]+/gi, "-").toLowerCase()}.ds`;
    await props.env.saveFile(filename, activeContent.value, [
        { displayName: "Downstage Files (*.ds)", pattern: "*.ds" }
    ]);
}

function handleExport() {
    if (isV1Document.value) {
        toastManager.value?.addToast("Upgrade this V1 document to V2 before exporting PDF", "error", 5000);
        return;
    }

    showExportDialog.value = true;
}

async function handleExportConfirmed(opts: ExportPdfOptions) {
    showExportDialog.value = false;
    exportPageSize.value = opts.pageSize;
    try {
        localStorage.setItem(pageSizeStorageKey, opts.pageSize);
    } catch {
        // ignore storage errors
    }

    // Only persist the layout on condensed exports. Manuscript always
    // comes through as layout=single and would clobber a previously chosen
    // 2up/booklet preference otherwise.
    if (opts.style === "condensed") {
        exportLayout.value = opts.layout;
        try {
            localStorage.setItem(layoutStorageKey, opts.layout);
        } catch {
            // ignore storage errors
        }
    }

    // Gutter only applies to booklet exports. Persisting it on single/2up
    // exports would overwrite the user's last booklet preference (and
    // could store a value they never intended for a booklet).
    if (opts.layout === "booklet") {
        exportGutter.value = opts.bookletGutter;
        try {
            localStorage.setItem(gutterStorageKey, opts.bookletGutter);
        } catch {
            // ignore storage errors
        }
    }

    const title = extractDocumentTitle(activeContent.value) || "untitled";
    const styleSlug = opts.style === "condensed" ? "acting-edition" : "manuscript";
    const layoutSuffix = opts.layout === "single" ? "" : `-${opts.layout}`;
    const filename = `${title.replace(/[^a-z0-9]+/gi, "-").toLowerCase()}-${styleSlug}${layoutSuffix}.pdf`;

    const pdfBytes = await props.env.renderPDF(activeContent.value, opts);
    // An empty Uint8Array means the WASM side rejected the request (bad
    // config, imposition failure, etc.). Saving it would produce a broken
    // file; surface the failure as a toast instead.
    if (!pdfBytes || pdfBytes.byteLength === 0) {
        toastManager.value?.addToast(
            "PDF export failed. Check the export settings and try again.",
            "error",
            5000,
        );
        return;
    }
    await props.env.saveFile(filename, pdfBytes, [
        { displayName: "PDF Files (*.pdf)", pattern: "*.pdf" }
    ]);
}

async function confirmDeleteDraft() {
    if (!draftToDelete.value) return;
    const id = draftToDelete.value.id;
    const title = draftToDelete.value.title;
    store.state.drafts = store.state.drafts.filter(d => d.id !== id);
    if (store.state.activeDraftId === id) {
        if (store.state.drafts.length > 0) {
            await activateDraft(store.state.drafts[0].id);
        } else {
            applyNewPlay();
        }
    }
    await props.env.saveDrafts(store.state.drafts);
    draftToDelete.value = null;
    toastManager.value?.addToast(`Deleted "${title}"`, "info");
}

function schedulePersist() {
    if (persistTimer) clearTimeout(persistTimer);
    persistTimer = window.setTimeout(() => {
        persistTimer = null;
        props.env.saveDrafts(store.state.drafts);
    }, 250);
}

watch(activeContent, (newContent) => {
    // Promote an unsaved placeholder (New Play, initial Example, "Try it"
    // snippet) into a real draft the first time the user edits it.
    if (pendingDraft.value && newContent !== pendingDraft.value.content) {
        const placeholder = pendingDraft.value;
        pendingDraft.value = null;
        const id = `draft-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
        const newDraft: SavedDraft = {
            id,
            title: extractDocumentTitle(newContent) || placeholder.title,
            content: newContent,
            updatedAt: new Date().toISOString(),
            spellAllowlist: [],
        };
        store.state.drafts.unshift(newDraft);
        store.state.activeDraftId = id;
        props.env.saveActiveDraftId(id);
        schedulePersist();
        return;
    }

    const draft = store.state.drafts.find(d => d.id === store.state.activeDraftId);
    if (draft) {
        draft.content = newContent;
        draft.title = extractDocumentTitle(newContent) || "Untitled Play";
        draft.updatedAt = new Date().toISOString();
        schedulePersist();
    }
});

</script>

<template>
  <div class="h-screen flex flex-col bg-page-glow dark:bg-page-glow text-text-main overflow-hidden font-sans transition-colors duration-300">
    <header v-if="isLoaded" class="flex items-center justify-between gap-5 px-5 py-3.5 bg-[var(--color-page-surface)] border-b border-border shadow-stage z-10">
      <div class="flex items-center gap-4">
        <h1 class="font-serif text-xl font-bold text-text-main">
          <a href="/" class="hover:text-brass-500 transition-colors">Downstage</a>
        </h1>
        <button
          class="flex items-center gap-1.5 text-xs px-3 py-1 rounded-full border border-black/10 dark:border-white/10 bg-black/5 dark:bg-white/5 text-text-muted hover:bg-black/10 dark:hover:bg-white/10 hover:text-text-main transition-colors font-medium"
          @click="env.openURL('https://www.getdownstage.com/syntax/')"
        >
          <ExternalLink class="w-3 h-3" /> <span class="hidden md:inline">Syntax Guide</span>
        </button>
        <span class="hidden lg:inline text-[11px] uppercase tracking-[0.15em] text-text-muted font-bold opacity-60">{{ store.state.appVersion }}</span>
      </div>

      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <ToolbarButton @click="handleNewPlay" title="Start a new play"><template #icon><Plus class="w-4 h-4" /></template>New Play</ToolbarButton>
          <ToolbarButton @click="showDrafts = true" title="Open one of your saved drafts"><template #icon><FolderOpen class="w-4 h-4" /></template>My Drafts</ToolbarButton>
          <ToolbarButton @click="handleCopy" title="Copy Content"><template #icon><Copy class="w-4 h-4" /></template>Copy</ToolbarButton>
          <ToolbarButton @click="handleSave" title="Save .ds File"><template #icon><Download class="w-4 h-4" /></template>Save .ds</ToolbarButton>
          <ToolbarButton
            @click="handleExport"
            :disabled="isV1Document"
            :title="isV1Document ? 'Upgrade this V1 document before exporting PDF' : 'Export to PDF'"
          ><template #icon><FileOutput class="w-4 h-4" /></template>Export PDF</ToolbarButton>
        </div>
      </div>
    </header>

    <div v-if="!isLoaded" class="flex-1 flex items-center justify-center text-text-muted italic bg-[var(--color-page-bg)]">
      Loading Downstage editor...
    </div>

    <main v-else class="flex-1 overflow-hidden flex flex-col">
      <Editor
        :env="env"
        :document-key="store.state.activeDraftId"
        v-model:content="activeContent"
        v-model:style="pageStyle"
        :get-spell-allowlist="() => activeSavedDraft?.spellAllowlist || []"
        :add-spell-allowlist-word="addSpellAllowlistWord"
        :remove-spell-allowlist-word="removeSpellAllowlistWord"
        @migration-state-change="isV1Document = $event"
      />
    </main>

    <BaseModal
      :open="showDrafts"
      title="My Drafts"
      kicker="Drafts saved in this browser"
      @close="showDrafts = false"
    >
      <div class="flex flex-col gap-6">
        <div>
          <button 
            @click="handleImport"
            class="w-full py-3 rounded-lg border-2 border-dashed border-black/10 dark:border-white/10 hover:border-brass-500/50 hover:bg-black/5 dark:hover:bg-white/5 transition-all text-center group"
          >
            <span class="flex items-center justify-center gap-2 text-brass-500 font-bold group-hover:text-brass-400 text-lg leading-tight">
                <Upload class="w-5 h-5" /> Import .ds File...
            </span>
            <span class="text-xs text-text-muted font-medium">Add a local file to your browser library</span>
          </button>
        </div>

        <div v-if="store.state.drafts.length > 0" class="flex flex-col border border-border rounded-lg overflow-hidden bg-black/5 dark:bg-white/5 divide-y divide-border">
            <div 
                v-for="(draft, index) in store.state.drafts" 
                :key="draft.id" 
                class="flex items-center gap-3 p-3 transition-colors cursor-pointer group hover:bg-black/5 dark:hover:bg-white/5"
                :class="[
                    store.state.activeDraftId === draft.id ? 'bg-brass-500/5' : '',
                    index % 2 === 0 ? 'bg-black/[0.02] dark:bg-white/[0.02]' : ''
                ]"
            >
            <div class="flex-1 min-w-0" @click="activateDraft(draft.id); showDrafts = false">
                <div class="text-text-main font-bold truncate text-sm flex items-center gap-2">
                    <FileText class="w-3.5 h-3.5 text-text-muted opacity-50" />
                    {{ draft.title }}
                    <span v-if="store.state.activeDraftId === draft.id" class="text-[9px] uppercase tracking-widest bg-brass-500/20 text-brass-600 dark:text-brass-400 px-1.5 py-0.5 rounded font-bold">Active</span>
                </div>
                <div class="text-[10px] text-text-muted uppercase tracking-[0.1em] mt-0.5 ml-5">{{ new Date(draft.updatedAt).toLocaleString() }}</div>
            </div>
            <button 
                @click.stop="draftToDelete = draft"
                class="p-2 text-text-muted hover:text-red-500 opacity-0 group-hover:opacity-100 transition-all rounded-md hover:bg-red-500/10"
                title="Delete Draft"
            >
                <Trash2 class="w-4 h-4" />
            </button>
            </div>
        </div>
      </div>
    </BaseModal>

    <DeleteConfirmationModal
        v-if="draftToDelete"
        :open="!!draftToDelete"
        :item-name="draftToDelete.title"
        @close="draftToDelete = null"
        @confirm="confirmDeleteDraft"
    />

    <BaseModal
        :open="showNewPlayConfirm"
        title="Start a new play?"
        kicker="New Play"
        @close="showNewPlayConfirm = false"
    >
        <div class="flex flex-col items-center text-center py-2">
          <div class="w-16 h-16 rounded-full bg-brass-500/10 flex items-center justify-center mb-4">
            <AlertTriangle class="w-8 h-8 text-brass-500" />
          </div>
          <p class="text-text-main font-medium mb-2">This will replace what's in the editor.</p>
          <p class="text-sm text-text-muted mb-8 leading-relaxed">
            <template v-if="activeSavedDraft">
              Your current draft <strong class="text-text-main">"{{ activeSavedDraft.title }}"</strong> is still saved.
            </template>
            <template v-else>
              Your current draft is still saved.
            </template>
            You can reopen it any time from <strong class="text-text-main">My Drafts</strong>.
          </p>
          <div class="flex gap-3 w-full">
            <button
              @click="showNewPlayConfirm = false"
              class="flex-1 px-4 py-2.5 rounded-lg border border-border text-sm font-bold hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
            >
              Cancel
            </button>
            <button
              @click="applyNewPlay"
              class="flex-1 px-4 py-2.5 rounded-lg bg-brass-500 text-black text-sm font-bold hover:bg-brass-400 transition-colors shadow-lg"
            >
              Start New Play
            </button>
          </div>
        </div>
    </BaseModal>

    <WelcomeModal
        :open="showWelcome"
        @close="dismissWelcome"
    />

    <ExportPdfModal
        :open="showExportDialog"
        :initial-options="{
          pageSize: exportPageSize,
          style: pageStyle === 'condensed' ? 'condensed' : 'standard',
          layout: exportLayout,
          bookletGutter: exportGutter,
        }"
        @close="showExportDialog = false"
        @confirm="handleExportConfirmed"
    />

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
