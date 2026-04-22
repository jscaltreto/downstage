<script setup lang="ts">
import { ref, watch } from 'vue';
import BaseModal from '../components/shared/BaseModal.vue';
import type { Store } from '../core/store';
import type { Workspace } from './workspace';
import type { DesktopCapabilities } from './types';
import AppearanceSettings from './settings/AppearanceSettings.vue';
import SpellcheckSettings from './settings/SpellcheckSettings.vue';
import LibrarySettings from './settings/LibrarySettings.vue';
import ExportSettings from './settings/ExportSettings.vue';

// Desktop Settings dialog. Four tabs: Library, Appearance, Export,
// Spellcheck. Library is first because it's the only place to change the
// library location now that the File menu no longer has "Open Folder…".
// Transient UI toggles (show preview, show sidebar) don't live here;
// they have their own affordances in the main UI.
//
// Git / Advanced get tabs when they have real controls; placeholder tabs
// rot faster than they help.

type SettingsTab = 'library' | 'appearance' | 'export' | 'spellcheck';

const props = defineProps<{
  open: boolean;
  tab: SettingsTab;
  store: Store;
  workspace: Workspace;
  env: DesktopCapabilities;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  // Bubbled by LibrarySettings when the user clicks "Change…". The
  // parent runs the full library-switch flow (flushSave, switch,
  // re-select first file, toast) so Settings stays presentational.
  (e: 'change-library'): void;
}>();

const currentTab = ref<SettingsTab>(props.tab);

watch(() => props.tab, (t) => { currentTab.value = t; });
watch(() => props.open, (isOpen) => {
  if (isOpen) currentTab.value = props.tab;
});

const tabs: Array<{ id: SettingsTab; label: string }> = [
  { id: 'library', label: 'Library' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'export', label: 'Export' },
  { id: 'spellcheck', label: 'Spellcheck' },
];
</script>

<template>
  <BaseModal
    :open="open"
    title="Settings"
    size="xl"
    padding="none"
    @close="$emit('close')"
  >
    <!-- Desktop-native settings layout. Wider modal (set by BaseModal
         size="xl") so the sidebar + content columns both breathe. The
         sidebar is deliberately tight: small text, compact rows, no
         pill selection — matches the density of native preference
         panes (Slack, VS Code). Fixed height keeps the pane from
         jumping as tabs switch. -->
    <div class="flex w-full h-[480px] border-t border-border">
      <nav class="w-48 shrink-0 flex flex-col gap-0.5 py-3 px-2 border-r border-border bg-black/5 dark:bg-white/5">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          class="text-left px-2.5 py-1.5 rounded text-xs transition-colors"
          :class="currentTab === t.id
            ? 'bg-black/10 dark:bg-white/10 text-text-main font-semibold'
            : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/5'"
          @click="currentTab = t.id"
        >
          {{ t.label }}
        </button>
      </nav>
      <div class="flex-1 min-w-0 overflow-y-auto custom-scrollbar px-6 py-5">
        <LibrarySettings
          v-if="currentTab === 'library'"
          :workspace="workspace"
          :env="env"
          @change-library="emit('change-library')"
        />
        <AppearanceSettings v-else-if="currentTab === 'appearance'" :store="store" />
        <ExportSettings v-else-if="currentTab === 'export'" :env="env" />
        <SpellcheckSettings v-else-if="currentTab === 'spellcheck'" :store="store" :workspace="workspace" />
      </div>
    </div>
  </BaseModal>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar { width: 6px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.1); border-radius: 10px; }
.dark .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); }
</style>
