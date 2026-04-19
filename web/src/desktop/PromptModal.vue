<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import BaseModal from '../components/shared/BaseModal.vue';

// Styled replacement for `window.prompt`. Used by the New Folder flow
// (library tree + palette) so the Wails app doesn't surface an unstyled
// native prompt.

const props = defineProps<{
  open: boolean;
  title: string;
  label: string;
  placeholder?: string;
  initialValue?: string;
  submitLabel?: string;
  error?: string | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'submit', value: string): void;
}>();

const value = ref('');
const inputRef = ref<HTMLInputElement | null>(null);

watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return;
    value.value = props.initialValue ?? '';
    await nextTick();
    inputRef.value?.focus();
    inputRef.value?.select();
  },
  { immediate: true },
);

function onSubmit() {
  const trimmed = value.value.trim();
  if (!trimmed) return;
  emit('submit', trimmed);
}
</script>

<template>
  <BaseModal :open="open" :title="title" @close="emit('close')">
    <form class="flex flex-col gap-4 py-1" @submit.prevent="onSubmit">
      <label class="flex flex-col gap-1.5">
        <span class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted">{{ label }}</span>
        <input
          ref="inputRef"
          v-model="value"
          type="text"
          :placeholder="placeholder ?? ''"
          class="rounded-md border border-border bg-[var(--color-page-surface)] px-3 py-2 text-sm text-text-main outline-none transition-colors focus:border-brass-500/60"
        />
      </label>
      <p v-if="error" class="text-xs text-red-500" role="alert">{{ error }}</p>
      <div class="flex justify-end gap-2 pt-1">
        <button
          type="button"
          class="rounded-md border border-border px-3 py-1.5 text-sm font-bold text-text-main transition-colors hover:bg-black/5 dark:hover:bg-white/5"
          @click="emit('close')"
        >
          Cancel
        </button>
        <button
          type="submit"
          class="rounded-md bg-brass-500 px-3 py-1.5 text-sm font-bold text-ember-950 transition-colors hover:bg-brass-400 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="value.trim().length === 0"
        >
          {{ submitLabel ?? 'Create' }}
        </button>
      </div>
    </form>
  </BaseModal>
</template>
