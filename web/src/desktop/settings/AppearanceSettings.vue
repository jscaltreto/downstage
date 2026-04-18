<script setup lang="ts">
import type { Store, Theme } from '../../core/store';

// Appearance preferences. Only theme today. Sidebar visibility and
// preview visibility used to live here but they're transient view
// toggles, not persisted preferences — those belong to the main UI
// (sidebar chevron, preview eye button, menu, keyboard shortcuts).

const props = defineProps<{
  store: Store;
}>();

const themes: Array<{ id: Theme; label: string }> = [
  { id: 'light', label: 'Light' },
  { id: 'dark', label: 'Dark' },
  { id: 'system', label: 'Follow System' },
];

function setTheme(t: Theme) {
  props.store.setTheme(t);
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-bold text-text-main">Appearance</h3>

    <div class="rounded-lg border border-border bg-black/5 px-4 py-3 dark:bg-white/5">
      <p class="text-sm font-bold text-text-main mb-2">Theme</p>
      <div class="flex gap-2">
        <button
          v-for="t in themes"
          :key="t.id"
          type="button"
          class="px-3 py-1.5 rounded-md text-sm transition-colors border"
          :class="store.state.theme === t.id
            ? 'bg-brass-500/10 text-brass-500 font-bold border-brass-500/20'
            : 'text-text-main border-border hover:bg-black/5 dark:hover:bg-white/5'"
          @click="setTheme(t.id)"
        >
          {{ t.label }}
        </button>
      </div>
    </div>
  </div>
</template>
