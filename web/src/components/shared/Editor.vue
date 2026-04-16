<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, inject, nextTick } from 'vue';
import {
    Bold, Italic, Underline, MessageSquare, ChevronRight,
    GalleryVerticalEnd, GalleryVertical, FilePlus2, Eye, EyeOff, HelpCircle, X, Music,
    Sun, Moon, ScrollText, BookOpenText, AlertTriangle, AlertCircle, Info, RefreshCw, SpellCheck, Trash2, Search, ListTree, BarChart3
} from 'lucide-vue-next';
import { Engine, type SearchMode, type SearchSummary } from '../../core/engine';
import type { SearchMatch, SearchOptions } from '../../core/search';
import { issuesStatus, summarizeIssues } from '../../core/issues';
import type { FilterSeverity } from '../../core/issues';
import type { Store } from '../../core/store';
import type { DocumentSymbol, EditorDiagnostic, EditorEnv, ManuscriptStats } from '../../core/types';
import PreviewFrame from './PreviewFrame.vue';
import ToolbarButton from './ToolbarButton.vue';
import BaseModal from './BaseModal.vue';
import WorkbenchDrawer, { type WorkbenchTab } from './WorkbenchDrawer.vue';
import IssuesTab from './IssuesTab.vue';
import FindReplaceTab from './FindReplaceTab.vue';
import OutlineTab from './OutlineTab.vue';
import StatsTab from './StatsTab.vue';
import HelpTab from './HelpTab.vue';

const props = defineProps<{
  env: EditorEnv;
  content: string;
  style: string;
  getSpellAllowlist: () => string[];
  addSpellAllowlistWord: (word: string) => Promise<boolean>;
  removeSpellAllowlistWord: (word: string) => Promise<boolean>;
}>();

const emit = defineEmits<{
  (e: 'update:content', value: string): void;
  (e: 'update:style', value: string): void;
  (e: 'migration-state-change', value: boolean): void;
}>();

const store = inject<Store>('store')!;
const editorContainer = ref<HTMLElement | null>(null);
const previewFrameComponent = ref<InstanceType<typeof PreviewFrame> | null>(null);
const renderedHtml = ref("");
let engine: Engine | null = null;

const previewVisible = ref(localStorage.getItem("downstage-editor-preview-hidden") !== "true");
const spellcheckEnabled = ref(localStorage.getItem("downstage-editor-spellcheck-disabled") !== "true");
const drawerOpen = ref(false);
const drawerTab = ref<WorkbenchTab>('issues');
const searchMatches = ref<SearchMatch[]>([]);
const searchIndex = ref(-1);
const searchError = ref<string | null>(null);
const searchInitialQuery = ref('');
const searchFocusReplace = ref(false);
const searchFocusNonce = ref(0);
const diagnostics = ref<EditorDiagnostic[]>([]);
const outlineSymbols = ref<DocumentSymbol[]>([]);
const manuscriptStats = ref<ManuscriptStats | null>(null);
const manuscriptStatsLoading = ref(false);
const isTyping = ref(false);
let typingTimer: number | null = null;
const outlineDebounceMs = 300;
const typingIndicatorMs = 1500;

function markTyping() {
    isTyping.value = true;
    if (typingTimer) window.clearTimeout(typingTimer);
    typingTimer = window.setTimeout(() => {
        isTyping.value = false;
        typingTimer = null;
    }, typingIndicatorMs);
}
const hiddenSeverities = ref<ReadonlySet<FilterSeverity>>(new Set());
const visibleDiagnostics = computed(() =>
  diagnostics.value.filter((d) => {
    if (d.severity === 'error') return !hiddenSeverities.value.has('error');
    if (d.severity === 'warning') return !hiddenSeverities.value.has('warning');
    return !hiddenSeverities.value.has('info');
  }),
);
const issuesSummary = computed(() => summarizeIssues(visibleDiagnostics.value));
const issuesStatusValue = computed(() => issuesStatus(issuesSummary.value));
const editorHideClasses = computed(() => {
  const classes: string[] = [];
  if (hiddenSeverities.value.has('error')) classes.push('cm-hide-error');
  if (hiddenSeverities.value.has('warning')) classes.push('cm-hide-warning');
  if (hiddenSeverities.value.has('info')) classes.push('cm-hide-info');
  return classes;
});
const showSpellcheckModal = ref(false);
const dictionaryWord = ref("");
const spellAllowlist = computed(() => props.getSpellAllowlist());
const v1DocumentDetected = ref(false);
const showV1Modal = ref(false);
const v1DismissedForDraftId = ref<string | null>(null);
const isUpgradingV1 = ref(false);

let lastRenderRequestId = 0;
let renderTimer: number | null = null;
let outlineRequestId = 0;
let outlineTimer: number | null = null;
let statsRequestId = 0;
let statsTimer: number | null = null;
const statsDebounceMs = 500;
const v1DiagnosticCode = "v1-document";

function scheduleOutlineRefresh(content: string) {
    if (outlineTimer) window.clearTimeout(outlineTimer);
    outlineTimer = window.setTimeout(async () => {
        outlineTimer = null;
        const requestId = ++outlineRequestId;
        try {
            const { symbols } = await props.env.documentSymbols(content);
            if (requestId !== outlineRequestId) return;
            outlineSymbols.value = symbols;
        } catch {
            if (requestId !== outlineRequestId) return;
            outlineSymbols.value = [];
        }
    }, outlineDebounceMs);
}

function scheduleStatsRefresh(content: string) {
    if (statsTimer) window.clearTimeout(statsTimer);
    manuscriptStatsLoading.value = true;
    statsTimer = window.setTimeout(async () => {
        statsTimer = null;
        const requestId = ++statsRequestId;
        try {
            const result = await props.env.stats(content);
            if (requestId !== statsRequestId) return;
            manuscriptStats.value = result;
        } catch {
            if (requestId !== statsRequestId) return;
            manuscriptStats.value = null;
        } finally {
            if (requestId === statsRequestId) {
                manuscriptStatsLoading.value = false;
            }
        }
    }, statsDebounceMs);
}

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
        emit('update:content', result.source);
    } finally {
        isUpgradingV1.value = false;
    }
}

function dismissV1Modal() {
    showV1Modal.value = false;
    v1DismissedForDraftId.value = store.state.activeDraftId;
}

function openSearch(mode: SearchMode) {
  const selection = engine?.getSelectionText() ?? '';
  if (selection) {
    searchInitialQuery.value = selection;
  }
  searchFocusReplace.value = mode === 'replace';
  drawerTab.value = 'find';
  drawerOpen.value = true;
  searchFocusNonce.value++;
}

function toggleSearch() {
  if (drawerOpen.value && drawerTab.value === 'find') {
    closeDrawer();
    return;
  }
  openSearch('find');
}

function applySearchSummary(summary: SearchSummary, matches: SearchMatch[]) {
  searchMatches.value = matches;
  searchIndex.value = summary.index;
  searchError.value = summary.error;
}

onMounted(async () => {
  await nextTick();

  const iframeEl = previewFrameComponent.value?.iframeEl;
  if (editorContainer.value && iframeEl) {
    engine = new Engine(
      editorContainer.value,
      props.env,
      async (content, info) => {
        emit('update:content', content);
        scheduleRender(content, props.style);
        scheduleOutlineRefresh(content);
        scheduleStatsRefresh(content);
        if (info.userInput) {
          markTyping();
        }
      },
      iframeEl,
      () => props.getSpellAllowlist(),
      (word) => props.addSpellAllowlistWord(word),
      (next) => { diagnostics.value = next; },
      openSearch,
      applySearchSummary,
    );
    engine.init(props.content, store.state.isDark, spellcheckEnabled.value);
    scheduleRender(props.content, props.style);
    scheduleOutlineRefresh(props.content);
    scheduleStatsRefresh(props.content);
  }
});

onUnmounted(() => {
  engine?.destroy();
  if (renderTimer) window.clearTimeout(renderTimer);
  if (outlineTimer) window.clearTimeout(outlineTimer);
  if (statsTimer) window.clearTimeout(statsTimer);
  if (typingTimer) window.clearTimeout(typingTimer);
});

watch(() => store.state.isDark, (isDark) => {
  engine?.setTheme(isDark);
});

watch(() => props.content, (newContent) => {
  if (engine && engine.getContent() !== newContent) {
    engine.setContent(newContent);
  }
  scheduleRender(newContent, props.style);
  scheduleOutlineRefresh(newContent);
  scheduleStatsRefresh(newContent);
});

watch(() => store.state.activeDraftId, () => {
  showV1Modal.value = false;
  diagnostics.value = [];
  manuscriptStats.value = null;
  manuscriptStatsLoading.value = true;
  engine?.refreshDiagnostics();
  engine?.clearSearch();
  searchMatches.value = [];
  searchIndex.value = -1;
  searchError.value = null;
  searchInitialQuery.value = '';
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

watch(spellcheckEnabled, (enabled) => {
    localStorage.setItem("downstage-editor-spellcheck-disabled", String(!enabled));
    engine?.setSpellcheckEnabled(enabled);
});

function handleFormat(action: string) {
    if (!engine) return;
    engine.applyFormat(action);
}

function toggleStyle() {
    const nextStyle = props.style === "condensed" ? "standard" : "condensed";
    emit('update:style', nextStyle);
}

async function addDictionaryWord() {
    const added = await props.addSpellAllowlistWord(dictionaryWord.value);
    if (added) {
        dictionaryWord.value = "";
        engine?.refreshDiagnostics();
    }
}

async function removeDictionaryWord(word: string) {
    const removed = await props.removeSpellAllowlistWord(word);
    if (removed) {
        engine?.refreshDiagnostics();
    }
}

function jumpToDiagnostic(d: EditorDiagnostic) {
    engine?.revealDiagnostic(d.from, d.to);
}

function jumpToSymbol(symbol: DocumentSymbol) {
    const start = symbol.selectionRange?.start ?? symbol.range.start;
    engine?.revealPosition(start.line, start.character);
}

function openWorkbenchTab(tab: WorkbenchTab) {
    if (drawerOpen.value && drawerTab.value === tab) {
        closeDrawer();
        return;
    }
    drawerTab.value = tab;
    drawerOpen.value = true;
}

function toggleOutline() {
    openWorkbenchTab('outline');
}

function toggleStats() {
    openWorkbenchTab('stats');
}

function toggleHelp() {
    openWorkbenchTab('help');
}

function openIssuesTab() {
    openWorkbenchTab('issues');
}

function closeDrawer() {
    drawerOpen.value = false;
    engine?.clearSearch();
    engine?.focus();
}

function onSearch(opts: SearchOptions) {
    if (!engine) return;
    engine.setSearch(opts);
}
function onFindNext() { engine?.findNext(); }
function onFindPrev() { engine?.findPrev(); }
function onReplaceOne(replacement: string) { engine?.replaceCurrent(replacement); }
function onReplaceAll(replacement: string) { engine?.replaceAll(replacement); }
function onJumpMatch(index: number) { engine?.selectMatch(index); }
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

            <div class="w-px h-4 bg-black/10 dark:bg-white/10 mx-1"></div>

            <ToolbarButton @click="showSpellcheckModal = true" title="Spell Check">
                <template #icon>
                    <span class="relative flex h-4 w-4 items-center justify-center">
                        <SpellCheck class="h-4 w-4" :class="{ 'opacity-45': !spellcheckEnabled }" />
                        <span
                            v-if="!spellcheckEnabled"
                            class="absolute h-[1.5px] w-5 rotate-[-35deg] rounded-full bg-current"
                            aria-hidden="true"
                        ></span>
                    </span>
                </template>
            </ToolbarButton>

            <ToolbarButton @click="toggleSearch" title="Find &amp; Replace (Ctrl/Cmd+F)">
                <template #icon><Search class="w-4 h-4" /></template>
            </ToolbarButton>

            <ToolbarButton
                @click="toggleOutline"
                :active="drawerOpen && drawerTab === 'outline'"
                title="Outline"
            >
                <template #icon><ListTree class="w-4 h-4" /></template>
            </ToolbarButton>

            <ToolbarButton
                @click="toggleStats"
                :active="drawerOpen && drawerTab === 'stats'"
                title="Stats"
            >
                <template #icon><BarChart3 class="w-4 h-4" /></template>
            </ToolbarButton>
        </div>

        <div class="flex items-center gap-1.5 border-l border-black/10 dark:border-white/10 pl-2">
            <ToolbarButton @click="toggleHelp" :active="drawerOpen && drawerTab === 'help'" title="Help" class="w-8 h-8 !p-0 rounded-full font-bold" transparent>
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
        <div class="flex-1 h-full flex flex-col border-r border-border bg-[var(--color-page-bg)]">
            <div class="flex-1 relative overflow-hidden">
                <div ref="editorContainer" :class="['absolute inset-0 overflow-hidden', ...editorHideClasses]"></div>

                <div
                    v-if="!drawerOpen"
                    class="absolute right-6 bottom-6 z-20 flex flex-col items-end gap-3"
                >
                    <Transition name="fab-fade">
                        <button
                            v-if="issuesStatusValue !== 'clean' && !isTyping"
                            type="button"
                            @click="openIssuesTab"
                            class="flex items-center justify-center gap-1.5 rounded-full h-10 px-3 min-w-10 text-sm font-bold shadow-2xl transition-transform duration-150 ease-out hover:scale-105"
                            :class="{
                                'bg-purple-200 text-purple-950 hover:bg-purple-300': issuesStatusValue === 'info',
                                'bg-amber-500 text-ember-950 hover:bg-amber-400': issuesStatusValue === 'warning',
                                'bg-red-500 text-white hover:bg-red-600': issuesStatusValue === 'error',
                            }"
                            :aria-label="`${issuesSummary.total} script issue${issuesSummary.total === 1 ? '' : 's'}`"
                            :title="`${issuesSummary.total} script issue${issuesSummary.total === 1 ? '' : 's'}`"
                        >
                            <Info v-if="issuesStatusValue === 'info'" class="w-4 h-4" />
                            <AlertTriangle v-else-if="issuesStatusValue === 'warning'" class="w-4 h-4" />
                            <AlertCircle v-else class="w-4 h-4" />
                            <span class="tabular-nums">{{ issuesSummary.total }}</span>
                        </button>
                    </Transition>

                    <button
                        v-if="!previewVisible"
                        @click="previewVisible = true"
                        class="w-12 h-12 rounded-full bg-brass-500 text-ember-950 shadow-2xl flex items-center justify-center hover:bg-brass-400 transition-all transform hover:scale-110"
                        title="Show Preview"
                    >
                        <Eye class="w-6 h-6 text-ember-950" />
                    </button>
                </div>
            </div>

            <WorkbenchDrawer
                :open="drawerOpen"
                :active-tab="drawerTab"
                :issues-badge="issuesSummary.total"
                @close="closeDrawer"
                @update:active-tab="drawerTab = $event"
            >
                <template #issues>
                    <IssuesTab
                        :diagnostics="diagnostics"
                        :hidden-severities="hiddenSeverities"
                        @jump="jumpToDiagnostic"
                        @update:hidden-severities="hiddenSeverities = $event"
                    />
                </template>
                <template #find>
                    <FindReplaceTab
                        :active="drawerOpen && drawerTab === 'find'"
                        :matches="searchMatches"
                        :current-index="searchIndex"
                        :error="searchError"
                        :initial-query="searchInitialQuery"
                        :focus-replace="searchFocusReplace"
                        :focus-nonce="searchFocusNonce"
                        @search="onSearch"
                        @next="onFindNext"
                        @prev="onFindPrev"
                        @replace="onReplaceOne"
                        @replace-all="onReplaceAll"
                        @jump="onJumpMatch"
                    />
                </template>
                <template #outline>
                    <OutlineTab
                        :symbols="outlineSymbols"
                        @jump="jumpToSymbol"
                    />
                </template>
                <template #stats>
                    <StatsTab :stats="manuscriptStats" :loading="manuscriptStatsLoading" />
                </template>
                <template #help>
                    <HelpTab :open-link="props.env.openURL" />
                </template>
            </WorkbenchDrawer>
        </div>

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

    </div>
  </div>

  <BaseModal
    :open="showSpellcheckModal"
    kicker="Writing Tools"
    title="Spell Check"
    message="Control spell check and manage custom words for this script only."
    @close="showSpellcheckModal = false"
  >
    <div class="flex flex-col gap-5 py-1">
        <label class="flex items-center justify-between gap-4 rounded-lg border border-border bg-black/5 px-4 py-3 dark:bg-white/5">
            <p class="text-sm font-bold text-text-main">Enable Spell Check</p>
            <button
                type="button"
                role="switch"
                :aria-checked="spellcheckEnabled"
                class="relative inline-flex h-7 w-12 shrink-0 items-center rounded-full border transition-colors"
                :class="spellcheckEnabled ? 'border-brass-500 bg-brass-500/80' : 'border-border bg-black/10 dark:bg-white/10'"
                @click="spellcheckEnabled = !spellcheckEnabled"
            >
                <span
                    class="inline-block h-5 w-5 rounded-full bg-white shadow transition-transform"
                    :class="spellcheckEnabled ? 'translate-x-6' : 'translate-x-1'"
                ></span>
            </button>
        </label>

        <div class="space-y-1">
            <p class="text-sm font-bold text-text-main">Script Dictionary</p>
            <p class="text-xs leading-relaxed text-text-muted">
                Add custom words for this draft. These entries do not affect any other script.
            </p>
        </div>

        <form class="flex gap-2" @submit.prevent="addDictionaryWord">
            <input
                v-model="dictionaryWord"
                type="text"
                class="flex-1 rounded-lg border border-border bg-black/5 px-3 py-2 text-sm text-text-main outline-none transition-colors placeholder:text-text-muted focus:border-brass-500 dark:bg-white/5"
                placeholder="Add a custom word"
            />
            <button
                type="submit"
                class="rounded-lg bg-brass-500 px-4 py-2 text-sm font-bold text-ember-950 transition-colors hover:bg-brass-400 disabled:opacity-50"
                :disabled="dictionaryWord.trim().length === 0"
            >
                Add
            </button>
        </form>

        <div v-if="spellAllowlist.length === 0" class="rounded-lg border border-dashed border-border bg-black/5 px-4 py-6 text-center text-sm text-text-muted dark:bg-white/5">
            No custom words yet.
        </div>

        <div v-else class="flex flex-col gap-2">
            <div
                v-for="word in spellAllowlist"
                :key="word"
                class="flex items-center justify-between gap-3 rounded-lg border border-border bg-black/5 px-3 py-2 dark:bg-white/5"
            >
                <span class="font-mono text-sm text-text-main">{{ word }}</span>
                <button
                    type="button"
                    class="rounded-md p-2 text-text-muted transition-colors hover:bg-red-500/10 hover:text-red-500"
                    :title="`Remove ${word} from this script dictionary`"
                    @click="removeDictionaryWord(word)"
                >
                    <Trash2 class="h-4 w-4" />
                </button>
            </div>
        </div>

        <div class="flex justify-end pt-2">
            <button
                type="button"
                class="rounded-lg border border-border px-4 py-2 text-sm font-bold text-text-main transition-colors hover:bg-black/5 dark:hover:bg-white/5"
                @click="showSpellcheckModal = false"
            >
                OK
            </button>
        </div>
    </div>
  </BaseModal>

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
.fab-fade-enter-active {
    transition: opacity 500ms ease-out;
}
.fab-fade-enter-from {
    opacity: 0;
}
.fab-fade-enter-to {
    opacity: 1;
}

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
.cm-tooltip-lint .cm-spellMessage {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
}
.cm-tooltip-lint .cm-spellSuggestions {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.3rem;
}
.cm-tooltip-lint .cm-spellLoading,
.cm-tooltip-lint .cm-spellEmpty {
    font-size: 0.85em;
    font-style: italic;
    color: var(--color-text-muted);
}

.no-scrollbar::-webkit-scrollbar { display: none; }
.no-scrollbar { -ms-overflow-style: none; scrollbar-width: none; }
</style>
