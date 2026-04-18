<script setup lang="ts">
import { computed } from 'vue';
import type { Store } from '../../core/store';
import type { Workspace } from '../workspace';
import SpellcheckPanel from '../../components/shared/SpellcheckPanel.vue';

const props = defineProps<{
  store: Store;
  workspace: Workspace;
}>();

const spellcheckEnabled = computed<boolean>({
  get: () => !props.store.state.spellcheckDisabled,
  set: (v) => { props.store.state.spellcheckDisabled = !v; },
});

// The panel takes allowlist add/remove callbacks; proxy through the
// Workspace so writes go via the Go-backed allowlist API and keep the
// reactive state in sync.
async function addWord(word: string) {
  return props.workspace.addAllowlistWord(word);
}

async function removeWord(word: string) {
  return props.workspace.removeAllowlistWord(word);
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-bold text-text-main">Spellcheck</h3>
    <SpellcheckPanel
      v-model:enabled="spellcheckEnabled"
      :allowlist="workspace.state.spellAllowlist"
      :add-word="addWord"
      :remove-word="removeWord"
    />
  </div>
</template>
