<script setup lang="ts">
import { ref, watch } from 'vue';
import BaseModal from '../components/shared/BaseModal.vue';
import type { Store } from '../core/store';
import type { Workspace } from './workspace';
import EditorSettings from './settings/EditorSettings.vue';
import AppearanceSettings from './settings/AppearanceSettings.vue';
import SpellcheckSettings from './settings/SpellcheckSettings.vue';

// Desktop Settings dialog. Three real tabs — Editor, Appearance,
// Spellcheck. The empty categories (Project / Export / Git / Advanced)
// from the vision aren't created until they have real controls —
// placeholder tabs rot faster than they help.

type SettingsTab = 'editor' | 'appearance' | 'spellcheck';

const props = defineProps<{
  open: boolean;
  tab: SettingsTab;
  store: Store;
  workspace: Workspace;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

const currentTab = ref<SettingsTab>(props.tab);

// Keep the local tab synced to the prop so "open Settings on Spellcheck"
// re-dispatches correctly after an already-open dialog.
watch(() => props.tab, (t) => { currentTab.value = t; });
watch(() => props.open, (isOpen) => {
  if (isOpen) currentTab.value = props.tab;
});

const tabs: Array<{ id: SettingsTab; label: string }> = [
  { id: 'editor', label: 'Editor' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'spellcheck', label: 'Spellcheck' },
];
</script>

<template>
  <BaseModal
    :open="open"
    title="Settings"
    @close="$emit('close')"
  >
    <div class="flex gap-6 min-h-[360px]">
      <nav class="w-36 shrink-0 flex flex-col gap-1">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          class="text-left px-3 py-2 rounded text-sm transition-colors border"
          :class="currentTab === t.id
            ? 'bg-brass-500/10 text-brass-500 font-bold border-brass-500/20'
            : 'text-text-main border-transparent hover:bg-black/5 dark:hover:bg-white/5'"
          @click="currentTab = t.id"
        >
          {{ t.label }}
        </button>
      </nav>
      <div class="flex-1 min-w-0">
        <EditorSettings v-if="currentTab === 'editor'" :store="store" />
        <AppearanceSettings v-else-if="currentTab === 'appearance'" :store="store" :workspace="workspace" />
        <SpellcheckSettings v-else-if="currentTab === 'spellcheck'" :store="store" :workspace="workspace" />
      </div>
    </div>
  </BaseModal>
</template>
