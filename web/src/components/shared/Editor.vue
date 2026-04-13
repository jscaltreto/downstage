<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch, inject, nextTick } from 'vue';
import { 
    Bold, Italic, Underline, MessageSquare, ChevronRight, 
    GalleryVerticalEnd, GalleryVertical, FilePlus2, Eye, EyeOff, HelpCircle, X, Music,
    Sun, Moon, ScrollText, BookOpenText, AlertTriangle, RefreshCw
} from 'lucide-vue-next';
import { Engine } from '../../core/engine';
import type { Store } from '../../core/store';
import type { EditorEnv } from '../../core/types';
import PreviewFrame from './PreviewFrame.vue';
import ToolbarButton from './ToolbarButton.vue';
import BaseModal from './BaseModal.vue';

const props = defineProps<{
  env: EditorEnv;
  content: string;
  style: string;
}>();

const emit = defineEmits<{
  (e: 'update:content', value: string): void;
  (e: 'update:style', value: string): void;
  (e: 'toggle-help'): void;
  (e: 'migration-state-change', value: boolean): void;
}>();

const store = inject<Store>('store')!;
const editorContainer = ref<HTMLElement | null>(null);
const previewFrameComponent = ref<InstanceType<typeof PreviewFrame> | null>(null);
const renderedHtml = ref("");
let engine: Engine | null = null;

const previewVisible = ref(localStorage.getItem("downstage-editor-preview-hidden") !== "true");
const v1DocumentDetected = ref(false);
const showV1Modal = ref(false);
const v1DismissedForDraftId = ref<string | null>(null);
const isUpgradingV1 = ref(false);

let lastRenderRequestId = 0;
let renderTimer: number | null = null;
const v1DiagnosticCode = "v1-document";

function setV1DocumentDetected(detected: boolean) {
    v1DocumentDetected.value = detected;
    emit('migration-state-change', detected);
}

async function scheduleRender(content: string, style: string) {
    if (!previewVisible.value) return;

    if (renderTimer) window.clearTimeout(renderTimer);
    
    renderTimer = window.setTimeout(async () => {
        renderTimer = null;
        const requestId = ++lastRenderRequestId;
        const { diagnostics } = await props.env.diagnostics(content);
        if (requestId !== lastRenderRequestId) return;

        const hasV1Diagnostic = diagnostics.some((diagnostic) => diagnostic.code === v1DiagnosticCode);
        setV1DocumentDetected(hasV1Diagnostic);

        if (hasV1Diagnostic) {
            renderedHtml.value = "";
            if (v1DismissedForDraftId.value !== store.state.activeDraftId) {
                showV1Modal.value = true;
            }
            return;
        }

        showV1Modal.value = false;
        v1DismissedForDraftId.value = null;

        const html = await props.env.renderHTML(content, style);
        if (requestId === lastRenderRequestId) {
            renderedHtml.value = html;
        }
    }, 300);
}

async function upgradeV1Document() {
    if (isUpgradingV1.value) return;

    const currentContent = engine?.getContent() || props.content;
    isUpgradingV1.value = true;
    try {
        const result = await props.env.upgradeV1(currentContent);
        if (!result.changed) {
            showV1Modal.value = false;
            v1DismissedForDraftId.value = store.state.activeDraftId;
            return;
        }

        v1DismissedForDraftId.value = null;
        showV1Modal.value = false;
        // Parent updates props.content; the watch below syncs the engine and
        // schedules a re-render, so no direct engine/render calls here.
        emit('update:content', result.source);
    } finally {
        isUpgradingV1.value = false;
    }
}

function dismissV1Modal() {
    showV1Modal.value = false;
    v1DismissedForDraftId.value = store.state.activeDraftId;
}

onMounted(async () => {
  await nextTick();

  const iframeEl = previewFrameComponent.value?.iframeEl;
  if (editorContainer.value && iframeEl) {
    engine = new Engine(
      editorContainer.value,
      props.env,
      async (content) => {
        emit('update:content', content);
        scheduleRender(content, props.style);
      },
      iframeEl
    );
    engine.init(props.content, store.state.isDark);
    scheduleRender(props.content, props.style);
  }
});

onUnmounted(() => {
  engine?.destroy();
  if (renderTimer) window.clearTimeout(renderTimer);
});

watch(() => store.state.isDark, (isDark) => {
  engine?.setTheme(isDark);
});

watch(() => props.content, (newContent) => {
  if (engine && engine.getContent() !== newContent) {
    engine.setContent(newContent);
  }
  scheduleRender(newContent, props.style);
});

watch(() => store.state.activeDraftId, () => {
  showV1Modal.value = false;
});

watch(() => props.style, (newStyle) => {
    if (engine) {
        scheduleRender(engine.getContent(), newStyle);
    }
});

watch(previewVisible, (visible) => {
    localStorage.setItem("downstage-editor-preview-hidden", String(!visible));
    if (visible && engine) {
        scheduleRender(engine.getContent(), props.style);
    }
});

function handleFormat(action: string) {
    if (!engine) return;
    engine.applyFormat(action);
}

function toggleStyle() {
    const nextStyle = props.style === "condensed" ? "standard" : "condensed";
    emit('update:style', nextStyle);
}
</script>

<template>
  <div class="flex-1 flex flex-col overflow-hidden bg-[var(--color-page-bg)]">
    <div class="px-4 py-2 border-b border-border bg-[var(--color-toolbar-bg)] flex items-center justify-between gap-2 shadow-sm z-10">
        <div class="flex items-center gap-1.5 overflow-x-auto no-scrollbar">
            <ToolbarButton @click="handleFormat('bold')" title="Bold"><template #icon><Bold class="w-4 h-4" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('italic')" title="Italic"><template #icon><Italic class="w-4 h-4" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('underline')" title="Underline"><template #icon><Underline class="w-4 h-4" /></template></ToolbarButton>
            
            <div class="w-px h-4 bg-black/10 dark:bg-white/10 mx-1"></div>
            
            <ToolbarButton @click="handleFormat('cue')" title="Dialogue"><template #icon><MessageSquare class="w-4 h-4" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('direction')" title="Stage Direction"><template #icon><ChevronRight class="w-4 h-4" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('act')" title="Act Heading"><template #icon><GalleryVerticalEnd class="w-4 h-4" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('scene')" title="Scene Heading"><template #icon><GalleryVertical class="w-4 h-4 opacity-70" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('song')" title="Song Block"><template #icon><Music class="w-4 h-4" /></template></ToolbarButton>
            <ToolbarButton @click="handleFormat('page-break')" title="Page Break"><template #icon><FilePlus2 class="w-4 h-4" /></template></ToolbarButton>
        </div>

        <div class="flex items-center gap-1.5 border-l border-black/10 dark:border-white/10 pl-2">
            <ToolbarButton @click="emit('toggle-help')" title="Help" class="w-8 h-8 !p-0 rounded-full font-bold" transparent>
                <template #icon><HelpCircle class="w-5 h-5" /></template>
            </ToolbarButton>

            <ToolbarButton 
                @click="store.toggleTheme()" 
                :title="store.state.isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'"
                transparent
                class="w-8 h-8 !p-0 rounded-full"
            >
                <template #icon>
                    <Sun v-if="store.state.isDark" class="w-4 h-4" />
                    <Moon v-else class="w-4 h-4" />
                </template>
            </ToolbarButton>

            <ToolbarButton 
                @click="toggleStyle()" 
                :title="style === 'condensed' ? 'Switch to Manuscript' : 'Switch to Acting Edition'"
                transparent
                class="w-8 h-8 !p-0 rounded-full"
            >
                <template #icon>
                    <BookOpenText v-if="style === 'condensed'" class="w-4 h-4" />
                    <ScrollText v-else class="w-4 h-4" />
                </template>
            </ToolbarButton>

            <ToolbarButton 
                @click="previewVisible = !previewVisible" 
                :title="previewVisible ? 'Hide Preview' : 'Show Preview'"
                class="w-8 h-8 !p-0 rounded-full"
                :active="previewVisible"
                transparent
            >
                <template #icon>
                    <EyeOff v-if="previewVisible" class="w-5 h-5" />
                    <Eye v-else class="w-5 h-5" />
                </template>
            </ToolbarButton>
        </div>
    </div>

    <div class="flex-1 flex overflow-hidden relative">
        <div ref="editorContainer" class="flex-1 h-full overflow-hidden border-r border-border bg-[var(--color-page-bg)]"></div>
        
        <div v-show="previewVisible" class="flex-1 h-full bg-[var(--color-page-surface)] flex flex-col min-w-[300px]">
            <div class="px-4 py-2 border-b border-border bg-[var(--color-toolbar-bg)] flex justify-between items-center shadow-sm">
                <h2 class="text-[10px] uppercase tracking-[0.2em] font-bold text-accent">Live Preview</h2>
                <button @click="previewVisible = false" class="text-text-muted hover:text-text-main transition-colors" title="Hide Preview">
                    <X class="w-4 h-4" />
                </button>
            </div>
            <div class="flex-1 bg-white relative font-sans">
                <div
                    v-if="v1DocumentDetected"
                    class="absolute inset-0 flex flex-col items-center justify-center gap-5 bg-[linear-gradient(180deg,#f8efe2_0%,#fff8f1_100%)] px-8 text-center text-ember-950"
                >
                    <div class="flex h-14 w-14 items-center justify-center rounded-full bg-amber-500/15 text-amber-700">
                        <AlertTriangle class="h-7 w-7" />
                    </div>
                    <div class="max-w-md space-y-2">
                        <p class="text-[11px] font-bold uppercase tracking-[0.2em] text-amber-700">V1 Document</p>
                        <h3 class="font-serif text-2xl font-bold leading-tight">Preview is disabled until this script is upgraded.</h3>
                        <p class="text-sm leading-relaxed text-ember-900/75">
                            This document matches the old Downstage V1 format. Rendering is unreliable in V2, so update the script before using preview or export.
                        </p>
                    </div>
                    <div class="flex flex-col gap-3 sm:flex-row">
                        <button
                            class="inline-flex items-center justify-center gap-2 rounded-lg bg-brass-500 px-4 py-2.5 text-sm font-bold text-ember-950 transition-colors hover:bg-brass-400 disabled:opacity-60"
                            :disabled="isUpgradingV1"
                            @click="upgradeV1Document"
                        >
                            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': isUpgradingV1 }" />
                            {{ isUpgradingV1 ? 'Updating…' : 'Update Script to V2' }}
                        </button>
                        <button
                            class="rounded-lg border border-ember-950/10 px-4 py-2.5 text-sm font-bold text-ember-950/80 transition-colors hover:bg-ember-950/5"
                            @click="showV1Modal = true"
                        >
                            Why?
                        </button>
                    </div>
                </div>
                <PreviewFrame v-else ref="previewFrameComponent" :html="renderedHtml" />
            </div>
        </div>

        <button 
            v-if="!previewVisible"
            @click="previewVisible = true"
            class="absolute right-6 bottom-6 w-12 h-12 rounded-full bg-brass-500 text-ember-950 shadow-2xl flex items-center justify-center hover:bg-brass-400 transition-all transform hover:scale-110 z-20"
            title="Show Preview"
        >
            <Eye class="w-6 h-6 text-ember-950" />
        </button>
    </div>
  </div>

  <BaseModal
    :open="showV1Modal"
    kicker="Migration Required"
    title="This looks like a V1 Downstage document"
    message="V1 scripts do not render correctly in the current editor. Update the document to V2 to keep using preview and export safely."
    @close="dismissV1Modal"
  >
    <div class="flex flex-col gap-5 py-1">
        <div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4">
            <div class="flex items-start gap-3">
                <div class="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-amber-500/15">
                    <AlertTriangle class="h-5 w-5 text-amber-500" />
                </div>
                <div class="space-y-2">
                    <p class="text-sm font-bold text-text-main">Preview and PDF export are blocked for V1 documents.</p>
                    <p class="text-sm leading-relaxed text-text-muted">
                        The editor can update the old title-page and Dramatis Personae structure for you. You can keep editing raw text if you want, but preview will stay disabled until the file is upgraded.
                    </p>
                </div>
            </div>
        </div>

        <div class="flex flex-col gap-3 pt-1 sm:flex-row">
            <button
                class="flex-1 rounded-lg border border-border px-4 py-2.5 text-sm font-bold text-text-main transition-colors hover:bg-black/5 dark:hover:bg-white/5"
                @click="dismissV1Modal"
            >
                Keep Raw Editing
            </button>
            <button
                class="flex flex-1 items-center justify-center gap-2 rounded-lg bg-brass-500 px-4 py-2.5 text-sm font-bold text-black transition-colors hover:bg-brass-400 disabled:opacity-60"
                :disabled="isUpgradingV1"
                @click="upgradeV1Document"
            >
                <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': isUpgradingV1 }" />
                {{ isUpgradingV1 ? 'Updating…' : 'Update Script to V2' }}
            </button>
        </div>
    </div>
  </BaseModal>
</template>

<style>
.cm-editor { height: 100%; outline: none !important; }
.cm-gutters {
    background-color: transparent !important;
    color: var(--color-text-muted) !important;
    border-right: 1px solid var(--border-color) !important;
}
.dark .cm-activeLine { background-color: rgba(255, 255, 255, 0.05) !important; }
.cm-activeLine { background-color: rgba(0, 0, 0, 0.03) !important; }

.dark .cm-tooltip.cm-tooltip-autocomplete > ul > li {
    color: var(--color-text-muted);
}
.dark .cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] {
    background: var(--color-brass-500);
    color: var(--color-ember-950);
}
.dark .cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail {
    color: var(--color-ember-900);
}

.cm-tooltip-lint .cm-diagnostic {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 0.4rem;
    padding: 0.5rem 0.65rem;
    max-width: 420px;
}
.cm-tooltip-lint .cm-diagnosticText {
    white-space: normal;
    line-height: 1.35;
}
.cm-tooltip-lint .cm-diagnosticAction {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0;
    padding: 0.25rem 0.55rem;
    border-radius: 4px;
    border: 1px solid var(--border-color);
    background: var(--color-toolbar-bg);
    color: var(--color-accent);
    font-size: 0.85em;
    font-weight: 500;
    cursor: pointer;
    align-self: flex-start;
    transition: background-color 0.15s, color 0.15s, border-color 0.15s;
}
.cm-tooltip-lint .cm-diagnosticAction:hover {
    background: var(--color-accent);
    color: var(--color-page-bg);
    border-color: var(--color-accent);
}
.cm-tooltip-lint .cm-diagnosticAction::before {
    content: "";
    display: inline-block;
    width: 1em;
    height: 1em;
    flex-shrink: 0;
    background-color: currentColor;
    -webkit-mask: var(--cm-lightbulb) no-repeat center / contain;
    mask: var(--cm-lightbulb) no-repeat center / contain;
    --cm-lightbulb: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='-2 -2 28 28' fill='none' stroke='black' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M15 14c.2-1 .7-1.7 1.5-2.5 1.1-1 2.5-2.2 2.5-4.5A6 6 0 0 0 7 7c0 2.3 1.4 3.5 2.5 4.5.8.8 1.3 1.5 1.5 2.5'/><path d='M9 18h6'/><path d='M10 22h4'/></svg>");
}

.no-scrollbar::-webkit-scrollbar { display: none; }
.no-scrollbar { -ms-overflow-style: none; scrollbar-width: none; }
</style>
