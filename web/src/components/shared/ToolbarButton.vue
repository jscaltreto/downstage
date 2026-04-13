<script setup lang="ts">
import { computed, useSlots } from 'vue';
import { Comment, Text } from 'vue';

defineProps<{
  disabled?: boolean;
  active?: boolean;
  title?: string;
  transparent?: boolean;
}>();

const slots = useSlots();
const hasText = computed(() => {
  const content = slots.default?.() || [];
  return content.some((node) => {
    if (node.type === Comment) return false;
    if (node.type === Text) {
      return String(node.children ?? '').trim().length > 0;
    }
    return true;
  });
});
</script>

<template>
  <button
    type="button"
    :disabled="disabled"
    :title="title"
    :class="[
      'rounded-md border text-sm transition-all duration-150 flex items-center gap-2 justify-center shrink-0',
      'disabled:opacity-40 disabled:cursor-not-allowed shadow-sm active:scale-95',
      
      // Padding logic: smaller if icon only
      hasText ? 'px-2.5 py-1.5' : 'p-1.5',
      
      // Theme aware colors
      transparent 
        ? 'bg-transparent border-transparent text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/5 shadow-none'
        : 'bg-black/[0.08] dark:bg-white/5 border-black/10 dark:border-white/10 text-text-main hover:bg-black/[0.12] dark:hover:bg-white/10',
      
      // Active state
      active ? 'border-brass-500 bg-brass-500/10 text-brass-500 font-bold' : ''
    ]"
  >
    <slot name="icon"></slot>
    <span v-if="hasText" class="hidden md:inline"><slot></slot></span>
  </button>
</template>
