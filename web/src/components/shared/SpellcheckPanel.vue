<script setup lang="ts">
import { computed, ref } from 'vue';
import { Trash2 } from 'lucide-vue-next';
import ToggleSwitch from './ToggleSwitch.vue';

// Reusable spellcheck control surface. Web hosts this inside the
// Editor's SpellCheck modal (click the toolbar button → modal opens
// with this panel inside). Desktop hosts this inside Settings >
// Spellcheck. Same controls, different chrome. Kept body-only (no
// modal/page wrapper) so each host chooses its own frame.

const props = withDefaults(
  defineProps<{
    enabled: boolean;
    allowlist: string[];
    addWord: (word: string) => Promise<boolean>;
    removeWord: (word: string) => Promise<boolean>;
    // 'default' (web host) uses roomy web-modal styling; 'compact'
    // (desktop host) tightens padding and text to match the
    // desktop-native settings pane.
    density?: 'default' | 'compact';
  }>(),
  { density: 'default' },
);

const emit = defineEmits<{
  (e: 'update:enabled', value: boolean): void;
}>();

const enabled = computed<boolean>({
  get: () => props.enabled,
  set: (v) => emit('update:enabled', v),
});

const dictionaryWord = ref('');

async function onAddWord() {
  const added = await props.addWord(dictionaryWord.value);
  if (added) dictionaryWord.value = '';
}

async function onRemoveWord(word: string) {
  await props.removeWord(word);
}
</script>

<template>
  <div v-if="density === 'compact'" class="flex flex-col gap-4">
    <label class="flex items-center justify-between gap-3">
      <span class="text-xs font-semibold text-text-main">Enable spell check</span>
      <ToggleSwitch v-model="enabled" aria-label="Enable spell check" />
    </label>

    <div class="flex flex-col gap-1.5">
      <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-text-muted">Custom dictionary</p>
      <form class="flex gap-1.5" @submit.prevent="onAddWord">
        <input
          v-model="dictionaryWord"
          type="text"
          class="flex-1 rounded-md border border-border bg-black/5 px-2.5 py-1 text-xs text-text-main outline-none transition-colors placeholder:text-text-muted focus:border-brass-500 dark:bg-white/5"
          placeholder="Add a word"
        />
        <button
          type="submit"
          class="rounded-md bg-brass-500 px-3 py-1 text-xs font-semibold text-ember-950 transition-colors hover:bg-brass-400 disabled:opacity-50"
          :disabled="dictionaryWord.trim().length === 0"
        >
          Add
        </button>
      </form>

      <div
        v-if="allowlist.length === 0"
        class="rounded-md border border-dashed border-border bg-black/5 px-3 py-3 text-center text-xs text-text-muted dark:bg-white/5"
      >
        No custom words yet.
      </div>

      <div v-else class="flex flex-col divide-y divide-border/60 rounded-md border border-border bg-black/5 dark:bg-white/5 max-h-[180px] overflow-y-auto">
        <div
          v-for="word in allowlist"
          :key="word"
          class="flex items-center justify-between gap-2 px-2.5 py-1"
        >
          <span class="font-mono text-xs text-text-main">{{ word }}</span>
          <button
            type="button"
            class="rounded p-1 text-text-muted transition-colors hover:bg-red-500/10 hover:text-red-500"
            :title="`Remove ${word} from this script dictionary`"
            @click="onRemoveWord(word)"
          >
            <Trash2 class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="flex flex-col gap-5 py-1">
    <label class="flex items-center justify-between gap-4 rounded-lg border border-border bg-black/5 px-4 py-3 dark:bg-white/5">
      <p class="text-sm font-bold text-text-main">Enable Spell Check</p>
      <ToggleSwitch v-model="enabled" aria-label="Enable spell check" />
    </label>

    <div class="space-y-1">
      <p class="text-sm font-bold text-text-main">Script Dictionary</p>
      <p class="text-xs leading-relaxed text-text-muted">
        Add custom words for this draft. These entries do not affect any other script.
      </p>
    </div>

    <form class="flex gap-2" @submit.prevent="onAddWord">
      <input
        v-model="dictionaryWord"
        type="text"
        class="flex-1 rounded-lg border border-border bg-black/5 px-3 py-2 text-sm text-text-main outline-none transition-colors placeholder:text-text-muted focus:border-brass-500 dark:bg-white/5"
        placeholder="Add a custom word"
      />
      <button
        type="submit"
        class="rounded-lg bg-brass-500 px-4 py-2 text-sm font-bold text-ember-950 transition-colors hover:bg-brass-400 disabled:opacity-50"
        :disabled="dictionaryWord.trim().length === 0"
      >
        Add
      </button>
    </form>

    <div v-if="allowlist.length === 0" class="rounded-lg border border-dashed border-border bg-black/5 px-4 py-6 text-center text-sm text-text-muted dark:bg-white/5">
      No custom words yet.
    </div>

    <div v-else class="flex flex-col gap-2">
      <div
        v-for="word in allowlist"
        :key="word"
        class="flex items-center justify-between gap-3 rounded-lg border border-border bg-black/5 px-3 py-2 dark:bg-white/5"
      >
        <span class="font-mono text-sm text-text-main">{{ word }}</span>
        <button
          type="button"
          class="rounded-md p-2 text-text-muted transition-colors hover:bg-red-500/10 hover:text-red-500"
          :title="`Remove ${word} from this script dictionary`"
          @click="onRemoveWord(word)"
        >
          <Trash2 class="h-4 w-4" />
        </button>
      </div>
    </div>
  </div>
</template>
