<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import BaseModal from './BaseModal.vue';
import ButtonRadioGroup from './ButtonRadioGroup.vue';
import type { ButtonRadioOption } from './button-radio-group';
import type { ExportPdfOptions, PdfExportStyle, PdfLayout, PdfPageSize } from '../../core/types';

const props = withDefaults(
  defineProps<{
    open: boolean;
    initialOptions: ExportPdfOptions;
    // Hide the Page size row. The desktop host exposes page size in
    // Settings instead of the dialog, so it's persistent rather than
    // per-export. Web passes page size through the dialog as before.
    hidePageSize?: boolean;
  }>(),
  { hidePageSize: false },
);

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

// Remembers the user's 2up/booklet pick so a quick detour to Manuscript
// (which hides the layout row) doesn't erase their choice. Restored when
// style flips back to condensed.
const lastCondensedLayout = ref<PdfLayout>(
  props.initialOptions.layout === 'single' ? 'single' : props.initialOptions.layout,
);

watch(
  () => [props.open, props.initialOptions.layout] as const,
  ([isOpen, initial]) => {
    if (isOpen) {
      lastCondensedLayout.value = initial;
    }
  },
);

watch(layout, (next) => {
  if (style.value === 'condensed') {
    lastCondensedLayout.value = next;
  }
});

watch(style, (next) => {
  if (next === 'condensed') {
    layout.value = lastCondensedLayout.value;
  }
});

const condensedDerivedSize = computed(() =>
  pageSize.value === 'a4' ? 'A5 (148 × 210 mm)' : 'half-letter (5.5 × 8.5 in)',
);

const maxGutterMM = computed(() => (pageSize.value === 'a4' ? 297 : 279.4));

function gutterInMM(): number {
  const v = gutterValue.value;
  if (!Number.isFinite(v)) return NaN;
  return gutterUnit.value === 'in' ? v * 25.4 : v;
}

const maxGutterDisplay = computed(() => {
  const mm = maxGutterMM.value;
  if (gutterUnit.value === 'in') return `${(mm / 25.4).toFixed(2)}in`;
  return `${mm.toFixed(1)}mm`;
});

const gutterString = computed(() => `${gutterValue.value}${gutterUnit.value}`);

const gutterError = computed<string | null>(() => {
  const v = gutterValue.value;
  if (!Number.isFinite(v)) return 'Gutter must be a number';
  if (v < 0) return 'Gutter must be non-negative';
  if (gutterInMM() >= maxGutterMM.value) {
    return `Gutter must be under ${maxGutterDisplay.value} for the selected page size`;
  }
  return null;
});

const gutterIsValid = computed(() => gutterError.value === null);

const canConfirm = computed(() => {
  // Gutter only matters for Acting Edition booklet exports. Manuscript
  // exports force layout=single downstream (see handleConfirm), so a
  // stale invalid gutter value should not block them.
  if (style.value !== 'condensed') return true;
  if (layout.value === 'booklet') return gutterIsValid.value;
  return true;
});

// Preserve the physical gutter when the user flips the unit: 0.125in
// becomes 3.18mm, not 0.125mm. Round to 2 dp for mm and 4 dp for inches
// so round-trip drift stays below the input's step.
function changeGutterUnit(next: 'in' | 'mm') {
  const prev = gutterUnit.value;
  if (next === prev) return;
  const v = gutterValue.value;
  if (Number.isFinite(v)) {
    if (prev === 'in' && next === 'mm') {
      gutterValue.value = Math.round(v * 25.4 * 100) / 100;
    } else if (prev === 'mm' && next === 'in') {
      gutterValue.value = Math.round((v / 25.4) * 10000) / 10000;
    }
  }
  gutterUnit.value = next;
}

// Option tables for the button-radio groups below. The `dataAttr` keys
// mirror the pre-refactor data-* attributes so Playwright selectors
// (e2e/pages/EditorPage.ts) keep working without churn.
const pageSizeOptions: ButtonRadioOption<PdfPageSize>[] = [
  { value: 'letter', label: 'Letter', dataAttr: { key: 'page-size', value: 'letter' } },
  { value: 'a4', label: 'A4', dataAttr: { key: 'page-size', value: 'a4' } },
];
const styleOptions: ButtonRadioOption<PdfExportStyle>[] = [
  { value: 'standard', label: 'Manuscript', dataAttr: { key: 'export-style', value: 'standard' } },
  { value: 'condensed', label: 'Acting Edition', dataAttr: { key: 'export-style', value: 'condensed' } },
];
const layoutOptions: ButtonRadioOption<PdfLayout>[] = [
  { value: 'single', label: 'Single page', dataAttr: { key: 'pdf-layout', value: 'single' } },
  { value: '2up', label: '2-up', dataAttr: { key: 'pdf-layout', value: '2up' } },
  { value: 'booklet', label: 'Booklet', dataAttr: { key: 'pdf-layout', value: 'booklet' } },
];
const gutterUnitOptions: ButtonRadioOption<'in' | 'mm'>[] = [
  { value: 'in', label: 'in', dataAttr: { key: 'gutter-unit', value: 'in' } },
  { value: 'mm', label: 'mm', dataAttr: { key: 'gutter-unit', value: 'mm' } },
];

function handleConfirm() {
  if (!canConfirm.value) return;
  // Force layout=single for Manuscript so config.Validate still sees a
  // valid combo (standard + 2up/booklet is rejected). lastCondensedLayout
  // keeps the user's actual pick around for the next condensed export.
  const emittedLayout = style.value === 'condensed' ? layout.value : 'single';
  emit('confirm', {
    pageSize: pageSize.value,
    style: style.value,
    layout: emittedLayout,
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
      <template v-if="!hidePageSize">
        <label class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2">
          Page size
        </label>
        <ButtonRadioGroup
          v-model="pageSize"
          :options="pageSizeOptions"
          aria-label="Page size"
          class="mb-5"
        />
      </template>

      <label class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2">
        Format
      </label>
      <ButtonRadioGroup
        v-model="style"
        :options="styleOptions"
        aria-label="Export format"
      />

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
        <ButtonRadioGroup
          v-model="layout"
          :options="layoutOptions"
          aria-label="PDF layout"
          :columns="3"
          size="sm"
          data-testid="layout-group"
          class="mb-5"
        />

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
              aria-describedby="gutter-help"
              :aria-invalid="!gutterIsValid"
              class="flex-1 px-3 py-2 rounded-md text-sm font-bold bg-black/5 dark:bg-white/5 border text-text-main focus:outline-none focus:ring-2 focus:ring-brass-500/40"
              :class="gutterError ? 'border-red-500/60' : 'border-border'"
            />
            <ButtonRadioGroup
              :model-value="gutterUnit"
              :options="gutterUnitOptions"
              aria-label="Gutter unit"
              columns="inline"
              size="xs"
              data-testid="gutter-unit"
              @update:model-value="changeGutterUnit"
            />
          </div>
          <p
            v-if="gutterError"
            data-testid="gutter-error"
            class="mt-2 text-xs text-red-500 leading-relaxed"
          >
            {{ gutterError }}
          </p>
          <p
            v-else
            id="gutter-help"
            class="mt-2 text-xs text-text-muted leading-relaxed"
          >
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
