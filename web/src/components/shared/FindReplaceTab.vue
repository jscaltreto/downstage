<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { ChevronUp, ChevronDown, Replace, ReplaceAll, CaseSensitive, WholeWord, Regex, X } from 'lucide-vue-next';
import type { SearchMatch, SearchOptions } from '../../core/search';

const props = defineProps<{
  active: boolean;
  matches: SearchMatch[];
  currentIndex: number;
  error: string | null;
  initialQuery?: string;
  focusReplace?: boolean;
  focusNonce?: number;
}>();

const emit = defineEmits<{
  (e: 'search', opts: SearchOptions): void;
  (e: 'next'): void;
  (e: 'prev'): void;
  (e: 'replace', replacement: string): void;
  (e: 'replace-all', replacement: string): void;
  (e: 'jump', index: number): void;
}>();

const query = ref('');
const replacement = ref('');
const caseSensitive = ref(false);
const wholeWord = ref(false);
const regex = ref(false);
const findInput = ref<HTMLInputElement | null>(null);
const replaceInput = ref<HTMLInputElement | null>(null);

let debounceTimer: number | null = null;

function focusField(mode: 'find' | 'replace' = 'find') {
  const el = mode === 'replace' ? replaceInput.value : findInput.value;
  el?.focus();
  el?.select();
}

function fireSearch() {
  emit('search', {
    query: query.value,
    caseSensitive: caseSensitive.value,
    wholeWord: wholeWord.value,
    regex: regex.value,
  });
}

function queueSearch() {
  if (debounceTimer) window.clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(() => {
    debounceTimer = null;
    fireSearch();
  }, 80);
}

watch(query, queueSearch);
watch([caseSensitive, wholeWord, regex], () => fireSearch());

watch(
  () => props.active,
  async (isActive) => {
    if (!isActive) return;
    if (props.initialQuery && props.initialQuery !== query.value) {
      query.value = props.initialQuery;
    } else if (query.value) {
      fireSearch();
    }
    await nextTick();
    focusField(props.focusReplace ? 'replace' : 'find');
  },
  { immediate: true },
);

watch(
  () => props.initialQuery,
  (next) => {
    const value = next ?? '';
    if (value !== query.value) {
      query.value = value;
    }
  },
);

watch(
  () => props.focusReplace,
  async (next) => {
    if (!props.active) return;
    await nextTick();
    focusField(next ? 'replace' : 'find');
  },
);

watch(
  () => props.focusNonce,
  async () => {
    await nextTick();
    if (!props.active) return;
    focusField(props.focusReplace ? 'replace' : 'find');
  },
);

const counterText = computed(() => {
  if (props.error) return props.error;
  if (!query.value) return '';
  if (props.matches.length === 0) return 'No results';
  const shown = props.currentIndex >= 0 ? props.currentIndex + 1 : 0;
  return `${shown} of ${props.matches.length}`;
});

function interceptFindShortcut(e: KeyboardEvent): boolean {
  if (!(e.ctrlKey || e.metaKey) || e.shiftKey) return false;
  if (e.key === 'f') {
    e.preventDefault();
    focusField('find');
    return true;
  }
  if (e.key === 'h') {
    e.preventDefault();
    focusField('replace');
    return true;
  }
  return false;
}

function onFindKeydown(e: KeyboardEvent) {
  if (interceptFindShortcut(e)) return;
  if (e.key === 'Enter') {
    e.preventDefault();
    if (e.altKey) {
      emit('replace-all', replacement.value);
    } else if (e.shiftKey) {
      emit('prev');
    } else {
      emit('next');
    }
  }
}

function onReplaceKeydown(e: KeyboardEvent) {
  if (interceptFindShortcut(e)) return;
  if (e.key === 'Enter') {
    e.preventDefault();
    if (e.altKey) {
      emit('replace-all', replacement.value);
    } else {
      emit('replace', replacement.value);
    }
  }
}

function clearFind() {
  query.value = '';
  focusField('find');
}
</script>

<template>
  <div class="flex h-full flex-col overflow-hidden">
    <div class="flex flex-col gap-2 border-b border-border bg-[var(--color-toolbar-bg)] px-4 py-3">
      <div class="flex items-center gap-2">
        <div class="relative flex flex-1 items-center gap-1 rounded-md border border-border bg-[var(--color-page-bg)] px-2 focus-within:border-brass-500">
          <input
            ref="findInput"
            v-model="query"
            type="text"
            spellcheck="false"
            placeholder="Find"
            aria-label="Find"
            class="flex-1 bg-transparent py-1.5 text-sm text-text-main outline-none placeholder:text-text-muted"
            @keydown="onFindKeydown"
          />
          <button
            v-if="query"
            type="button"
            title="Clear find"
            aria-label="Clear find"
            class="flex h-6 w-6 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 dark:hover:bg-white/10"
            @click="clearFind"
          >
            <X class="h-3.5 w-3.5" />
          </button>
          <div class="flex items-center gap-0.5">
            <button
              type="button"
              :title="`Match Case${caseSensitive ? ' (on)' : ''}`"
              :aria-pressed="caseSensitive"
              class="flex h-6 w-6 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 dark:hover:bg-white/10"
              :class="{ 'bg-brass-500/25 text-accent': caseSensitive }"
              @click="caseSensitive = !caseSensitive"
            >
              <CaseSensitive class="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              :title="`Match Whole Word${wholeWord ? ' (on)' : ''}`"
              :aria-pressed="wholeWord"
              class="flex h-6 w-6 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 dark:hover:bg-white/10"
              :class="{ 'bg-brass-500/25 text-accent': wholeWord }"
              @click="wholeWord = !wholeWord"
            >
              <WholeWord class="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              :title="`Use Regular Expression${regex ? ' (on)' : ''}`"
              :aria-pressed="regex"
              class="flex h-6 w-6 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 dark:hover:bg-white/10"
              :class="{ 'bg-brass-500/25 text-accent': regex }"
              @click="regex = !regex"
            >
              <Regex class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
        <div
          class="min-w-[5.5rem] text-right text-[11px] font-medium tabular-nums"
          :class="error ? 'text-red-500' : 'text-text-muted'"
        >
          {{ counterText }}
        </div>
        <div class="flex items-center gap-0.5">
          <button
            type="button"
            title="Previous match (Shift+Enter)"
            aria-label="Previous match"
            class="flex h-7 w-7 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 disabled:opacity-40 dark:hover:bg-white/10"
            :disabled="matches.length === 0"
            @click="emit('prev')"
          >
            <ChevronUp class="h-4 w-4" />
          </button>
          <button
            type="button"
            title="Next match (Enter)"
            aria-label="Next match"
            class="flex h-7 w-7 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 disabled:opacity-40 dark:hover:bg-white/10"
            :disabled="matches.length === 0"
            @click="emit('next')"
          >
            <ChevronDown class="h-4 w-4" />
          </button>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <div class="relative flex flex-1 items-center rounded-md border border-border bg-[var(--color-page-bg)] px-2 focus-within:border-brass-500">
          <input
            ref="replaceInput"
            v-model="replacement"
            type="text"
            spellcheck="false"
            placeholder="Replace"
            aria-label="Replace with"
            class="flex-1 bg-transparent py-1.5 text-sm text-text-main outline-none placeholder:text-text-muted"
            @keydown="onReplaceKeydown"
          />
          <button
            v-if="replacement"
            type="button"
            title="Clear replace"
            aria-label="Clear replace"
            class="flex h-6 w-6 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 dark:hover:bg-white/10"
            @click="replacement = ''"
          >
            <X class="h-3.5 w-3.5" />
          </button>
        </div>
        <div class="min-w-[5.5rem]"></div>
        <div class="flex items-center gap-0.5">
          <button
            type="button"
            title="Replace (Enter in replace field)"
            aria-label="Replace current match"
            class="flex h-7 w-7 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 disabled:opacity-40 dark:hover:bg-white/10"
            :disabled="matches.length === 0 || currentIndex < 0"
            @click="emit('replace', replacement)"
          >
            <Replace class="h-4 w-4" />
          </button>
          <button
            type="button"
            title="Replace All (Alt+Enter)"
            aria-label="Replace all matches"
            class="flex h-7 w-7 items-center justify-center rounded text-text-muted transition-colors hover:bg-black/10 disabled:opacity-40 dark:hover:bg-white/10"
            :disabled="matches.length === 0"
            @click="emit('replace-all', replacement)"
          >
            <ReplaceAll class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>

    <div v-if="matches.length === 0" class="flex flex-1 items-center justify-center px-6 text-center text-xs text-text-muted">
      <template v-if="error">
        <span class="text-red-500">{{ error }}</span>
      </template>
      <template v-else-if="query">
        No matches.
      </template>
      <template v-else>
        Type a query to find or replace text.
      </template>
    </div>

    <ul v-else class="flex-1 divide-y divide-border overflow-y-auto">
      <li
        v-for="(m, index) in matches"
        :key="`${m.from}-${index}`"
        class="group cursor-pointer px-4 py-1.5 transition-colors hover:bg-black/5 dark:hover:bg-white/5"
        :class="{ 'bg-brass-500/10': index === currentIndex }"
        @click="emit('jump', index)"
      >
        <div class="flex items-start gap-3">
          <span class="shrink-0 font-mono text-[11px] text-text-muted tabular-nums">{{ m.line }}:{{ m.col }}</span>
          <span class="flex-1 truncate font-mono text-xs leading-snug text-text-main">{{ m.lineText }}</span>
        </div>
      </li>
    </ul>
  </div>
</template>
