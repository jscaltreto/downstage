<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch, inject, nextTick } from 'vue';
import { 
    Bold, Italic, Underline, MessageSquare, ChevronRight, 
    GalleryVerticalEnd, GalleryVertical, FilePlus2, Eye, EyeOff, HelpCircle, X, Music,
    Sun, Moon, ScrollText, BookOpenText
} from 'lucide-vue-next';
import { Engine } from '../../core/engine';
import type { Store } from '../../core/store';
import type { EditorEnv } from '../../core/types';
import PreviewFrame from './PreviewFrame.vue';
import ToolbarButton from './ToolbarButton.vue';

const props = defineProps<{
  env: EditorEnv;
  content: string;
  style: string;
}>();

const emit = defineEmits<{
  (e: 'update:content', value: string): void;
  (e: 'update:style', value: string): void;
  (e: 'toggle-help'): void;
}>();

const store = inject<Store>('store')!;
const editorContainer = ref<HTMLElement | null>(null);
const previewFrameComponent = ref<InstanceType<typeof PreviewFrame> | null>(null);
const renderedHtml = ref("");
let engine: Engine | null = null;

const previewVisible = ref(localStorage.getItem("downstage-editor-preview-hidden") !== "true");

let lastRenderRequestId = 0;
let renderTimer: number | null = null;

async function scheduleRender(content: string, style: string) {
    if (!previewVisible.value) return;

    if (renderTimer) window.clearTimeout(renderTimer);
    
    renderTimer = window.setTimeout(async () => {
        renderTimer = null;
        const requestId = ++lastRenderRequestId;
        const html = await props.env.renderHTML(content, style);
        
        // Only update if this is still the latest request
        if (requestId === lastRenderRequestId) {
            renderedHtml.value = html;
        }
    }, 300);
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
    props.env.renderHTML(props.content, props.style).then(h => {
        renderedHtml.value = h;
    });
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
            
            <ToolbarButton @click="handleFormat('cue')" title="Character Cue"><template #icon><MessageSquare class="w-4 h-4" /></template></ToolbarButton>
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
                <PreviewFrame ref="previewFrameComponent" :html="renderedHtml" />
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

.no-scrollbar::-webkit-scrollbar { display: none; }
.no-scrollbar { -ms-overflow-style: none; scrollbar-width: none; }
</style>
