<script setup lang="ts">
import { computed } from 'vue';
import { FolderOpen } from 'lucide-vue-next';
import type { FileGitStatus } from './types';

// StatusBar is the desktop app's single bottom chrome strip. It carries:
// - an always-clickable project-label (the sole persistent open-folder
//   affordance — the welcome-screen button disappears once a project is
//   open, so this is what the writer uses to switch projects mid-session)
// - the active file's basename
// - 1-based cursor Ln/Col
// - word count (from manuscriptStats.totalWords, 500ms debounced upstream)
// - dirty dot + "Last snapshot N ago" (from the backend FileGitStatus,
//   with a local fast-path dirty flag for instant feedback on save)

const props = defineProps<{
  projectName: string;
  activeFile: string;
  cursor: { line: number; col: number };
  wordCount: number;
  gitStatus: FileGitStatus | null;
  hasProject: boolean;
  hasActiveFile: boolean;
}>();

defineEmits<{
  (e: 'openFolder'): void;
}>();

// formatRelativeTime renders a compact human-readable duration for the
// "Last snapshot" label. Kept inline rather than shared to avoid a util
// file for a single caller. All branches round down; "just now" covers
// the sub-10s window so the label doesn't flicker right after a save.
function formatRelativeTime(iso: string): string {
  if (!iso) return '';
  const then = Date.parse(iso);
  if (Number.isNaN(then)) return '';
  const deltaSec = Math.max(0, Math.floor((Date.now() - then) / 1000));
  if (deltaSec < 10) return 'just now';
  if (deltaSec < 60) return `${deltaSec}s ago`;
  const deltaMin = Math.floor(deltaSec / 60);
  if (deltaMin < 60) return `${deltaMin}m ago`;
  const deltaHr = Math.floor(deltaMin / 60);
  if (deltaHr < 24) return `${deltaHr}h ago`;
  const deltaDay = Math.floor(deltaHr / 24);
  if (deltaDay < 7) return `${deltaDay}d ago`;
  const deltaWk = Math.floor(deltaDay / 7);
  if (deltaWk < 52) return `${deltaWk}w ago`;
  return `${Math.floor(deltaDay / 365)}y ago`;
}

const snapshotLabel = computed(() => {
  const st = props.gitStatus;
  if (!st) return '';
  if (st.missing) return 'File moved or deleted';
  if (!st.hasHead) return 'No snapshots';
  const rel = formatRelativeTime(st.headAt);
  return rel ? `Last snapshot ${rel}` : '';
});

const showDirty = computed(() => !!props.gitStatus?.dirty && !props.gitStatus?.missing);

const projectButtonLabel = computed(() => {
  if (!props.hasProject) return 'No project — Open folder…';
  return props.projectName || 'Open folder…';
});

const projectButtonTitle = computed(() => {
  return props.hasProject ? 'Change project folder…' : 'Open a project folder';
});
</script>

<template>
  <footer
    class="flex items-center gap-3 h-6 px-3 text-[11px] border-t border-border bg-[var(--color-toolbar-bg)] text-text-muted shrink-0 select-none"
    role="status"
    aria-label="Status bar"
  >
    <!-- Left cluster: project + active file. Project is always a real
         button; clicking it opens the folder picker regardless of
         whether a project is already loaded. -->
    <button
      type="button"
      class="inline-flex items-center gap-1.5 text-text-main hover:text-brass-500 focus:outline-none focus:text-brass-500 transition-colors max-w-[240px] truncate"
      :title="projectButtonTitle"
      @click="$emit('openFolder')"
    >
      <FolderOpen class="w-3 h-3 shrink-0 opacity-70" />
      <span class="font-bold truncate">{{ projectButtonLabel }}</span>
    </button>

    <template v-if="hasActiveFile">
      <span class="opacity-40">/</span>
      <span class="truncate max-w-[200px]" :title="activeFile">{{ activeFile || '—' }}</span>
    </template>

    <!-- Spacer pushes the right cluster to the edge. -->
    <span class="flex-1" aria-hidden="true"></span>

    <span v-if="hasActiveFile" class="tabular-nums">
      Ln {{ cursor.line }}, Col {{ cursor.col }}
    </span>

    <span
      v-if="hasActiveFile && wordCount > 0"
      class="tabular-nums"
    >
      {{ wordCount.toLocaleString() }} words
    </span>

    <span v-if="gitStatus && hasActiveFile" class="inline-flex items-center gap-1.5">
      <span
        v-if="showDirty"
        class="inline-block w-1.5 h-1.5 rounded-full bg-amber-500"
        :title="'Unsaved changes since last snapshot'"
        aria-hidden="true"
      ></span>
      <span>{{ snapshotLabel }}</span>
    </span>
  </footer>
</template>
