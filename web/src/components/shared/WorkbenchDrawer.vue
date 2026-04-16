<script setup lang="ts">
import { X, AlertTriangle, Search, ListTree, BarChart3 } from 'lucide-vue-next';

export type WorkbenchTab = 'issues' | 'find' | 'outline' | 'stats';

defineProps<{
  open: boolean;
  activeTab: WorkbenchTab;
  issuesBadge?: number;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'update:activeTab', next: WorkbenchTab): void;
}>();

function switchTab(tab: WorkbenchTab) {
  emit('update:activeTab', tab);
}
</script>

<template>
  <section
    class="shrink-0 flex flex-col bg-[var(--color-page-surface)] shadow-[0_-8px_24px_rgba(0,0,0,0.12)] overflow-hidden transition-[height,border-color] duration-200 ease-out"
    :class="open ? 'border-t border-border' : 'border-t border-transparent'"
    :style="{ height: open ? 'min(40vh, 360px)' : '0px' }"
    :aria-hidden="!open"
    role="region"
    aria-label="Workbench"
  >
    <header class="flex items-center justify-between gap-3 border-b border-border bg-[var(--color-toolbar-bg)] px-4">
      <div class="flex items-center gap-1" role="tablist">
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'outline'"
          class="flex items-center gap-2 px-3 py-2.5 text-[10px] font-bold uppercase tracking-[0.2em] border-b-2 transition-colors"
          :class="activeTab === 'outline'
            ? 'border-brass-500 text-accent'
            : 'border-transparent text-text-muted hover:text-text-main'"
          @click="switchTab('outline')"
        >
          <ListTree class="h-3.5 w-3.5" />
          <span>Outline</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'stats'"
          class="flex items-center gap-2 px-3 py-2.5 text-[10px] font-bold uppercase tracking-[0.2em] border-b-2 transition-colors"
          :class="activeTab === 'stats'
            ? 'border-brass-500 text-accent'
            : 'border-transparent text-text-muted hover:text-text-main'"
          @click="switchTab('stats')"
        >
          <BarChart3 class="h-3.5 w-3.5" />
          <span>Stats</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'issues'"
          class="flex items-center gap-2 px-3 py-2.5 text-[10px] font-bold uppercase tracking-[0.2em] border-b-2 transition-colors"
          :class="activeTab === 'issues'
            ? 'border-brass-500 text-accent'
            : 'border-transparent text-text-muted hover:text-text-main'"
          @click="switchTab('issues')"
        >
          <AlertTriangle class="h-3.5 w-3.5" />
          <span>Issues</span>
          <span
            v-if="issuesBadge && issuesBadge > 0"
            class="rounded-full bg-black/10 px-1.5 py-0.5 text-[9px] font-bold normal-case tracking-normal text-text-main dark:bg-white/10"
          >{{ issuesBadge }}</span>
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'find'"
          class="flex items-center gap-2 px-3 py-2.5 text-[10px] font-bold uppercase tracking-[0.2em] border-b-2 transition-colors"
          :class="activeTab === 'find'
            ? 'border-brass-500 text-accent'
            : 'border-transparent text-text-muted hover:text-text-main'"
          @click="switchTab('find')"
        >
          <Search class="h-3.5 w-3.5" />
          <span>Find &amp; Replace</span>
        </button>
      </div>
      <button
        type="button"
        class="rounded-full p-1.5 text-text-muted transition-colors hover:bg-black/5 hover:text-text-main dark:hover:bg-white/5"
        aria-label="Close workbench"
        @click="emit('close')"
      >
        <X class="h-4 w-4" />
      </button>
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
    </div>
  </section>
</template>
