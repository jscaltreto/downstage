<script setup lang="ts">
import { onMounted, ref } from 'vue';
import type { DesktopCapabilities } from '../types';
import type { PdfPageSize } from '../../core/types';

// Export settings. Page size is a one-time preference here rather than a
// per-export choice in the dialog — most writers settle on one paper size
// and stick with it. Style/layout/gutter stay in the export dialog
// (they're per-export decisions) with the last-used values remembered.

const props = defineProps<{
  env: DesktopCapabilities;
}>();

const pageSize = ref<PdfPageSize>('letter');
const loaded = ref(false);

onMounted(async () => {
  const prefs = await props.env.getExportPreferences();
  pageSize.value = prefs.pageSize;
  loaded.value = true;
});

async function selectPageSize(next: PdfPageSize) {
  if (pageSize.value === next) return;
  pageSize.value = next;
  const prefs = await props.env.getExportPreferences();
  await props.env.setExportPreferences({ ...prefs, pageSize: next });
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-bold text-text-main">Export</h3>

    <div class="rounded-lg border border-border bg-black/5 p-4 dark:bg-white/5">
      <p class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-3">Page size</p>
      <div
        role="radiogroup"
        aria-label="Page size"
        class="grid grid-cols-2 gap-2 p-1 rounded-lg bg-black/5 dark:bg-white/5 border border-border"
      >
        <button
          type="button"
          role="radio"
          :aria-checked="pageSize === 'letter'"
          data-page-size="letter"
          :disabled="!loaded"
          class="px-4 py-2 rounded-md text-sm font-bold transition-colors disabled:opacity-50"
          :class="pageSize === 'letter'
            ? 'bg-brass-500 text-ember-850 shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
          @click="selectPageSize('letter')"
        >
          Letter
        </button>
        <button
          type="button"
          role="radio"
          :aria-checked="pageSize === 'a4'"
          data-page-size="a4"
          :disabled="!loaded"
          class="px-4 py-2 rounded-md text-sm font-bold transition-colors disabled:opacity-50"
          :class="pageSize === 'a4'
            ? 'bg-brass-500 text-ember-850 shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
          @click="selectPageSize('a4')"
        >
          A4
        </button>
      </div>
    </div>

    <p class="text-xs text-text-muted leading-relaxed">
      The paper size used for every PDF export. Format (Manuscript or
      Acting Edition) and layout (Single / 2-up / Booklet) are chosen per
      export in the Export PDF dialog.
    </p>
  </div>
</template>
