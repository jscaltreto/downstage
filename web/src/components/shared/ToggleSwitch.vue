<script setup lang="ts">
// Shared pill-style slide toggle. The canonical boolean affordance in
// the app — use this instead of `<input type="checkbox">` so the visual
// language stays consistent across modals, settings tabs, and inline
// controls.

const props = defineProps<{
  modelValue: boolean;
  ariaLabel?: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
}>();

function toggle() {
  emit('update:modelValue', !props.modelValue);
}
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :aria-label="ariaLabel"
    class="relative inline-flex h-7 w-12 shrink-0 items-center rounded-full border transition-colors"
    :class="modelValue ? 'border-brass-500 bg-brass-500/80' : 'border-border bg-black/10 dark:bg-white/10'"
    @click="toggle"
  >
    <span
      class="inline-block h-5 w-5 rounded-full bg-white shadow transition-transform"
      :class="modelValue ? 'translate-x-6' : 'translate-x-1'"
    ></span>
  </button>
</template>
