<script setup lang="ts">
import BaseModal from '../components/shared/BaseModal.vue';

// Styled replacement for `window.confirm`. Used by delete / permanent-delete
// / discard flows so the Wails app doesn't surface an unstyled native dialog.

const props = withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    destructive?: boolean;
  }>(),
  { confirmLabel: 'Confirm', cancelLabel: 'Cancel', destructive: false },
);

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'confirm'): void;
}>();

function onConfirm() {
  emit('confirm');
}
</script>

<template>
  <BaseModal :open="open" :title="title" @close="emit('close')">
    <div class="flex flex-col gap-5 py-1">
      <p class="text-sm text-text-main leading-relaxed whitespace-pre-line">{{ message }}</p>
      <div class="flex justify-end gap-2 pt-1">
        <button
          type="button"
          class="rounded-md border border-border px-3 py-1.5 text-sm font-bold text-text-main transition-colors hover:bg-black/5 dark:hover:bg-white/5"
          @click="emit('close')"
        >
          {{ cancelLabel }}
        </button>
        <button
          type="button"
          :class="[
            'rounded-md px-3 py-1.5 text-sm font-bold transition-colors',
            destructive
              ? 'bg-red-600 text-white hover:bg-red-500'
              : 'bg-brass-500 text-ember-950 hover:bg-brass-400',
          ]"
          @click="onConfirm"
        >
          {{ confirmLabel }}
        </button>
      </div>
    </div>
  </BaseModal>
</template>
