<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import BaseModal from './BaseModal.vue';
import type { ExportPdfOptions, PdfExportStyle, PdfLayout, PdfPageSize } from '../../core/types';

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
const layout = ref<PdfLayout>(props.initialOptions.layout);
const gutterValue = ref<number>(parseGutterValue(props.initialOptions.bookletGutter));
const gutterUnit = ref<'in' | 'mm'>(parseGutterUnit(props.initialOptions.bookletGutter));

function parseGutterValue(s: string): number {
  const m = s.match(/^\s*(-?[\d.]+)\s*(in|mm)\s*$/i);
  return m ? Number(m[1]) : 0.125;
}

function parseGutterUnit(s: string): 'in' | 'mm' {
  const m = s.match(/^\s*(-?[\d.]+)\s*(in|mm)\s*$/i);
  return m ? (m[2].toLowerCase() as 'in' | 'mm') : 'in';
}

watch(
  () => [props.open, props.initialOptions] as const,
  ([isOpen, initial]) => {
    if (isOpen) {
      pageSize.value = initial.pageSize;
      style.value = initial.style;
      layout.value = initial.layout;
      gutterValue.value = parseGutterValue(initial.bookletGutter);
      gutterUnit.value = parseGutterUnit(initial.bookletGutter);
    }
  },
  { deep: true },
);

// Snap layout back to single when switching to Manuscript so a stray
// 2-up/booklet selection never leaks into a manuscript export (validation
// would reject it anyway).
watch(style, (next) => {
  if (next !== 'condensed') {
    layout.value = 'single';
  }
});

const condensedDerivedSize = computed(() =>
  pageSize.value === 'a4' ? 'A5 (148 × 210 mm)' : 'half-letter (5.5 × 8.5 in)',
);

const gutterString = computed(() => `${gutterValue.value}${gutterUnit.value}`);

const gutterIsValid = computed(() => gutterValue.value >= 0 && Number.isFinite(gutterValue.value));

const canConfirm = computed(() => {
  if (layout.value === 'booklet') return gutterIsValid.value;
  return true;
});

function selectPageSize(value: PdfPageSize) {
  pageSize.value = value;
}

function selectStyle(value: PdfExportStyle) {
  style.value = value;
}

function selectLayout(value: PdfLayout) {
  layout.value = value;
}

function handleConfirm() {
  if (!canConfirm.value) return;
  emit('confirm', {
    pageSize: pageSize.value,
    style: style.value,
    layout: layout.value,
    bookletGutter: gutterString.value,
  });
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
        class="mt-3 mb-5 px-3 py-2 rounded-lg border border-brass-500/30 bg-brass-500/5 text-xs text-text-muted leading-relaxed"
      >
        Acting edition renders on <strong class="font-semibold text-text-main">{{ condensedDerivedSize }}</strong> —
        the half-sheet derived from the selected page size.
      </div>

      <template v-if="style === 'condensed'">
        <label class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2">
          Layout
        </label>
        <div
          role="radiogroup"
          aria-label="PDF layout"
          data-testid="layout-group"
          class="grid grid-cols-3 gap-2 mb-5 p-1 rounded-lg bg-black/5 dark:bg-white/5 border border-border"
        >
          <button
            type="button"
            role="radio"
            :aria-checked="layout === 'single'"
            data-pdf-layout="single"
            class="px-3 py-2 rounded-md text-xs font-bold transition-colors"
            :class="layout === 'single'
              ? 'bg-brass-500 text-ember-850 shadow-sm'
              : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
            @click="selectLayout('single')"
          >
            Single page
          </button>
          <button
            type="button"
            role="radio"
            :aria-checked="layout === '2up'"
            data-pdf-layout="2up"
            class="px-3 py-2 rounded-md text-xs font-bold transition-colors"
            :class="layout === '2up'
              ? 'bg-brass-500 text-ember-850 shadow-sm'
              : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
            @click="selectLayout('2up')"
          >
            2-up
          </button>
          <button
            type="button"
            role="radio"
            :aria-checked="layout === 'booklet'"
            data-pdf-layout="booklet"
            class="px-3 py-2 rounded-md text-xs font-bold transition-colors"
            :class="layout === 'booklet'
              ? 'bg-brass-500 text-ember-850 shadow-sm'
              : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10'"
            @click="selectLayout('booklet')"
          >
            Booklet
          </button>
        </div>

        <div v-if="layout === 'booklet'" data-testid="gutter-row" class="mb-5">
          <label class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2 block">
            Booklet gutter
          </label>
          <div class="flex gap-2 items-center">
            <input
              type="number"
              step="0.025"
              min="0"
              v-model.number="gutterValue"
              data-testid="gutter-value"
              class="flex-1 px-3 py-2 rounded-md text-sm font-bold bg-black/5 dark:bg-white/5 border border-border text-text-main focus:outline-none focus:ring-2 focus:ring-brass-500/40"
            />
            <select
              v-model="gutterUnit"
              data-testid="gutter-unit"
              class="px-3 py-2 rounded-md text-sm font-bold bg-black/5 dark:bg-white/5 border border-border text-text-main focus:outline-none focus:ring-2 focus:ring-brass-500/40"
            >
              <option value="in">in</option>
              <option value="mm">mm</option>
            </select>
          </div>
          <p class="mt-2 text-xs text-text-muted leading-relaxed">
            Inside spacing between the two pages on each sheet. Booklet output
            is duplex: print double-sided, then fold in half.
          </p>
        </div>
      </template>

      <div v-else class="mt-3 mb-5 text-xs text-text-muted italic">
        2-up and booklet layouts are available for Acting Edition only.
      </div>

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
          :disabled="!canConfirm"
          class="flex-1 px-4 py-2.5 rounded-lg bg-brass-500 text-ember-850 text-sm font-bold hover:brightness-110 transition-all shadow-lg disabled:opacity-50 disabled:cursor-not-allowed"
          @click="handleConfirm"
        >
          Export PDF
        </button>
      </div>
    </div>
  </BaseModal>
</template>
