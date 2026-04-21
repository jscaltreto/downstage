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
    @close="$emit('close')"
  >
    <!-- Fixed outer dimensions so tab switches don't resize the modal.
         Width must stay inside BaseModal's max-w-lg (512px) minus its
         p-6 padding (48px) = 464px of usable room. 432px fits with a
         little breathing space and avoids a horizontal scrollbar. -->
    <div class="flex flex-col w-[432px] h-[420px]">
      <div class="flex gap-6 flex-1 min-h-0">
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
        <div class="flex-1 min-w-0 overflow-y-auto custom-scrollbar pr-1">
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
      <div class="flex justify-end pt-4 mt-4 border-t border-border">
        <button
          type="button"
          class="rounded-lg bg-brass-500 px-4 py-2 text-sm font-bold text-ember-950 transition-colors hover:bg-brass-400"
          @click="$emit('close')"
        >
          Done
        </button>
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
