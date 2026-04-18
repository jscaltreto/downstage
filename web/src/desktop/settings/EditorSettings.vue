<script setup lang="ts">
import { computed } from 'vue';
import type { Store } from '../../core/store';

const props = defineProps<{
  store: Store;
}>();

const previewVisible = computed<boolean>({
  get: () => !props.store.state.previewHidden,
  set: (v) => { props.store.state.previewHidden = !v; },
});

const spellcheckEnabled = computed<boolean>({
  get: () => !props.store.state.spellcheckDisabled,
  set: (v) => { props.store.state.spellcheckDisabled = !v; },
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-bold text-text-main">Editor</h3>
    <label class="flex items-center justify-between gap-4 rounded-lg border border-border bg-black/5 px-4 py-3 dark:bg-white/5">
      <div class="min-w-0">
        <p class="text-sm font-bold text-text-main">Show Preview</p>
        <p class="text-xs text-text-muted">Render the manuscript alongside the editor.</p>
      </div>
      <input type="checkbox" class="scale-125" v-model="previewVisible" />
    </label>

    <label class="flex items-center justify-between gap-4 rounded-lg border border-border bg-black/5 px-4 py-3 dark:bg-white/5">
      <div class="min-w-0">
        <p class="text-sm font-bold text-text-main">Enable Spell Check</p>
        <p class="text-xs text-text-muted">Underline unknown words in the editor.</p>
      </div>
      <input type="checkbox" class="scale-125" v-model="spellcheckEnabled" />
    </label>
  </div>
</template>
