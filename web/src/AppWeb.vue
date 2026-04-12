<script setup lang="ts">
import { provide, onMounted, ref, watch, onUnmounted } from 'vue';
import { 
    Plus, FolderOpen, FileText, Download, Copy, ExternalLink, Trash2, X, FileOutput, Upload
} from 'lucide-vue-next';
import { Store } from './core/store';
import type { EditorEnv, SavedDraft } from './core/types';
import ToolbarButton from './components/shared/ToolbarButton.vue';
import BaseModal from './components/shared/BaseModal.vue';
import DeleteConfirmationModal from './components/shared/DeleteConfirmationModal.vue';
import ToastManager from './components/shared/ToastManager.vue';
import Editor from './components/shared/Editor.vue';
import WelcomeModal from './components/shared/WelcomeModal.vue';

const props = defineProps<{
  env: EditorEnv;
}>();

const store = new Store(props.env);
provide('store', store);

const quickReferenceStorageKey = "downstage-quick-reference-hidden";
const welcomeStorageKey = "downstage-editor-welcome-dismissed";

const isLoaded = ref(false);
const showDrafts = ref(false);
const showQuickReference = ref(false);
const showWelcome = ref(false);
const activeContent = ref("");
const pageStyle = ref("standard");

// Toast and Delete state
const toastManager = ref<InstanceType<typeof ToastManager> | null>(null);
const draftToDelete = ref<SavedDraft | null>(null);

let persistTimer: number | null = null;

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

onMounted(async () => {
  await store.init();
  
  // Restore URL content if present (handling UTF-8 correctly)
  const params = new URLSearchParams(window.location.search);
  const urlContent = params.get("content");
  if (urlContent) {
    try {
        const decoded = decodeURIComponent(escape(atob(urlContent)));
        await createDraft("Imported Play", decoded);
        toastManager.value?.addToast("Imported content from URL", "success");
        // Clear param
        window.history.replaceState({}, '', window.location.pathname);
    } catch (e) {
        console.error("Failed to decode URL content", e);
    }
  } else if (store.state.activeDraftId) {
    const draft = store.state.drafts.find(d => d.id === store.state.activeDraftId);
    if (draft) {
        activeContent.value = draft.content;
    } else if (store.state.drafts.length > 0) {
        // Fallback to first draft if active draft is missing
        await activateDraft(store.state.drafts[0].id);
    }
  }

  if (store.state.drafts.length === 0 && !activeContent.value) {
    await createDraft("The Example Play", exampleContent);
  }

  showQuickReference.value = localStorage.getItem(quickReferenceStorageKey) === "false";
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
  const id = `draft-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const newDraft = {
      id,
      title,
      content,
      updatedAt: new Date().toISOString()
  };
  store.state.drafts.unshift(newDraft);
  await activateDraft(id);
  await props.env.saveDrafts(store.state.drafts);
}

async function activateDraft(id: string) {
    const draft = store.state.drafts.find(d => d.id === id);
    if (draft) {
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

function openQuickReferenceFromWelcome() {
  showQuickReference.value = true;
  dismissWelcome();
}

function handleNewPlay() {
  const template = `Title: Untitled Play\nAuthor: Your Name\nDate: ${new Date().getFullYear()}\nDraft: First\n\n# Dramatis Personae\n\nPROTAGONIST — Add your cast here\n\n# Untitled Play\n\n## ACT I\n\n### SCENE 1\n\n> Describe the setting here.\n\nPROTAGONIST\nWrite your opening lines here.\n`;
  createDraft("Untitled Play", template);
  toastManager.value?.addToast("Created new play", "success");
}

function handleLoadExample() {
    createDraft("The Example Play", exampleContent);
    toastManager.value?.addToast("Loaded example play", "success");
}

async function handleImport() {
    const imported = await props.env.importLocalFile();
    if (imported) {
        const title = imported.content.match(/^Title:\s*(.+)$/m)?.[1]?.trim() || imported.name.replace(/\.ds$/, "");
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
    const title = activeContent.value.match(/^Title:\s*(.+)$/m)?.[1]?.trim() || "untitled";
    const filename = `${title.replace(/[^a-z0-9]+/gi, "-").toLowerCase()}.ds`;
    await props.env.saveFile(filename, activeContent.value, [
        { displayName: "Downstage Files (*.ds)", pattern: "*.ds" }
    ]);
}

async function handleExport() {
    const title = activeContent.value.match(/^Title:\s*(.+)$/m)?.[1]?.trim() || "untitled";
    const styleSlug = pageStyle.value === "condensed" ? "acting-edition" : "manuscript";
    const filename = `${title.replace(/[^a-z0-9]+/gi, "-").toLowerCase()}-${styleSlug}.pdf`;
    
    const pdfBytes = await props.env.renderPDF(activeContent.value, pageStyle.value);
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
            handleNewPlay();
        }
    }
    await props.env.saveDrafts(store.state.drafts);
    draftToDelete.value = null;
    toastManager.value?.addToast(`Deleted "${title}"`, "info");
}

watch(activeContent, (newContent) => {
    const draft = store.state.drafts.find(d => d.id === store.state.activeDraftId);
    if (draft) {
        draft.content = newContent;
        draft.title = newContent.match(/^Title:\s*(.+)$/m)?.[1]?.trim() || "Untitled Play";
        draft.updatedAt = new Date().toISOString();
        
        if (persistTimer) clearTimeout(persistTimer);
        persistTimer = window.setTimeout(() => {
            persistTimer = null;
            props.env.saveDrafts(store.state.drafts);
        }, 250);
    }
});

watch(showQuickReference, (val) => {
    localStorage.setItem(quickReferenceStorageKey, String(!val));
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
          @click="env.openURL('https://www.getdownstage.com/docs/')"
        >
          <ExternalLink class="w-3 h-3" /> <span class="hidden md:inline">Docs</span>
        </button>
        <span class="hidden lg:inline text-[11px] uppercase tracking-[0.15em] text-text-muted font-bold opacity-60">{{ store.state.appVersion }}</span>
      </div>

      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <ToolbarButton @click="handleNewPlay" title="New Play"><template #icon><Plus class="w-4 h-4" /></template>New Play</ToolbarButton>
          <ToolbarButton @click="showDrafts = true" title="Open Manuscript"><template #icon><FolderOpen class="w-4 h-4" /></template>Open</ToolbarButton>
          <ToolbarButton @click="handleLoadExample" title="Load Example Play"><template #icon><FileText class="w-4 h-4" /></template>Example</ToolbarButton>
          <ToolbarButton @click="handleCopy" title="Copy Content"><template #icon><Copy class="w-4 h-4" /></template>Copy</ToolbarButton>
          <ToolbarButton @click="handleSave" title="Save .ds File"><template #icon><Download class="w-4 h-4" /></template>Save .ds</ToolbarButton>
          <ToolbarButton @click="handleExport" title="Export to PDF"><template #icon><FileOutput class="w-4 h-4" /></template>Export PDF</ToolbarButton>
        </div>
      </div>
    </header>

    <div v-if="!isLoaded" class="flex-1 flex items-center justify-center text-text-muted italic bg-[var(--color-page-bg)]">
      Loading Downstage editor...
    </div>

    <main v-else class="flex-1 overflow-hidden flex flex-col">
      <section
        v-if="showQuickReference"
        class="border-b border-border bg-[var(--color-page-surface)] px-4 py-3 shadow-sm"
      >
        <div class="mx-auto flex max-w-7xl flex-col gap-3">
          <div class="flex items-start justify-between gap-4">
            <div>
              <h2 class="text-[10px] font-bold uppercase tracking-[0.2em] text-brass-500">Quick Reference</h2>
              <p class="mt-1 text-sm text-text-muted">
                The basics only. Keep writing, and open the
                <button
                  class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
                  @click="env.openURL('https://www.getdownstage.com/docs/')"
                >
                  docs
                </button>
                for the full spec.
              </p>
            </div>
            <button
              class="rounded-full p-2 text-text-muted transition-colors hover:bg-black/5 hover:text-text-main dark:hover:bg-white/5"
              @click="showQuickReference = false"
              aria-label="Close quick reference"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <dl class="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
            <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
              <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Title Block</dt>
              <dd>
                <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>Title: My Play
Author: Your Name
Draft: First</code></pre>
              </dd>
            </div>

            <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
              <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Cue + Dialogue</dt>
              <dd>
                <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>ALICE
I know this looks reckless.</code></pre>
              </dd>
            </div>

            <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
              <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Stage Direction</dt>
              <dd>
                <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>&gt; The lights cut to black.</code></pre>
              </dd>
            </div>

            <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
              <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Structure</dt>
              <dd>
                <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>## ACT I
### SCENE 1</code></pre>
              </dd>
            </div>

            <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
              <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Emphasis</dt>
              <dd>
                <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>**bold**
*italic*
_underline_</code></pre>
              </dd>
            </div>
          </dl>
        </div>
      </section>

      <Editor 
        :env="env"
        v-model:content="activeContent"
        v-model:style="pageStyle"
        @toggle-help="showQuickReference = !showQuickReference"
      />
    </main>

    <BaseModal 
      :open="showDrafts" 
      title="Open Manuscript" 
      kicker="Manuscript Library"
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

    <WelcomeModal
        :open="showWelcome"
        @close="dismissWelcome"
        @open-quick-reference="openQuickReferenceFromWelcome"
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
