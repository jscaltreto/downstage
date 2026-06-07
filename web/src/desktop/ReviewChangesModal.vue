<script setup lang="ts">
import { computed, ref } from 'vue';
import { Check, GitCommit, RefreshCw, Trash2 } from 'lucide-vue-next';
import BaseModal from '../components/shared/BaseModal.vue';
import PromptModal from './PromptModal.vue';
import ConfirmModal from './ConfirmModal.vue';
import type { DirtyPath, LibraryDirty } from './types';

// Surfaces library-wide uncommitted changes — sibling-file edits, untracked
// new files, deleted-tracked files, sidecar (.downstage/*) drift — and lets
// the user commit or discard them in bulk. The canonical path for sibling
// reconciliation; SnapshotFile stays single-path and intentionally does NOT
// pick these up.

const props = defineProps<{
  open: boolean;
  dirty: LibraryDirty | null;
  busy: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'commit', paths: string[], message: string): void;
  (e: 'discard', paths: string[]): void;
  (e: 'refresh'): void;
}>();

const promptOpen = ref(false);
const pendingCommitPaths = ref<string[]>([]);
const promptInitial = ref('');

const discardConfirmOpen = ref(false);
const pendingDiscardPaths = ref<string[]>([]);
const discardConfirmTitle = ref('');
const discardConfirmMessage = ref('');

const sections = computed(() => {
  const d = props.dirty;
  if (!d) return [];
  return [
    { id: 'plays', label: 'Plays', items: d.plays },
    { id: 'sidecars', label: 'Library settings — auto-managed by Downstage', items: d.sidecars },
    { id: 'other', label: 'Other', items: d.other },
  ].filter((s) => s.items.length > 0);
});

const totalCount = computed(() => props.dirty?.count ?? 0);

function kindLabel(kind: DirtyPath['kind']): string {
  if (kind === 'untracked') return 'new';
  if (kind === 'deleted') return 'deleted';
  return 'modified';
}

function kindClass(kind: DirtyPath['kind']): string {
  if (kind === 'untracked') return 'text-emerald-600 dark:text-emerald-400';
  if (kind === 'deleted') return 'text-red-600 dark:text-red-400';
  return 'text-amber-600 dark:text-amber-400';
}

function openCommitPrompt(paths: string[], initial: string) {
  pendingCommitPaths.value = paths;
  promptInitial.value = initial;
  promptOpen.value = true;
}

function submitCommit(message: string) {
  const paths = pendingCommitPaths.value;
  promptOpen.value = false;
  pendingCommitPaths.value = [];
  if (paths.length > 0) emit('commit', paths, message);
}

function commitRow(dp: DirtyPath) {
  const verb = dp.kind === 'deleted' ? 'Delete' : dp.kind === 'untracked' ? 'Add' : 'Update';
  openCommitPrompt([dp.path], `${verb} ${fileName(dp.path)}`);
}

function commitSection(sectionId: string, items: DirtyPath[]) {
  const paths = items.map((i) => i.path);
  let initial = 'Library cleanup';
  if (sectionId === 'sidecars') initial = 'Update library settings';
  else if (sectionId === 'plays') initial = `Commit ${items.length} script change${items.length === 1 ? '' : 's'}`;
  openCommitPrompt(paths, initial);
}

function commitAll() {
  if (!props.dirty) return;
  const paths = [
    ...props.dirty.plays.map((p) => p.path),
    ...props.dirty.sidecars.map((p) => p.path),
    ...props.dirty.other.map((p) => p.path),
  ];
  if (paths.length === 0) return;
  openCommitPrompt(paths, 'Library cleanup');
}

function discardRow(dp: DirtyPath) {
  pendingDiscardPaths.value = [dp.path];
  discardConfirmTitle.value = 'Discard change?';
  discardConfirmMessage.value = `Discard changes to ${dp.path}? This cannot be undone.`;
  discardConfirmOpen.value = true;
}

function discardSection(items: DirtyPath[]) {
  if (items.length === 0) return;
  pendingDiscardPaths.value = items.map((i) => i.path);
  discardConfirmTitle.value = 'Discard changes?';
  discardConfirmMessage.value = `Discard ${items.length} change${items.length === 1 ? '' : 's'}? This cannot be undone.`;
  discardConfirmOpen.value = true;
}

function confirmDiscard() {
  const paths = pendingDiscardPaths.value;
  discardConfirmOpen.value = false;
  pendingDiscardPaths.value = [];
  if (paths.length > 0) emit('discard', paths);
}

function cancelDiscard() {
  discardConfirmOpen.value = false;
  pendingDiscardPaths.value = [];
}

function fileName(path: string): string {
  return path.includes('/') ? path.slice(path.lastIndexOf('/') + 1) : path;
}
</script>

<template>
  <BaseModal :open="open" title="Review Library Changes" @close="emit('close')">
    <div class="flex flex-col gap-4 min-w-[480px] max-w-[640px]">
      <div class="flex items-center justify-between">
        <p class="text-xs text-text-muted">
          <span v-if="totalCount === 0">Library is clean — no uncommitted changes.</span>
          <span v-else>
            {{ totalCount }} uncommitted change{{ totalCount === 1 ? '' : 's' }} across your library.
            Snapshots only commit the active file; this is where sibling and out-of-band changes get reconciled.
          </span>
        </p>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded border border-border px-2 py-1 text-xs text-text-main hover:bg-black/5 dark:hover:bg-white/5 disabled:opacity-50"
          :disabled="busy"
          @click="emit('refresh')"
        >
          <RefreshCw class="w-3 h-3" /> Refresh
        </button>
      </div>

      <div
        v-for="section in sections"
        :key="section.id"
        class="flex flex-col gap-1.5 rounded-md border border-border p-3"
      >
        <div class="flex items-center justify-between">
          <h4 class="text-[10px] uppercase tracking-[0.15em] font-bold text-text-muted">
            {{ section.label }} ({{ section.items.length }})
          </h4>
          <div class="flex gap-1.5">
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-text-main hover:bg-brass-500/10 disabled:opacity-50"
              :disabled="busy"
              @click="commitSection(section.id, section.items)"
            >
              <GitCommit class="w-3 h-3" /> Commit all
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-text-muted hover:bg-red-500/10 hover:text-red-600 dark:hover:text-red-400 disabled:opacity-50"
              :disabled="busy"
              @click="discardSection(section.items)"
            >
              <Trash2 class="w-3 h-3" /> Discard all
            </button>
          </div>
        </div>
        <ul class="flex flex-col gap-0.5">
          <li
            v-for="item in section.items"
            :key="item.path"
            class="flex items-center justify-between gap-2 text-xs py-1 px-2 rounded hover:bg-black/5 dark:hover:bg-white/5"
          >
            <span class="flex items-center gap-2 truncate flex-1">
              <span
                class="text-[9px] uppercase tracking-wide font-bold w-16 shrink-0"
                :class="kindClass(item.kind)"
              >
                {{ kindLabel(item.kind) }}
              </span>
              <span class="truncate text-text-main" :title="item.path">{{ item.path }}</span>
            </span>
            <div class="flex gap-1 shrink-0">
              <button
                type="button"
                class="rounded p-1 text-text-muted hover:bg-brass-500/10 hover:text-brass-500 disabled:opacity-50"
                title="Commit this change"
                :disabled="busy"
                @click="commitRow(item)"
              >
                <Check class="w-3 h-3" />
              </button>
              <button
                type="button"
                class="rounded p-1 text-text-muted hover:bg-red-500/10 hover:text-red-600 dark:hover:text-red-400 disabled:opacity-50"
                title="Discard this change"
                :disabled="busy"
                @click="discardRow(item)"
              >
                <Trash2 class="w-3 h-3" />
              </button>
            </div>
          </li>
        </ul>
      </div>

      <div class="flex justify-end gap-2 pt-1 border-t border-border">
        <button
          type="button"
          class="rounded-md border border-border px-3 py-1.5 text-sm font-bold text-text-main hover:bg-black/5 dark:hover:bg-white/5"
          @click="emit('close')"
        >
          Close
        </button>
        <button
          type="button"
          class="rounded-md bg-brass-500 px-3 py-1.5 text-sm font-bold text-ember-950 hover:bg-brass-400 disabled:opacity-50"
          :disabled="busy || totalCount === 0"
          @click="commitAll"
        >
          Commit all changes…
        </button>
      </div>
    </div>
  </BaseModal>

  <PromptModal
    :open="promptOpen"
    title="Commit Changes"
    label="Commit message"
    :initial-value="promptInitial"
    submit-label="Commit"
    @close="promptOpen = false; pendingCommitPaths = []"
    @submit="submitCommit"
  />

  <ConfirmModal
    :open="discardConfirmOpen"
    :title="discardConfirmTitle"
    :message="discardConfirmMessage"
    confirm-label="Discard"
    destructive
    @close="cancelDiscard"
    @confirm="confirmDiscard"
  />
</template>
