<script setup lang="ts">
import { computed } from 'vue';
import { BarChart3, Clock, Users, MessageSquare, ChevronRight, Music, GalleryVerticalEnd, GalleryVertical } from 'lucide-vue-next';
import type { ManuscriptStats } from '../../core/types';

const props = defineProps<{
  stats: ManuscriptStats | null;
}>();

function formatRuntime(minutes: number): string {
  if (minutes < 1) return '< 1 min';
  const h = Math.floor(minutes / 60);
  const m = Math.round(minutes % 60);
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
  return Math.max(1, topCharacters.value[0]?.dialogueWords ?? 1);
});
</script>

<template>
  <div class="flex flex-1 flex-col overflow-hidden">
    <div
      v-if="!stats"
      class="flex flex-1 flex-col items-center justify-center gap-2 text-text-muted"
    >
      <div class="flex h-10 w-10 items-center justify-center rounded-full bg-black/5 text-text-muted dark:bg-white/5">
        <BarChart3 class="h-5 w-5" />
      </div>
      <p class="text-sm font-medium text-text-main">No stats available</p>
      <p class="text-xs">Start writing to see manuscript statistics.</p>
    </div>

    <div v-else class="flex-1 overflow-y-auto">
      <div class="grid grid-cols-2 gap-2 p-3 sm:grid-cols-4">
        <div class="rounded-lg border border-border bg-black/[0.03] p-2.5 dark:bg-white/[0.03]">
          <div class="flex items-center gap-1.5 text-text-muted">
            <Clock class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Runtime</span>
          </div>
          <p class="mt-1 text-sm font-bold text-text-main">{{ formatRuntime(stats.runtime.minutes) }}</p>
        </div>
        <div class="rounded-lg border border-border bg-black/[0.03] p-2.5 dark:bg-white/[0.03]">
          <div class="flex items-center gap-1.5 text-text-muted">
            <MessageSquare class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Words</span>
          </div>
          <p class="mt-1 text-sm font-bold text-text-main">{{ formatNumber(stats.totalWords) }}</p>
          <p class="text-[10px] text-text-muted">{{ formatNumber(stats.dialogueWords) }} dialogue</p>
        </div>
        <div class="rounded-lg border border-border bg-black/[0.03] p-2.5 dark:bg-white/[0.03]">
          <div class="flex items-center gap-1.5 text-text-muted">
            <GalleryVerticalEnd class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Structure</span>
          </div>
          <p class="mt-1 text-sm font-bold text-text-main">{{ stats.acts }} act{{ stats.acts === 1 ? '' : 's' }}</p>
          <p class="text-[10px] text-text-muted">{{ stats.scenes }} scene{{ stats.scenes === 1 ? '' : 's' }}</p>
        </div>
        <div class="rounded-lg border border-border bg-black/[0.03] p-2.5 dark:bg-white/[0.03]">
          <div class="flex items-center gap-1.5 text-text-muted">
            <ChevronRight class="h-3 w-3" />
            <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Directions</span>
          </div>
          <p class="mt-1 text-sm font-bold text-text-main">{{ formatNumber(stats.stageDirections) }}</p>
          <p class="text-[10px] text-text-muted">{{ formatNumber(stats.stageDirectionWords) }} words</p>
        </div>
      </div>

      <div v-if="stats.songs > 0" class="px-3 pb-2">
        <div class="inline-flex items-center gap-1.5 rounded-md border border-border bg-black/[0.03] px-2.5 py-1.5 dark:bg-white/[0.03]">
          <Music class="h-3 w-3 text-text-muted" />
          <span class="text-xs font-medium text-text-main">{{ stats.songs }} song{{ stats.songs === 1 ? '' : 's' }}</span>
        </div>
      </div>

      <div v-if="topCharacters.length > 0" class="border-t border-border px-3 py-2">
        <div class="flex items-center gap-1.5 pb-2 text-text-muted">
          <Users class="h-3 w-3" />
          <span class="text-[10px] font-bold uppercase tracking-[0.15em]">Characters ({{ stats.characters.length }})</span>
        </div>
        <div class="flex flex-col gap-1">
          <div
            v-for="char in topCharacters"
            :key="char.name"
            class="group flex items-center gap-2 rounded px-1.5 py-1"
          >
            <span class="w-28 shrink-0 truncate text-xs font-medium text-text-main" :title="char.name">{{ char.name }}</span>
            <div class="relative flex-1 h-3 rounded-sm bg-black/5 dark:bg-white/5 overflow-hidden">
              <div
                class="absolute inset-y-0 left-0 rounded-sm bg-brass-500/40"
                :style="{ width: `${Math.max(2, (char.dialogueWords / maxCharacterWords) * 100)}%` }"
              ></div>
            </div>
            <span class="shrink-0 text-[10px] tabular-nums text-text-muted">{{ char.lines }}L / {{ formatNumber(char.dialogueWords) }}W</span>
          </div>
        </div>
      </div>

      <div class="border-t border-border px-3 py-2">
        <p class="text-[10px] text-text-muted">
          Runtime estimate based on {{ stats.runtime.wordsPerMinute }} WPM ({{ stats.runtime.preset }}) with {{ Math.round(stats.runtime.pauseFactor * 100) }}% pause factor.
        </p>
      </div>
    </div>
  </div>
</template>
