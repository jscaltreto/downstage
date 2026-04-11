<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { X } from 'lucide-vue-next';

const props = defineProps<{
  open: boolean;
  title: string;
  kicker?: string;
  message?: string;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

const dialogRef = ref<HTMLDialogElement | null>(null);

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    dialogRef.value?.showModal();
  } else {
    dialogRef.value?.close();
  }
});

onMounted(() => {
  if (props.open) {
    dialogRef.value?.showModal();
  }
});

function handleClose() {
  emit('close');
}
</script>

<template>
  <dialog 
    ref="dialogRef" 
    class="rounded-xl bg-[var(--color-page-bg)] border border-border text-text-main p-0 shadow-stage fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 m-0 backdrop:bg-black/60 backdrop:backdrop-blur-sm"
    @close="handleClose"
  >
    <div class="p-6 min-w-[440px] max-w-lg relative flex flex-col">
      <div class="mb-5 pr-8">
        <p v-if="kicker" class="text-[10px] uppercase tracking-[0.2em] text-brass-500 font-bold mb-1">{{ kicker }}</p>
        <h2 class="text-xl font-serif font-bold text-text-main leading-none">{{ title }}</h2>
        <p v-if="message" class="text-xs text-text-muted mt-2 leading-relaxed">{{ message }}</p>
      </div>
      
      <div class="flex-1 min-h-0 overflow-y-auto custom-scrollbar">
        <slot></slot>
      </div>

      <button 
        class="absolute top-4 right-4 text-text-muted hover:text-text-main p-2 transition-colors rounded-full hover:bg-black/5 dark:hover:bg-white/5"
        @click="handleClose"
        aria-label="Close"
      >
        <X class="w-5 h-5" />
      </button>
    </div>
  </dialog>
</template>

<style scoped>
dialog::backdrop {
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
}
</style>
