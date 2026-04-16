<script setup lang="ts">
import { computed } from 'vue';
import { BarChart3, ChevronRight, Clock, GalleryVerticalEnd, MessageSquare, Music, RefreshCw, Users } from 'lucide-vue-next';
import type { ManuscriptStats } from '../../core/types';

const props = defineProps<{
  stats: ManuscriptStats | null;
  loading?: boolean;
}>();

function formatRuntime(minutes: number): string {
  if (minutes < 1) return '< 1 min';
  const total = Math.round(minutes);
  const h = Math.floor(total / 60);
  const m = total % 60;
  if (h === 0) return `${m} min`;
  if (m === 0) return `${h} hr`;
  return `${h} hr ${m} min`;
}

function formatNumber(n: number): string {
  return n.toLocaleString();
}

const topCharacters = computed(() => {
  if (!props.stats) return [];
  return props.stats.characters.slice(0, 20);
});

const maxCharacterWords = computed(() => {
  if (topCharacters.value.length === 0) return 1;
  return Math.max(1, ...topCharacters.value.map(c => c.dialogueWords));
});
</script>

<template>
  <div class="relative flex flex-1 flex-col overflow-hidden bg-[var(--color-page-surface)]">
    <div
      v-if="loading && !stats"
      class="relative z-10 flex flex-1 flex-col items-center justify-center gap-2 px-4 text-text-muted"
    >
      <div class="flex h-11 w-11 items-center justify-center rounded-full border border-border/70 bg-white/70 text-accent shadow-sm backdrop-blur dark:bg-black/20">
        <RefreshCw class="h-5 w-5 animate-spin" />
      </div>
      <p class="text-sm font-medium text-text-main">Calculating stats</p>
      <p class="text-xs">This usually takes a moment after you pause typing.</p>
    </div>

    <div
      v-else-if="!stats"
      class="relative z-10 flex flex-1 flex-col items-center justify-center gap-2 px-4 text-text-muted"
    >
      <div class="flex h-11 w-11 items-center justify-center rounded-full border border-border/70 bg-white/70 text-accent shadow-sm backdrop-blur dark:bg-black/20">
        <BarChart3 class="h-5 w-5" />
      </div>
      <p class="text-sm font-medium text-text-main">Stats unavailable</p>
      <p class="text-xs">Try again after making an edit.</p>
    </div>

    <div v-else class="relative z-10 flex-1 overflow-y-auto">
      <div class="grid grid-cols-2 gap-2 px-3 pb-3 pt-3 sm:grid-cols-4">
        <div class="rounded-xl border border-border bg-white/70 p-2.5 shadow-sm backdrop-blur dark:bg-black/20">
          <div class="flex items-center gap-1.5 text-text-muted">
            <Clock class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Est. Runtime<sup class="text-text-muted">&dagger;</sup></span>
          </div>
          <p class="mt-1 text-base font-bold text-text-main">{{ formatRuntime(stats.runtime.minutes) }}</p>
        </div>
        <div class="rounded-xl border border-border bg-white/70 p-2.5 shadow-sm backdrop-blur dark:bg-black/20">
          <div class="flex items-center gap-1.5 text-text-muted">
            <MessageSquare class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Words</span>
          </div>
          <p class="mt-1 text-base font-bold text-text-main">{{ formatNumber(stats.totalWords) }}</p>
          <p class="text-[10px] text-text-muted">{{ formatNumber(stats.dialogueWords) }} dialogue</p>
        </div>
        <div class="rounded-xl border border-border bg-white/70 p-2.5 shadow-sm backdrop-blur dark:bg-black/20">
          <div class="flex items-center gap-1.5 text-text-muted">
            <GalleryVerticalEnd class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Structure</span>
          </div>
          <p class="mt-1 text-base font-bold text-text-main">{{ stats.acts }} act{{ stats.acts === 1 ? '' : 's' }}</p>
          <p class="text-[10px] text-text-muted">{{ stats.scenes }} scene{{ stats.scenes === 1 ? '' : 's' }}</p>
        </div>
        <div class="rounded-xl border border-border bg-white/70 p-2.5 shadow-sm backdrop-blur dark:bg-black/20">
          <div class="flex items-center gap-1.5 text-text-muted">
            <ChevronRight class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Stage Directions</span>
          </div>
          <p class="mt-1 text-base font-bold text-text-main">{{ formatNumber(stats.stageDirections) }}</p>
          <p class="text-[10px] text-text-muted">{{ formatNumber(stats.stageDirectionWords) }} words</p>
        </div>
      </div>

      <div v-if="stats.songs > 0" class="px-3 pb-2">
        <div class="inline-flex items-center gap-1.5 rounded-full border border-brass-500/20 bg-brass-500/10 px-2.5 py-1 text-[11px] font-medium text-text-main shadow-sm">
          <Music class="h-3 w-3 text-brass-700 dark:text-brass-300" />
          <span>{{ stats.songs }} song{{ stats.songs === 1 ? '' : 's' }}</span>
        </div>
      </div>

      <div v-if="topCharacters.length > 0" class="border-t border-border/80 px-3 py-2">
        <div class="flex items-center gap-1.5 pb-2 text-text-muted">
          <Users class="h-3 w-3" />
          <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Characters ({{ stats.characters.length }})</span>
        </div>
        <div class="flex flex-col gap-1">
          <div
            v-for="char in topCharacters"
            :key="char.name"
            class="group flex items-center gap-2 rounded-lg px-1.5 py-1 transition-colors hover:bg-black/[0.03] dark:hover:bg-white/[0.04]"
          >
            <span class="w-28 shrink-0 truncate text-xs font-medium text-text-main" :title="char.name">{{ char.name }}</span>
            <div class="relative h-3 flex-1 overflow-hidden rounded-full bg-black/5 dark:bg-white/5">
              <div
                class="absolute inset-y-0 left-0 rounded-full bg-brass-500/40"
                :style="{ width: `${Math.max(2, (char.dialogueWords / maxCharacterWords) * 100)}%` }"
              ></div>
            </div>
            <span class="shrink-0 text-[10px] tabular-nums text-text-muted">{{ char.lines }}L / {{ formatNumber(char.dialogueWords) }}W</span>
          </div>
        </div>
      </div>

      <div class="border-t border-border/80 px-3 py-2">
        <p class="rounded-lg bg-black/[0.03] px-2 py-1.5 text-[10px] leading-relaxed text-text-muted dark:bg-white/[0.04]">
          &dagger; Runtime estimate based on {{ stats.runtime.wordsPerMinute }} WPM ({{ stats.runtime.preset }}) and about {{ Math.round(stats.runtime.pauseFactor * 100) }}% extra time for pauses and beats.
        </p>
      </div>
    </div>
  </div>
</template>
