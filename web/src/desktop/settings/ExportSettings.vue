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
  <section class="flex flex-col gap-5">
    <header>
      <h3 class="text-base font-semibold text-text-main">Export</h3>
      <p class="text-xs text-text-muted mt-0.5">
        Paper size for every PDF export. Format and layout are chosen per-export in the dialog.
      </p>
    </header>

    <div class="flex flex-col gap-1.5">
      <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-text-muted">Page size</p>
      <ButtonRadioGroup
        :model-value="pageSize"
        :options="pageSizeOptions"
        aria-label="Page size"
        size="compact"
        :class="['max-w-[240px]', loaded ? '' : 'opacity-50 pointer-events-none']"
        @update:model-value="selectPageSize"
      />
    </div>
  </section>
</template>
