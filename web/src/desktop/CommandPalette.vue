<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import type { CommandMeta, DesktopCapabilities, ProjectFile } from './types';
import { dispatchCommand } from './dispatcher-registry';

// Command palette overlay. Consumes the Go-declared catalog (labels,
// accelerators, categories) through env.getCommands so there's no
// duplicate UI text in TS. Disabled commands are read from the
// CommandDispatcher's latest snapshot via the `disabledIds` prop.
//
// Two modes: "command" (default) shows every palette-visible command;
// "file" swaps the source list for the current project's files and
// emits `select-file` on Enter.

const props = defineProps<{
  open: boolean;
  mode: 'command' | 'file';
  env: DesktopCapabilities;
  projectFiles: ProjectFile[];
  disabledIds: string[];
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'select-file', path: string): void;
}>();

const query = ref('');
const selectedIndex = ref(0);
const inputRef = ref<HTMLInputElement | null>(null);
const commands = ref<CommandMeta[]>([]);

// Refresh the command list every time the palette opens so a Disabled
// change (pushed to Go asynchronously) is reflected in the ordering.
// Small list; cheap refetch. immediate: true handles the case where
// the palette is mounted already open.
watch(() => props.open, async (isOpen) => {
  if (!isOpen) return;
  query.value = '';
  selectedIndex.value = 0;
  if (props.mode === 'command') {
    commands.value = (await props.env.getCommands()).filter((c) => !c.paletteHidden);
  }
  await nextTick();
  inputRef.value?.focus();
}, { immediate: true });

const disabledSet = computed(() => new Set(props.disabledIds));

interface Row {
  key: string;
  label: string;
  secondary: string;
  accelerator?: string;
  disabled: boolean;
  run: () => void;
}

const rows = computed<Row[]>(() => {
  if (props.mode === 'file') {
    return props.projectFiles.map<Row>((f) => ({
      key: `file:${f.path}`,
      label: f.name,
      secondary: f.path !== f.name ? f.path : 'File',
      disabled: false,
      run: () => emit('select-file', f.path),
    }));
  }
  return commands.value.map<Row>((c) => ({
    key: c.id,
    label: c.label,
    secondary: c.category,
    accelerator: c.accelerator,
    disabled: disabledSet.value.has(c.id),
    run: () => {
      emit('close');
      void dispatchCommand(c.id);
    },
  }));
});

// Simple fuzzy filter: substring + prefix-boost. No new dependency; the
// catalog is small enough that a linear pass is fine for every keystroke.
const filtered = computed<Row[]>(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return rows.value;
  const scored: Array<{ row: Row; score: number }> = [];
  for (const row of rows.value) {
    const label = row.label.toLowerCase();
    const secondary = row.secondary.toLowerCase();
    let score = 0;
    if (label.startsWith(q)) score = 3;
    else if (label.includes(q)) score = 2;
    else if (secondary.includes(q)) score = 1;
    else continue;
    scored.push({ row, score });
  }
  scored.sort((a, b) => b.score - a.score);
  return scored.map((s) => s.row);
});

watch(filtered, () => {
  if (selectedIndex.value >= filtered.value.length) {
    selectedIndex.value = Math.max(0, filtered.value.length - 1);
  }
});

// Step through the visible rows, wrapping at the ends. Disabled rows
// are still stepped to so the user can see why — but Enter refuses to
// execute them.
function step(direction: 1 | -1) {
  if (filtered.value.length === 0) return;
  const n = filtered.value.length;
  selectedIndex.value = (selectedIndex.value + direction + n) % n;
}

function commit() {
  const row = filtered.value[selectedIndex.value];
  if (!row || row.disabled) return;
  row.run();
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') { e.preventDefault(); step(1); }
  else if (e.key === 'ArrowUp') { e.preventDefault(); step(-1); }
  else if (e.key === 'Enter') { e.preventDefault(); commit(); }
  else if (e.key === 'Escape') { e.preventDefault(); emit('close'); }
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-start justify-center pt-24 bg-black/40 backdrop-blur-sm"
    @click.self="$emit('close')"
  >
    <div class="w-[520px] max-w-[90vw] rounded-xl border border-border bg-[var(--color-page-bg)] shadow-2xl overflow-hidden">
      <div class="p-3 border-b border-border">
        <input
          ref="inputRef"
          v-model="query"
          type="text"
          :placeholder="mode === 'file' ? 'Go to file…' : 'Type a command…'"
          class="w-full bg-transparent outline-none text-text-main placeholder:text-text-muted text-sm"
          @keydown="onKeydown"
        />
      </div>
      <ul class="max-h-[360px] overflow-y-auto custom-scrollbar">
        <li
          v-for="(row, idx) in filtered"
          :key="row.key"
          class="px-4 py-2 flex items-center justify-between gap-3 cursor-pointer transition-colors"
          :class="[
            idx === selectedIndex ? 'bg-brass-500/15' : '',
            row.disabled ? 'opacity-40 cursor-not-allowed' : 'hover:bg-black/5 dark:hover:bg-white/5',
          ]"
          @mouseenter="selectedIndex = idx"
          @click="() => { if (!row.disabled) row.run(); }"
        >
          <div class="min-w-0 flex-1">
            <div class="text-sm text-text-main truncate">{{ row.label }}</div>
            <div class="text-[10px] uppercase tracking-wider text-text-muted truncate">{{ row.secondary }}</div>
          </div>
          <span
            v-if="row.accelerator"
            class="text-[10px] text-text-muted font-mono px-2 py-0.5 rounded border border-border bg-black/5 dark:bg-white/5 shrink-0"
          >{{ row.accelerator }}</span>
        </li>
        <li v-if="filtered.length === 0" class="px-4 py-6 text-center text-sm text-text-muted italic">
          No matches
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar { width: 6px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.1); border-radius: 10px; }
</style>
