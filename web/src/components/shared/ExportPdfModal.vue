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

// Preserve the last condensed layout when the user toggles style so a
// booklet/2-up choice isn't erased by a quick detour to Manuscript. The
// Layout and Gutter controls are hidden when style !== 'condensed', and
// handleConfirm emits layout='single' for manuscript exports regardless
// of the stored value, so manuscript validation still sees a valid combo.
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

// Landscape sheet width in millimeters, which is the hard upper bound on
// gutter: a gutter at or above this value would leave zero-width cells.
// For Letter the landscape width is 11in (279.4mm); for A4 it is 297mm.
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

// Switching the unit preserves the physical gutter by converting the
// displayed value. Rounded to 4 decimal places for inches and 2 for
// millimeters, which is well below the step the user can dial in.
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
  // Manuscript exports force layout=single so downstream validation
  // (standard + 2up/booklet is rejected) always sees a valid combo.
  // lastCondensedLayout is preserved for the next Acting Edition export.
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
              aria-describedby="gutter-help"
              :aria-invalid="!gutterIsValid"
              class="flex-1 px-3 py-2 rounded-md text-sm font-bold bg-black/5 dark:bg-white/5 border text-text-main focus:outline-none focus:ring-2 focus:ring-brass-500/40"
              :class="gutterError ? 'border-red-500/60' : 'border-border'"
            />
            <select
              :value="gutterUnit"
              @change="changeGutterUnit(($event.target as HTMLSelectElement).value as 'in' | 'mm')"
              data-testid="gutter-unit"
              class="px-3 py-2 rounded-md text-sm font-bold bg-black/5 dark:bg-white/5 border border-border text-text-main focus:outline-none focus:ring-2 focus:ring-brass-500/40"
            >
              <option value="in">in</option>
              <option value="mm">mm</option>
            </select>
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
