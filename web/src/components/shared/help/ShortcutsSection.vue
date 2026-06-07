<script setup lang="ts">
import { computed } from 'vue';
import { Loader2 } from 'lucide-vue-next';
import type { HelpHost, ShortcutEntry } from '../help-sections';
import { shortcutGroupLabel, shortcutGroupOrder, sortShortcuts } from '../help-sections';

const { shortcuts, loading = false } = defineProps<{
  openLink: (url: string) => Promise<void>;
  host: HelpHost;
  shortcuts: ShortcutEntry[];
  loading?: boolean;
}>();

const grouped = computed(() => {
  const sorted = sortShortcuts([...shortcuts]);
  const groups = new Map<string, ShortcutEntry[]>();
  for (const entry of sorted) {
    const bucket = groups.get(entry.group);
    if (bucket) bucket.push(entry);
    else groups.set(entry.group, [entry]);
  }
  const orderedKeys = [
    ...shortcutGroupOrder.filter((g) => groups.has(g)),
    ...Array.from(groups.keys()).filter((g) => !shortcutGroupOrder.includes(g)),
  ];
  return orderedKeys.map((key) => ({
    key,
    label: shortcutGroupLabel[key] ?? titleCase(key),
    entries: groups.get(key) ?? [],
  }));
});

function titleCase(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
</script>

<template>
  <div class="space-y-3">
    <p class="text-xs text-text-muted">
      <template v-if="host === 'desktop'">
        These shortcuts mirror the native menu. Pressing the keys does the same thing as choosing the menu item.
      </template>
      <template v-else>
        These shortcuts work in the browser editor. The desktop app adds many more.
      </template>
    </p>

    <div
      v-if="loading"
      class="flex items-center gap-2 rounded-md bg-black/[0.03] px-3 py-3 text-xs text-text-muted dark:bg-white/[0.03]"
    >
      <Loader2 class="h-3.5 w-3.5 animate-spin" />
      Loading shortcuts…
    </div>

    <div
      v-else-if="grouped.length === 0"
      class="rounded-md border border-dashed border-border px-3 py-3 text-xs text-text-muted"
    >
      Shortcut list unavailable.
    </div>

    <template v-else>
      <section v-for="g in grouped" :key="g.key" class="space-y-1.5">
        <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">{{ g.label }}</h3>
        <div
          v-for="s in g.entries"
          :key="s.id"
          class="flex items-center justify-between gap-3 rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
        >
          <span class="text-xs text-text-main">{{ s.label }}</span>
          <kbd class="rounded border border-border bg-[var(--color-page-surface)] px-1.5 py-0.5 text-[10px] font-mono font-bold text-text-muted shadow-sm">{{ s.keys }}</kbd>
        </div>
      </section>
    </template>
  </div>
</template>
