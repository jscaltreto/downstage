<script setup lang="ts" generic="T extends string | number">
import { computed } from 'vue';
import type { ButtonRadioOption } from './button-radio-group';

// Shared button-style radio group. Replaces six hand-rolled copies of
// the same `<div role="radiogroup"><button role="radio">...` block —
// the pattern used for page size, export format, PDF layout, gutter
// unit, Settings > Export page size, etc.
//
// Three knobs:
//   - `columns`: number for an N-column grid, or 'inline' for a flex
//     row (used when the group sits next to another input).
//   - `size`: 'md' (default, roomy), 'sm' (tighter, for 3-col groups
//     that need to share width), 'xs' (tightest, for inline groups
//     adjacent to an input).
//   - `ariaLabel`: required; read by screen readers as the group label.
//
// Option-level `dataAttr` preserves per-button `data-*` hooks so e2e
// tests (Playwright) and headless unit tests can keep targeting stable
// selectors across the refactor.

type Size = 'md' | 'sm' | 'xs';
type Layout = number | 'inline';

const props = withDefaults(
  defineProps<{
    modelValue: T;
    options: ButtonRadioOption<T>[];
    ariaLabel: string;
    columns?: Layout;
    size?: Size;
  }>(),
  { columns: 2, size: 'md' },
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: T): void;
}>();

const containerClass = computed(() => {
  const base = 'p-1 bg-black/5 dark:bg-white/5 border border-border';
  if (props.columns === 'inline') {
    // rounded-md matches the smaller, inline variant (sits next to an
    // input of similar height). Flex gap is tighter than the grid
    // version because inline groups are narrower overall.
    return `${base} flex gap-1 rounded-md`;
  }
  return `${base} grid gap-2 rounded-lg`;
});

const containerStyle = computed(() => {
  if (props.columns === 'inline') return undefined;
  // Inline style avoids Tailwind JIT surprises with dynamic grid-cols-N
  // classes. Callers pass a concrete number; this stays simple.
  return { gridTemplateColumns: `repeat(${props.columns}, minmax(0, 1fr))` };
});

function buttonClass(selected: boolean): string {
  const shape =
    props.size === 'xs'
      ? 'px-3 py-1.5 rounded text-sm font-bold'
      : props.size === 'sm'
        ? 'px-3 py-2 rounded-md text-xs font-bold'
        : 'px-4 py-2 rounded-md text-sm font-bold';
  const state = selected
    ? 'bg-brass-500 text-ember-850 shadow-sm'
    : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/10';
  return `${shape} transition-colors ${state}`;
}

function select(value: T) {
  if (value === props.modelValue) return;
  emit('update:modelValue', value);
}
</script>

<template>
  <div
    role="radiogroup"
    :aria-label="ariaLabel"
    :class="containerClass"
    :style="containerStyle"
  >
    <button
      v-for="opt in options"
      :key="String(opt.value)"
      type="button"
      role="radio"
      :aria-checked="opt.value === modelValue"
      :class="buttonClass(opt.value === modelValue)"
      v-bind="opt.dataAttr ? { [`data-${opt.dataAttr.key}`]: opt.dataAttr.value } : {}"
      @click="select(opt.value)"
    >
      {{ opt.label }}
    </button>
  </div>
</template>
