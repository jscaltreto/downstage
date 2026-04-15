<script setup lang="ts">
import { computed } from 'vue';
import { GalleryVerticalEnd, GalleryVertical, Users, Music, Drama, ListTree } from 'lucide-vue-next';
import type { DocumentSymbol } from '../../core/types';
import { SymbolKind } from '../../core/types';

interface FlatSymbol {
  symbol: DocumentSymbol;
  depth: number;
  key: string;
}

const props = defineProps<{
  symbols: DocumentSymbol[];
}>();

const emit = defineEmits<{
  (e: 'jump', symbol: DocumentSymbol): void;
}>();

function isSongSymbol(symbol: DocumentSymbol): boolean {
  return (
    symbol.kind === SymbolKind.Function &&
    (symbol.name.endsWith(' (song)') || symbol.name === 'Song')
  );
}

function isRenderableSymbol(symbol: DocumentSymbol): boolean {
  return symbol.kind !== SymbolKind.Function || isSongSymbol(symbol);
}

const flat = computed<FlatSymbol[]>(() => {
  const out: FlatSymbol[] = [];
  const walk = (nodes: DocumentSymbol[], depth: number, prefix: string) => {
    nodes.forEach((symbol, index) => {
      if (!isRenderableSymbol(symbol)) return;
      const key = `${prefix}${index}:${symbol.range.start.line}:${symbol.range.start.character}`;
      out.push({ symbol, depth, key });
      if (symbol.children && symbol.children.length > 0) {
        walk(symbol.children, depth + 1, `${key}/`);
      }
    });
  };
  walk(props.symbols, 0, '');
  return out;
});

function iconFor(kind: number) {
  switch (kind) {
    case SymbolKind.Namespace:
      return GalleryVerticalEnd;
    case SymbolKind.Class:
      return GalleryVertical;
    case SymbolKind.Struct:
      return Users;
    case SymbolKind.File:
      return Drama;
    default:
      return Drama;
  }
}

function iconColor(kind: number) {
  switch (kind) {
    case SymbolKind.Namespace:
      return 'text-brass-500';
    case SymbolKind.Class:
      return 'text-accent';
    case SymbolKind.Struct:
      return 'text-purple-600 dark:text-purple-300';
    default:
      return 'text-text-muted';
  }
}

function resolveIcon(symbol: DocumentSymbol) {
  if (isSongSymbol(symbol)) {
    return Music;
  }
  return iconFor(symbol.kind);
}

function paddingFor(depth: number): string {
  const pl = 16 + depth * 16;
  return `${pl}px`;
}
</script>

<template>
  <div class="flex flex-1 flex-col overflow-hidden">
    <div
      v-if="flat.length === 0"
      class="flex flex-1 flex-col items-center justify-center gap-2 text-text-muted"
    >
      <div class="flex h-10 w-10 items-center justify-center rounded-full bg-black/5 text-text-muted dark:bg-white/5">
        <ListTree class="h-5 w-5" />
      </div>
      <p class="text-sm font-medium text-text-main">No structure yet</p>
      <p class="text-xs">Add a heading (e.g. <code class="font-mono">## ACT I</code>) to build an outline.</p>
    </div>

    <ul v-else class="flex-1 overflow-y-auto py-1">
      <li v-for="entry in flat" :key="entry.key">
        <button
          type="button"
          class="group flex w-full items-center gap-2 pr-4 py-1 text-left transition-colors hover:bg-black/5 dark:hover:bg-white/5"
          :style="{ paddingLeft: paddingFor(entry.depth) }"
          :title="`Jump to ${entry.symbol.name} (line ${entry.symbol.range.start.line + 1})`"
          @click="emit('jump', entry.symbol)"
        >
          <component
            :is="resolveIcon(entry.symbol)"
            class="shrink-0"
            :class="[
              entry.depth === 0 ? 'h-3.5 w-3.5' : 'h-3 w-3',
              iconColor(entry.symbol.kind),
            ]"
          />
          <span
            class="flex-1 truncate"
            :class="entry.depth === 0 ? 'text-sm font-medium text-text-main' : 'text-xs text-text-main'"
          >{{ entry.symbol.name }}</span>
          <span class="shrink-0 font-mono text-[10px] text-text-muted tabular-nums">{{ entry.symbol.range.start.line + 1 }}</span>
        </button>
      </li>
    </ul>
  </div>
</template>
