<script setup lang="ts">
import { computed, ref } from 'vue';
import { X, AlertTriangle, Search, ListTree, BarChart3, HelpCircle, PanelRight, PanelBottom } from 'lucide-vue-next';
import type { WorkbenchTab } from './workbench-tabs';
export type { WorkbenchTab };

export type DrawerDock = 'bottom' | 'right';

const props = withDefaults(defineProps<{
  open: boolean;
  activeTab: WorkbenchTab;
  issuesBadge?: number;
  // Where the drawer docks relative to the editor. Default keeps the
  // historical bottom-docked behavior; web host never passes this.
  dock?: DrawerDock;
  // Width for the right-docked mode. Ignored in bottom mode.
  rightWidth?: number;
}>(), {
  dock: 'bottom',
  rightWidth: 360,
});

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'update:activeTab', next: WorkbenchTab): void;
  (e: 'update:dock', next: DrawerDock): void;
  (e: 'update:rightWidth', next: number): void;
}>();

function switchTab(tab: WorkbenchTab) {
  emit('update:activeTab', tab);
}

function toggleDock() {
  emit('update:dock', props.dock === 'right' ? 'bottom' : 'right');
}

// In right-dock mode the drawer is typically ~360 px wide — too narrow
// for labeled tabs to fit without clipping. Hide labels and render
// icon-only buttons; `title` attrs carry the readable name.
const isCompact = computed(() => props.dock === 'right');
// Class applied to every tab button. Keeps the label-or-icon branch
// tidy and avoids copy-pasting the class string across five tabs.
const tabButtonClass = computed(() => {
  const base = 'flex items-center border-b-2 transition-colors text-[10px] font-bold uppercase tracking-[0.2em]';
  const layout = isCompact.value ? 'justify-center px-2 py-2 gap-0' : 'gap-2 px-3 py-2.5';
  return `${base} ${layout}`;
});
const tabActiveClass = 'border-brass-500 text-accent';
const tabInactiveClass = 'border-transparent text-text-muted hover:text-text-main';

// The closed/open root style differs per dock:
//   bottom → animated height
//   right  → fixed width (no height animation; v-if flip on open)
const rootStyle = computed<Record<string, string>>(() => {
  if (props.dock === 'right') {
    return props.open ? { width: `${props.rightWidth}px` } : { width: '0px' };
  }
  return { height: props.open ? 'min(40vh, 360px)' : '0px' };
});

const rootClass = computed(() => {
  // Only animate the dimension that actually changes per dock mode —
  // transitioning the orthogonal axis would be dead work that some
  // compositors still allocate layers for.
  const base = 'shrink-0 flex flex-col bg-[var(--color-page-surface)] overflow-hidden';
  if (props.dock === 'right') {
    const right = 'transition-[width,border-color] duration-200 ease-out';
    return `${base} ${right} ${props.open ? 'border-l border-border shadow-[-4px_0_12px_rgba(0,0,0,0.08)]' : 'border-l border-transparent'}`;
  }
  const bottom = 'transition-[height,border-color] duration-200 ease-out';
  return `${base} ${bottom} ${props.open ? 'border-t border-border shadow-[0_-4px_12px_rgba(0,0,0,0.08)]' : 'border-t border-transparent'}`;
});

// Drag state for the right-mode left-edge resize handle. Mirrors the
// sidebar handle in AppDesktop but inlined here so the drawer owns
// its own resize UI.
const dragStartX = ref(0);
const dragStartWidth = ref(0);
function beginResize(e: MouseEvent) {
  if (props.dock !== 'right') return;
  dragStartX.value = e.clientX;
  dragStartWidth.value = props.rightWidth;
  document.body.style.cursor = 'col-resize';
  window.addEventListener('mousemove', onResizeMove);
  window.addEventListener('mouseup', onResizeEnd, { once: true });
}
function onResizeMove(e: MouseEvent) {
  // Dragging the LEFT edge to the LEFT grows the drawer — negate
  // delta so left-drag increases width.
  const delta = dragStartX.value - e.clientX;
  emit('update:rightWidth', dragStartWidth.value + delta);
}
function onResizeEnd() {
  window.removeEventListener('mousemove', onResizeMove);
  document.body.style.cursor = '';
}
</script>

<template>
  <!-- Optional left-edge resize handle for the right-docked mode. Kept
       outside the <section> so it sits at the boundary between editor
       and drawer; absolutely positioned so it doesn't affect layout. -->
  <div
    v-if="dock === 'right' && open"
    class="drawer-right-resize-handle shrink-0"
    role="separator"
    aria-orientation="vertical"
    :aria-valuenow="rightWidth"
    title="Drag to resize drawer"
    @mousedown.prevent="beginResize"
  ></div>
  <section
    :class="rootClass"
    :style="rootStyle"
    :aria-hidden="!open"
    role="region"
    aria-label="Workbench"
  >
    <header
      class="flex items-center justify-between gap-2 border-b border-border bg-[var(--color-toolbar-bg)]"
      :class="isCompact ? 'px-2' : 'px-4'"
    >
      <div class="flex items-center gap-1 min-w-0" role="tablist">
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'outline'"
          :class="[tabButtonClass, activeTab === 'outline' ? tabActiveClass : tabInactiveClass]"
          :title="isCompact ? 'Outline' : undefined"
          :aria-label="isCompact ? 'Outline' : undefined"
          @click="switchTab('outline')"
        >
          <ListTree class="h-3.5 w-3.5" />
          <span v-if="!isCompact">Outline</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'stats'"
          :class="[tabButtonClass, activeTab === 'stats' ? tabActiveClass : tabInactiveClass]"
          :title="isCompact ? 'Stats' : undefined"
          :aria-label="isCompact ? 'Stats' : undefined"
          @click="switchTab('stats')"
        >
          <BarChart3 class="h-3.5 w-3.5" />
          <span v-if="!isCompact">Stats</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'issues'"
          :class="[tabButtonClass, activeTab === 'issues' ? tabActiveClass : tabInactiveClass]"
          :title="isCompact ? `Issues${issuesBadge ? ` (${issuesBadge})` : ''}` : undefined"
          :aria-label="isCompact ? 'Issues' : undefined"
          @click="switchTab('issues')"
        >
          <AlertTriangle class="h-3.5 w-3.5" />
          <span v-if="!isCompact">Issues</span>
          <span
            v-if="issuesBadge && issuesBadge > 0"
            class="rounded-full bg-black/10 px-1.5 py-0.5 text-[9px] font-bold normal-case tracking-normal text-text-main dark:bg-white/10"
            :class="isCompact ? 'ml-1' : ''"
          >{{ issuesBadge }}</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'find'"
          :class="[tabButtonClass, activeTab === 'find' ? tabActiveClass : tabInactiveClass]"
          :title="isCompact ? 'Find & Replace' : undefined"
          :aria-label="isCompact ? 'Find and Replace' : undefined"
          @click="switchTab('find')"
        >
          <Search class="h-3.5 w-3.5" />
          <span v-if="!isCompact">Find &amp; Replace</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'help'"
          :class="[tabButtonClass, activeTab === 'help' ? tabActiveClass : tabInactiveClass]"
          :title="isCompact ? 'Help' : undefined"
          :aria-label="isCompact ? 'Help' : undefined"
          @click="switchTab('help')"
        >
          <HelpCircle class="h-3.5 w-3.5" />
          <span v-if="!isCompact">Help</span>
        </button>
      </div>
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="rounded-full p-1.5 text-text-muted transition-colors hover:bg-black/5 hover:text-text-main dark:hover:bg-white/5"
          :aria-label="dock === 'right' ? 'Dock drawer to bottom' : 'Dock drawer to right'"
          :title="dock === 'right' ? 'Dock to bottom' : 'Dock to right'"
          @click="toggleDock"
        >
          <PanelRight v-if="dock === 'bottom'" class="h-4 w-4" />
          <PanelBottom v-else class="h-4 w-4" />
        </button>
        <button
          type="button"
          class="rounded-full p-1.5 text-text-muted transition-colors hover:bg-black/5 hover:text-text-main dark:hover:bg-white/5"
          aria-label="Close workbench"
          @click="emit('close')"
        >
          <X class="h-4 w-4" />
        </button>
      </div>
    </header>

    <div class="flex-1 min-h-0 min-w-0 overflow-hidden">
      <div v-show="activeTab === 'outline'" class="flex h-full min-w-0 flex-col overflow-hidden">
        <slot name="outline" />
      </div>
      <div v-show="activeTab === 'stats'" class="flex h-full min-w-0 flex-col overflow-hidden">
        <slot name="stats" />
      </div>
      <div v-show="activeTab === 'issues'" class="flex h-full min-w-0 flex-col overflow-hidden">
        <slot name="issues" />
      </div>
      <div v-show="activeTab === 'find'" class="flex h-full min-w-0 flex-col overflow-hidden">
        <slot name="find" />
      </div>
      <div v-show="activeTab === 'help'" class="flex h-full min-w-0 flex-col overflow-hidden">
        <slot name="help" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.drawer-right-resize-handle {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background-color 0.12s ease-out;
}
.drawer-right-resize-handle:hover,
.drawer-right-resize-handle:active {
  background: rgba(227, 168, 87, 0.25);
}
</style>
