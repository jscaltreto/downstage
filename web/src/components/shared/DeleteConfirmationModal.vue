<script setup lang="ts">
import BaseModal from './BaseModal.vue';
import { AlertTriangle } from 'lucide-vue-next';

defineProps<{
  open: boolean;
  itemName: string;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'confirm'): void;
}>();
</script>

<template>
  <BaseModal 
    :open="open" 
    title="Confirm Deletion" 
    @close="emit('close')"
  >
    <div class="flex flex-col items-center text-center py-4">
      <div class="w-16 h-16 rounded-full bg-red-500/10 flex items-center justify-center mb-4">
        <AlertTriangle class="w-8 h-8 text-red-500" />
      </div>
      <p class="text-text-main font-medium mb-2">Are you sure you want to delete this?</p>
      <p class="text-sm text-text-muted mb-8 leading-relaxed">
        You are about to permanently delete <strong class="text-text-main">"{{ itemName }}"</strong>. This action cannot be undone.
      </p>
      
      <div class="flex gap-3 w-full">
        <button 
          @click="emit('close')"
          class="flex-1 px-4 py-2.5 rounded-lg border border-border text-sm font-bold hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
        >
          Cancel
        </button>
        <button 
          @click="emit('confirm')"
          class="flex-1 px-4 py-2.5 rounded-lg bg-red-600 text-white text-sm font-bold hover:bg-red-700 transition-colors shadow-lg"
        >
          Delete
        </button>
      </div>
    </div>
  </BaseModal>
</template>
