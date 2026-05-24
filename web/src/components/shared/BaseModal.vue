<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { X } from 'lucide-vue-next';

// `size` controls the modal's max-width. Default 'md' (max-w-lg / 512px)
// matches the original design used by single-question dialogs (export,
// welcome, prompt). 'lg' and 'xl' are for denser surfaces — settings,
// anything that hosts a sidebar + content pane. `padding` opts a
// caller out of BaseModal's default p-6 so tightly-laid-out surfaces
// can manage their own gutters.
type Size = 'md' | 'lg' | 'xl';

const props = withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    kicker?: string;
    message?: string;
    size?: Size;
    padding?: 'default' | 'none';
  }>(),
  { size: 'md', padding: 'default' },
);

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

const containerClass = computed(() => {
  const width =
    props.size === 'xl'
      ? 'min-w-[720px] max-w-[820px]'
      : props.size === 'lg'
        ? 'min-w-[560px] max-w-[640px]'
        : 'min-w-[440px] max-w-lg';
  const pad = props.padding === 'none' ? '' : 'p-6';
  return `${width} ${pad} relative flex flex-col`;
});

// Header padding tracks the body padding so title/close-X position is
// stable across padding=none and padding=default consumers. `padding=none`
// callers still get a standardized title bar.
const headerClass = computed(() =>
  props.padding === 'none'
    ? 'px-6 pt-6 pb-3 pr-12'
    : 'mb-5 pr-8',
);

const closeButtonClass = computed(() =>
  props.padding === 'none'
    ? 'absolute top-4 right-4 text-text-muted hover:text-text-main p-1.5 transition-colors rounded-full hover:bg-black/5 dark:hover:bg-white/5'
    : 'absolute top-4 right-4 text-text-muted hover:text-text-main p-2 transition-colors rounded-full hover:bg-black/5 dark:hover:bg-white/5',
);
</script>

<template>
  <dialog 
    ref="dialogRef" 
    class="rounded-xl bg-[var(--color-page-bg)] border border-border text-text-main p-0 shadow-stage fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 m-0 backdrop:bg-black/60 backdrop:backdrop-blur-sm"
    @close="handleClose"
  >
    <div :class="containerClass">
      <div :class="headerClass">
        <p v-if="kicker" class="text-[10px] uppercase tracking-[0.2em] text-brass-500 font-bold mb-1">{{ kicker }}</p>
        <h2 class="text-xl font-serif font-bold text-text-main leading-none">{{ title }}</h2>
        <p v-if="message" class="text-xs text-text-muted mt-2 leading-relaxed">{{ message }}</p>
      </div>

      <div class="flex-1 min-h-0 overflow-y-auto custom-scrollbar">
        <slot></slot>
      </div>

      <button
        :class="closeButtonClass"
        @click="handleClose"
        aria-label="Close"
      >
        <X :class="padding === 'none' ? 'w-4 h-4' : 'w-5 h-5'" />
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
