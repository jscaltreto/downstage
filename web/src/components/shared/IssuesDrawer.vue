<script setup lang="ts">
import { computed } from 'vue';
import { AlertCircle, AlertTriangle, CheckCircle2, Info, X } from 'lucide-vue-next';
import type { EditorDiagnostic } from '../../core/types';
import { summarizeIssues } from '../../core/issues';

const props = defineProps<{
  diagnostics: EditorDiagnostic[];
  open: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'jump', diagnostic: EditorDiagnostic): void;
}>();

const summary = computed(() => summarizeIssues(props.diagnostics));

function severityLabel(d: EditorDiagnostic) {
  if (d.severity === 'error') return 'Error';
  if (d.severity === 'warning') return 'Warning';
  if (d.severity === 'info') return 'Info';
  return 'Hint';
}
</script>

<template>
  <section
    class="shrink-0 flex flex-col bg-[var(--color-page-surface)] shadow-[0_-8px_24px_rgba(0,0,0,0.12)] overflow-hidden transition-[height,border-color] duration-200 ease-out"
    :class="open ? 'border-t border-border' : 'border-t border-transparent'"
    :style="{ height: open ? 'min(40vh, 360px)' : '0px' }"
    :aria-hidden="!open"
    role="region"
    aria-label="Script issues"
  >
      <header class="flex items-center justify-between gap-3 border-b border-border bg-[var(--color-toolbar-bg)] px-4 py-2">
        <div class="flex items-center gap-3">
          <h2 class="text-[10px] font-bold uppercase tracking-[0.2em] text-accent">Script Issues</h2>
          <div class="flex items-center gap-2 text-xs font-medium text-text-muted">
            <span v-if="summary.errors > 0" class="flex items-center gap-1 rounded-full bg-red-500/15 px-2 py-0.5 text-red-600 dark:text-red-400">
              <AlertCircle class="h-3 w-3" /> {{ summary.errors }}
            </span>
            <span v-if="summary.warnings > 0" class="flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-amber-700 dark:text-amber-400">
              <AlertTriangle class="h-3 w-3" /> {{ summary.warnings }}
            </span>
            <span v-if="summary.infos + summary.hints > 0" class="flex items-center gap-1 rounded-full bg-black/5 px-2 py-0.5 dark:bg-white/10">
              <Info class="h-3 w-3" /> {{ summary.infos + summary.hints }}
            </span>
            <span v-if="summary.total === 0" class="text-[10px] uppercase tracking-wider text-text-muted">All clear</span>
          </div>
        </div>
        <button
          type="button"
          class="rounded-full p-1.5 text-text-muted transition-colors hover:bg-black/5 hover:text-text-main dark:hover:bg-white/5"
          aria-label="Close script issues"
          @click="emit('close')"
        >
          <X class="h-4 w-4" />
        </button>
      </header>

      <div
        v-if="diagnostics.length === 0"
        class="flex flex-1 flex-col items-center justify-center gap-2 text-text-muted"
      >
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
          <CheckCircle2 class="h-5 w-5" />
        </div>
        <p class="text-sm font-medium text-text-main">No script issues</p>
        <p class="text-xs">The script is clean.</p>
      </div>

      <ul v-else class="flex-1 divide-y divide-border overflow-y-auto">
        <li
          v-for="(d, index) in diagnostics"
          :key="`${d.from}-${d.to}-${index}`"
          class="group cursor-pointer px-4 py-2 transition-colors hover:bg-black/5 dark:hover:bg-white/5"
          tabindex="0"
          role="button"
          :aria-label="`${severityLabel(d)} at line ${d.line}, column ${d.col}: ${d.message}`"
          @click="emit('jump', d)"
          @keydown.enter.prevent="emit('jump', d)"
          @keydown.space.prevent="emit('jump', d)"
        >
          <div class="flex items-start gap-3">
            <span class="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center">
              <AlertCircle v-if="d.severity === 'error'" class="h-4 w-4 text-red-500" />
              <AlertTriangle v-else-if="d.severity === 'warning'" class="h-4 w-4 text-amber-500" />
              <Info v-else class="h-4 w-4 text-text-muted" />
            </span>
            <span class="shrink-0 font-mono text-[11px] text-text-muted tabular-nums">{{ d.line }}:{{ d.col }}</span>
            <span class="flex-1 text-sm leading-snug text-text-main line-clamp-2">{{ d.message }}</span>
            <span
              v-if="d.code"
              class="shrink-0 rounded border border-border bg-black/5 px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wider text-text-muted dark:bg-white/5"
            >{{ d.code }}</span>
          </div>
        </li>
      </ul>
  </section>
</template>
