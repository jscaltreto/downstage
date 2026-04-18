<script setup lang="ts">
import { ref, watch } from 'vue';
import BaseModal from '../components/shared/BaseModal.vue';
import type { Store } from '../core/store';
import type { Workspace } from './workspace';
import AppearanceSettings from './settings/AppearanceSettings.vue';
import SpellcheckSettings from './settings/SpellcheckSettings.vue';

// Desktop Settings dialog. Only two tabs today — Appearance and
// Spellcheck — because those are the only categories that actually
// persist user preferences. Transient UI toggles (show preview, show
// sidebar) don't live here; they have their own affordances in the
// main UI (toolbar button, floating re-expand button, menu item, and
// keyboard shortcut).
//
// Project / Export / Git / Advanced get tabs when they have real
// controls; placeholder tabs rot faster than they help.

type SettingsTab = 'appearance' | 'spellcheck';

const props = defineProps<{
  open: boolean;
  tab: SettingsTab;
  store: Store;
  workspace: Workspace;
}>();

defineEmits<{
  (e: 'close'): void;
}>();

const currentTab = ref<SettingsTab>(props.tab);

watch(() => props.tab, (t) => { currentTab.value = t; });
watch(() => props.open, (isOpen) => {
  if (isOpen) currentTab.value = props.tab;
});

const tabs: Array<{ id: SettingsTab; label: string }> = [
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
    <div class="flex gap-6 min-h-[320px]">
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
        <AppearanceSettings v-if="currentTab === 'appearance'" :store="store" />
        <SpellcheckSettings v-else-if="currentTab === 'spellcheck'" :store="store" :workspace="workspace" />
      </div>
    </div>
  </BaseModal>
</template>
