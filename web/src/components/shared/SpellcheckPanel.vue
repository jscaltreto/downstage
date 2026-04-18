<script setup lang="ts">
import { computed, ref } from 'vue';
import { Trash2 } from 'lucide-vue-next';

// Reusable spellcheck control surface. Web hosts this inside the
// Editor's SpellCheck modal (click the toolbar button → modal opens
// with this panel inside). Desktop hosts this inside Settings >
// Spellcheck. Same controls, different chrome. Kept body-only (no
// modal/page wrapper) so each host chooses its own frame.

const props = defineProps<{
  enabled: boolean;
  allowlist: string[];
  addWord: (word: string) => Promise<boolean>;
  removeWord: (word: string) => Promise<boolean>;
}>();

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
  <div class="flex flex-col gap-5 py-1">
    <label class="flex items-center justify-between gap-4 rounded-lg border border-border bg-black/5 px-4 py-3 dark:bg-white/5">
      <p class="text-sm font-bold text-text-main">Enable Spell Check</p>
      <button
        type="button"
        role="switch"
        :aria-checked="enabled"
        class="relative inline-flex h-7 w-12 shrink-0 items-center rounded-full border transition-colors"
        :class="enabled ? 'border-brass-500 bg-brass-500/80' : 'border-border bg-black/10 dark:bg-white/10'"
        @click="enabled = !enabled"
      >
        <span
          class="inline-block h-5 w-5 rounded-full bg-white shadow transition-transform"
          :class="enabled ? 'translate-x-6' : 'translate-x-1'"
        ></span>
      </button>
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
