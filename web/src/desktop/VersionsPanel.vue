<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { History, FolderSync } from 'lucide-vue-next';
import type { Workspace } from './workspace';
import type { Revision } from './types';
import { formatRevisionTimestamp } from './revision-format';

const props = defineProps<{
  workspace: Workspace;
  isViewingRevision: boolean;
}>();

const emit = defineEmits<{
  (e: 'view-revision', hash: string): void;
  (e: 'exit-revision-view'): void;
  (e: 'compare-to-current', hash: string): void;
  (e: 'error', message: string): void;
  (e: 'info', message: string): void;
}>();

const inPickingMode = computed(
  () => props.workspace.state.pickingSecondForCompare,
);

function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as { message?: unknown }).message ?? error);
  }
  return String(error);
}

const revisionMenu = ref<{ rev: Revision; x: number; y: number } | null>(null);

function openRevisionMenu(event: MouseEvent, rev: Revision) {
  event.preventDefault();
  revisionMenu.value = { rev, x: event.clientX, y: event.clientY };
}

function closeRevisionMenu() {
  revisionMenu.value = null;
}

async function onRevisionRowClick(hash: string) {
  if (props.workspace.state.pickingSecondForCompare) {
    try {
      await props.workspace.resolvePickSecond(hash);
    } catch (error: unknown) {
      emit('error', `Failed to load second version: ${errorMessage(error)}`);
    }
    return;
  }
  emit('view-revision', hash);
}

function menuViewRevision(rev: Revision) {
  closeRevisionMenu();
  emit('view-revision', rev.hash);
}

function menuCompareToCurrent(rev: Revision) {
  closeRevisionMenu();
  emit('compare-to-current', rev.hash);
}

async function menuCompareWith(rev: Revision) {
  closeRevisionMenu();
  try {
    await props.workspace.startPickSecond(rev.hash);
  } catch (error: unknown) {
    emit('error', `Failed to load version: ${errorMessage(error)}`);
  }
}

async function menuCopyHash(rev: Revision) {
  closeRevisionMenu();
  try {
    await navigator.clipboard.writeText(rev.hash);
    emit('info', 'Hash copied to clipboard');
  } catch {
    emit('error', 'Failed to copy hash');
  }
}

async function menuHideRevision(rev: Revision) {
  closeRevisionMenu();
  try {
    await props.workspace.hideRevision(rev.hash);
  } catch (error: unknown) {
    emit('error', `Failed to hide version: ${errorMessage(error)}`);
  }
}

async function menuUnhideRevision(rev: Revision) {
  closeRevisionMenu();
  try {
    await props.workspace.unhideRevision(rev.hash);
  } catch (error: unknown) {
    emit('error', `Failed to unhide version: ${errorMessage(error)}`);
  }
}

function onPickingKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.workspace.state.pickingSecondForCompare) {
    props.workspace.cancelPickSecond();
  }
}
onMounted(() => {
  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', onPickingKeydown);
  }
});
onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', onPickingKeydown);
  }
});
</script>

<template>
  <div v-if="workspace.state.activeFile" class="h-1/3 flex flex-col bg-black/[0.01] dark:bg-white/[0.01]">
    <div class="p-3 border-b border-border bg-black/[0.02] dark:bg-white/[0.02] flex items-center justify-between gap-2">
      <h3 class="text-[10px] uppercase tracking-[0.2em] text-text-muted font-bold flex items-center gap-2">
        <History class="w-3.5 h-3.5 opacity-50" /> Versions
      </h3>
      <button
        v-if="workspace.state.hiddenRevisionHashes.size > 0"
        type="button"
        @click="workspace.toggleShowHidden()"
        class="text-[9px] uppercase tracking-wider font-bold text-text-muted hover:text-brass-500 transition-colors"
        :title="workspace.state.showHidden ? 'Collapse hidden versions' : 'Show hidden versions for unhiding'"
      >
        {{ workspace.state.showHidden ? 'Hide hidden' : `Show hidden (${workspace.state.hiddenRevisionHashes.size})` }}
      </button>
    </div>
    <div
      v-if="inPickingMode && workspace.state.viewingRevisionMeta"
      class="px-3 py-2 border-b border-amber-500/30 bg-amber-500/10 text-[10px] text-amber-700 dark:text-amber-300 flex items-center justify-between gap-2"
    >
      <span class="truncate">
        <span class="font-bold">Pick another version</span> to compare with
        <span class="italic">{{ formatRevisionTimestamp(workspace.state.viewingRevisionMeta.timestamp) }}</span>
      </span>
      <button
        type="button"
        @click="workspace.cancelPickSecond()"
        class="shrink-0 font-bold hover:underline"
      >
        Cancel
      </button>
    </div>
    <div class="flex-1 overflow-y-auto custom-scrollbar p-2 space-y-1" @click="closeRevisionMenu">
      <button
        v-if="workspace.state.revisions.length > 0"
        type="button"
        @click="emit('exit-revision-view')"
        class="w-full text-left p-2 rounded transition-colors border"
        :class="!isViewingRevision
            ? 'bg-brass-500/10 border-brass-500/20 text-brass-500 font-bold'
            : 'border-transparent hover:bg-black/5 dark:hover:bg-white/5 text-text-main'"
        :title="isViewingRevision ? 'Return to current version' : 'Current version'"
      >
        <div class="text-[11px] font-bold truncate flex items-center gap-1.5">
          <FolderSync v-if="!isViewingRevision" class="w-3 h-3" />
          <span>Current (editing)</span>
        </div>
      </button>
      <button
        v-for="rev in workspace.visibleRevisions.value"
        :key="rev.hash"
        type="button"
        @click="onRevisionRowClick(rev.hash)"
        @contextmenu.prevent="openRevisionMenu($event, rev)"
        class="w-full text-left p-2 rounded transition-colors border"
        :class="[
          workspace.state.viewingRevisionHash === rev.hash
            ? 'bg-brass-500/10 border-brass-500/20 text-brass-500'
            : 'border-transparent hover:bg-black/5 dark:hover:bg-white/5 text-text-main',
          workspace.state.compareSecondHash === rev.hash
            ? 'ring-1 ring-amber-500/60' : '',
          workspace.state.hiddenRevisionHashes.has(rev.hash)
            ? 'opacity-50 italic' : '',
        ]"
        :title="inPickingMode
          ? (workspace.state.viewingRevisionHash === rev.hash
              ? 'This version is already selected as A'
              : `Compare with this version (${formatRevisionTimestamp(rev.timestamp)})`)
          : `Preview this version (${formatRevisionTimestamp(rev.timestamp)})`"
      >
        <div class="text-[11px] font-bold truncate flex items-center gap-1.5">
          <span
            v-if="inPickingMode && workspace.state.viewingRevisionHash === rev.hash"
            class="inline-flex items-center justify-center w-3.5 h-3.5 rounded-full bg-amber-500/30 text-amber-700 dark:text-amber-200 text-[8px] font-bold"
          >A</span>
          <span class="truncate">{{ rev.message }}</span>
        </div>
        <div class="flex justify-end items-center mt-1">
          <span class="text-[9px] text-text-muted italic">{{ formatRevisionTimestamp(rev.timestamp) }}</span>
        </div>
      </button>
      <div v-if="workspace.visibleRevisions.value.length === 0 && workspace.state.revisions.length === 0" class="p-4 text-center">
        <p class="text-[10px] text-text-muted italic">No versions yet. Click "Save Version" to create one.</p>
      </div>
      <div v-else-if="workspace.visibleRevisions.value.length === 0" class="p-4 text-center">
        <p class="text-[10px] text-text-muted italic">All versions hidden. Click "Show hidden" above to unhide.</p>
      </div>
    </div>
    <div
      v-if="revisionMenu"
      class="fixed z-50 min-w-[180px] rounded-md border border-border bg-[var(--color-page-surface)] shadow-lg py-1"
      :style="{ left: revisionMenu.x + 'px', top: revisionMenu.y + 'px' }"
      @click.stop
    >
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main"
        @click="menuViewRevision(revisionMenu.rev)"
      >
        View this version
      </button>
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main"
        @click="menuCompareToCurrent(revisionMenu.rev)"
      >
        Compare to current
      </button>
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main"
        @click="menuCompareWith(revisionMenu.rev)"
      >
        Compare with…
      </button>
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main"
        @click="menuCopyHash(revisionMenu.rev)"
      >
        Copy hash
      </button>
      <div class="my-1 border-t border-border" />
      <button
        v-if="!workspace.state.hiddenRevisionHashes.has(revisionMenu.rev.hash)"
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main"
        @click="menuHideRevision(revisionMenu.rev)"
      >
        Hide this version
      </button>
      <button
        v-else
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main"
        @click="menuUnhideRevision(revisionMenu.rev)"
      >
        Unhide
      </button>
    </div>
  </div>
</template>
