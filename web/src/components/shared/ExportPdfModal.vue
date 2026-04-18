<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import BaseModal from './BaseModal.vue';
import type { ExportPdfOptions, PdfExportStyle, PdfPageSize } from '../../core/types';

const props = defineProps<{
  open: boolean;
  initialOptions: ExportPdfOptions;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'confirm', options: ExportPdfOptions): void;
}>();

const pageSize = ref<PdfPageSize>(props.initialOptions.pageSize);
const style = ref<PdfExportStyle>(props.initialOptions.style);

watch(
  () => [props.open, props.initialOptions.pageSize, props.initialOptions.style] as const,
  ([isOpen, initialSize, initialStyle]) => {
    if (isOpen) {
      pageSize.value = initialSize;
      style.value = initialStyle;
    }
  },
);

const condensedDerivedSize = computed(() =>
  pageSize.value === 'a4' ? 'A5 (148 × 210 mm)' : 'half-letter (5.5 × 8.5 in)',
);

function selectPageSize(value: PdfPageSize) {
  pageSize.value = value;
}

function selectStyle(value: PdfExportStyle) {
  style.value = value;
}

function handleConfirm() {
  emit('confirm', { pageSize: pageSize.value, style: style.value });
}
</script>

<template>
  <BaseModal
    :open="open"
    title="Export PDF"
    message="Choose the format and sheet size for this export."
    @close="emit('close')"
  >
    <div class="flex flex-col py-2 w-[392px]">
      <label class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2">
        Page size
      </label>
      <div
        role="radiogroup"
        aria-label="Page size"
        class="grid grid-cols-2 gap-2 mb-5 p-1 rounded-lg bg-black/5 dark:bg-white/5 border border-border"
      >
        <button
          type="button"
          role="radio"
          :aria-checked="pageSize === 'letter'"
          data-page-size="letter"
          class="px-4 py-2 rounded-md text-sm font-bold transition-colors"
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
          class="px-4 py-2 rounded-md text-sm font-bold transition-colors"
          :class="pageSize === 'a4'
            ? 'bg-brass-500 text-ember-850 shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
          @click="selectPageSize('a4')"
        >
          A4
        </button>
      </div>

      <label class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2">
        Format
      </label>
      <div
        role="radiogroup"
        aria-label="Export format"
        class="grid grid-cols-2 gap-2 p-1 rounded-lg bg-black/5 dark:bg-white/5 border border-border"
      >
        <button
          type="button"
          role="radio"
          :aria-checked="style === 'standard'"
          data-export-style="standard"
          class="px-4 py-2 rounded-md text-sm font-bold transition-colors"
          :class="style === 'standard'
            ? 'bg-brass-500 text-ember-850 shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
          @click="selectStyle('standard')"
        >
          Manuscript
        </button>
        <button
          type="button"
          role="radio"
          :aria-checked="style === 'condensed'"
          data-export-style="condensed"
          class="px-4 py-2 rounded-md text-sm font-bold transition-colors"
          :class="style === 'condensed'
            ? 'bg-brass-500 text-ember-850 shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
          @click="selectStyle('condensed')"
        >
          Acting Edition
        </button>
      </div>

      <div
        v-if="style === 'condensed'"
        data-testid="condensed-sheet-note"
        class="mt-3 mb-6 px-3 py-2 rounded-lg border border-brass-500/30 bg-brass-500/5 text-xs text-text-muted leading-relaxed"
      >
        Acting edition renders on <strong class="font-semibold text-text-main">{{ condensedDerivedSize }}</strong> —
        the half-sheet derived from the selected page size.
      </div>
      <div v-else class="mb-6" />

      <div class="flex gap-3 w-full">
        <button
          type="button"
          class="flex-1 px-4 py-2.5 rounded-lg border border-border text-sm font-bold hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
          @click="emit('close')"
        >
          Cancel
        </button>
        <button
          type="button"
          data-testid="export-confirm"
          class="flex-1 px-4 py-2.5 rounded-lg bg-brass-500 text-ember-850 text-sm font-bold hover:brightness-110 transition-all shadow-lg"
          @click="handleConfirm"
        >
          Export PDF
        </button>
      </div>
    </div>
  </BaseModal>
</template>
