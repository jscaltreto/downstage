<script setup lang="ts">
import { onMounted, ref } from 'vue';
import type { DesktopCapabilities } from '../types';
import type { PdfPageSize } from '../../core/types';
import ButtonRadioGroup from '../../components/shared/ButtonRadioGroup.vue';
import type { ButtonRadioOption } from '../../components/shared/button-radio-group';

// Export settings. Page size is a one-time preference here rather than a
// per-export choice in the dialog — most writers settle on one paper size
// and stick with it. Style/layout/gutter stay in the export dialog
// (they're per-export decisions) with the last-used values remembered.

const props = defineProps<{
  env: DesktopCapabilities;
}>();

const pageSize = ref<PdfPageSize>('letter');
const loaded = ref(false);

const pageSizeOptions: ButtonRadioOption<PdfPageSize>[] = [
  { value: 'letter', label: 'Letter', dataAttr: { key: 'page-size', value: 'letter' } },
  { value: 'a4', label: 'A4', dataAttr: { key: 'page-size', value: 'a4' } },
];

onMounted(async () => {
  const prefs = await props.env.getExportPreferences();
  pageSize.value = prefs.pageSize;
  loaded.value = true;
});

async function selectPageSize(next: PdfPageSize) {
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
      <ButtonRadioGroup
        :model-value="pageSize"
        :options="pageSizeOptions"
        aria-label="Page size"
        :class="loaded ? '' : 'opacity-50 pointer-events-none'"
        @update:model-value="selectPageSize"
      />
    </div>

    <p class="text-xs text-text-muted leading-relaxed">
      The paper size used for every PDF export. Format (Manuscript or
      Acting Edition) and layout (Single / 2-up / Booklet) are chosen per
      export in the Export PDF dialog.
    </p>
  </div>
</template>
