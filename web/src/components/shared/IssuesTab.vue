<script setup lang="ts">
import { computed } from 'vue';
import { AlertCircle, AlertTriangle, CheckCircle2, Info } from 'lucide-vue-next';
import type { EditorDiagnostic } from '../../core/types';
import { summarizeIssues } from '../../core/issues';
import type { FilterSeverity } from '../../core/issues';

const props = withDefaults(
  defineProps<{
    diagnostics: EditorDiagnostic[];
    hiddenSeverities?: ReadonlySet<FilterSeverity>;
  }>(),
  { hiddenSeverities: () => new Set() },
);

const emit = defineEmits<{
  (e: 'jump', diagnostic: EditorDiagnostic): void;
  (e: 'update:hiddenSeverities', next: ReadonlySet<FilterSeverity>): void;
}>();

const summary = computed(() => summarizeIssues(props.diagnostics));

function isHidden(kind: FilterSeverity): boolean {
  return props.hiddenSeverities.has(kind);
}

function toggleSeverity(kind: FilterSeverity) {
  const next = new Set(props.hiddenSeverities);
  if (next.has(kind)) next.delete(kind);
  else next.add(kind);
  emit('update:hiddenSeverities', next);
}

function pluralize(n: number, label: string): string {
  return `${n} ${label}${n === 1 ? '' : 's'}`;
}

const errorTitle = computed(() =>
  `${pluralize(summary.value.errors, 'error')} — click to ${isHidden('error') ? 'show' : 'hide'}`,
);
const warningTitle = computed(() =>
  `${pluralize(summary.value.warnings, 'warning')} — click to ${isHidden('warning') ? 'show' : 'hide'}`,
);
const infoTitle = computed(() => {
  const count = summary.value.infos + summary.value.hints;
  return `${pluralize(count, 'info issue')} — click to ${isHidden('info') ? 'show' : 'hide'}`;
});

const visibleDiagnostics = computed(() =>
  props.diagnostics.filter((d) => {
    if (d.severity === 'error') return !isHidden('error');
    if (d.severity === 'warning') return !isHidden('warning');
    return !isHidden('info');
  }),
);

function severityLabel(d: EditorDiagnostic) {
  if (d.severity === 'error') return 'Error';
  if (d.severity === 'warning') return 'Warning';
  if (d.severity === 'info') return 'Info';
  return 'Hint';
}
</script>

<template>
  <div class="flex flex-1 flex-col overflow-hidden">
    <div class="flex items-center gap-3 border-b border-border bg-[var(--color-toolbar-bg)] px-4 py-2">
      <div class="flex items-center gap-2 text-xs font-medium text-text-muted">
        <button
          v-if="summary.errors > 0"
          type="button"
          class="flex items-center gap-1 rounded-full bg-red-500/15 px-2 py-0.5 text-red-600 transition-opacity hover:opacity-100 dark:text-red-400"
          :class="{ 'opacity-40': isHidden('error') }"
          :title="errorTitle"
          :aria-pressed="!isHidden('error')"
          @click="toggleSeverity('error')"
        >
          <AlertCircle class="h-3 w-3" /> {{ summary.errors }}
        </button>
        <button
          v-if="summary.warnings > 0"
          type="button"
          class="flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-amber-700 transition-opacity hover:opacity-100 dark:text-amber-400"
          :class="{ 'opacity-40': isHidden('warning') }"
          :title="warningTitle"
          :aria-pressed="!isHidden('warning')"
          @click="toggleSeverity('warning')"
        >
          <AlertTriangle class="h-3 w-3" /> {{ summary.warnings }}
        </button>
        <button
          v-if="summary.infos + summary.hints > 0"
          type="button"
          class="flex items-center gap-1 rounded-full bg-purple-500/15 px-2 py-0.5 text-purple-700 transition-opacity hover:opacity-100 dark:text-purple-300"
          :class="{ 'opacity-40': isHidden('info') }"
          :title="infoTitle"
          :aria-pressed="!isHidden('info')"
          @click="toggleSeverity('info')"
        >
          <Info class="h-3 w-3" /> {{ summary.infos + summary.hints }}
        </button>
        <span v-if="summary.total === 0" class="text-[10px] uppercase tracking-wider text-text-muted">All clear</span>
      </div>
    </div>

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

    <ul v-else-if="visibleDiagnostics.length > 0" class="flex-1 divide-y divide-border overflow-y-auto">
      <li
        v-for="(d, index) in visibleDiagnostics"
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
            <Info v-else class="h-4 w-4 text-purple-600 dark:text-purple-300" />
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

    <div
      v-else
      class="flex flex-1 flex-col items-center justify-center gap-1 px-6 text-center text-text-muted"
    >
      <p class="text-sm font-medium text-text-main">All matching issues hidden</p>
      <p class="text-xs">Click a pill to show its issues.</p>
    </div>
  </div>
</template>
