<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { CheckCircle, Info, AlertCircle, X } from 'lucide-vue-next';

export type ToastType = 'success' | 'info' | 'error';

interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

const toasts = ref<Toast[]>([]);
let nextId = 0;

function addToast(message: string, type: ToastType = 'info', duration = 3000) {
  const id = nextId++;
  toasts.value.push({ id, message, type });
  
  if (duration > 0) {
    setTimeout(() => {
      removeToast(id);
    }, duration);
  }
}

function removeToast(id: number) {
  toasts.value = toasts.value.filter(t => t.id !== id);
}

defineExpose({ addToast });
</script>

<template>
  <div class="fixed bottom-6 right-6 z-50 flex flex-col gap-3 pointer-events-none">
    <TransitionGroup 
      name="toast"
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="transform translate-y-4 opacity-0"
      enter-to-class="transform translate-y-0 opacity-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div 
        v-for="toast in toasts" 
        :key="toast.id"
        class="pointer-events-auto flex items-center gap-3 px-4 py-3 rounded-lg shadow-2xl border border-border min-w-[280px] max-w-md bg-[var(--color-page-surface)] text-text-main"
      >
        <div :class="[
          'p-1.5 rounded-full shrink-0',
          toast.type === 'success' ? 'bg-green-500/10 text-green-500' : 
          toast.type === 'error' ? 'bg-red-500/10 text-red-500' : 
          'bg-brass-500/10 text-brass-500'
        ]">
          <CheckCircle v-if="toast.type === 'success'" class="w-4 h-4" />
          <AlertCircle v-else-if="toast.type === 'error'" class="w-4 h-4" />
          <Info v-else class="w-4 h-4" />
        </div>

        <p class="flex-1 text-sm font-medium leading-snug">{{ toast.message }}</p>

        <button 
          @click="removeToast(toast.id)"
          class="text-text-muted hover:text-text-main p-1 rounded-md transition-colors"
        >
          <X class="w-4 h-4" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-move {
  transition: transform 0.4s ease;
}
</style>
