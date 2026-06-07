<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { FolderPlus, Edit3, Trash2, Undo2 } from 'lucide-vue-next';
import type { Workspace } from './workspace';
import type { DirtyPath, LibraryNode } from './types';
import LibraryTreeNode from './LibraryTreeNode.vue';
import { displayFileName, displayFilePath, normalizeFileRename } from './naming';

const props = defineProps<{
  workspace: Workspace;
}>();

const emit = defineEmits<{
  (e: 'select-file', path: string): void;
  (e: 'error', message: string): void;
  (e: 'info', message: string): void;
  (e: 'request-new-folder', parentPath: string): void;
  (e: 'request-delete-file', path: string): void;
  (e: 'request-restore-file', path: string): void;
  (e: 'request-permanent-delete', path: string): void;
}>();

const renamingPath = ref<string | null>(null);
const renameValue = ref('');
const renameInput = ref<HTMLInputElement | null>(null);

const contextMenu = ref<{
  node: LibraryNode;
  x: number;
  y: number;
} | null>(null);

const deletedMenu = ref<{ path: string; name: string; x: number; y: number } | null>(null);

const draggedPath = ref<string | null>(null);
const dropTarget = ref<string | null>(null);

const deletedFiles = computed(() => props.workspace.deletedFiles.value);

function fileName(path: string): string {
  return path.includes('/') ? path.slice(path.lastIndexOf('/') + 1) : path;
}

function openDeletedMenu(event: MouseEvent, dp: DirtyPath) {
  event.preventDefault();
  deletedMenu.value = { path: dp.path, name: fileName(dp.path), x: event.clientX, y: event.clientY };
}

function closeDeletedMenu() {
  deletedMenu.value = null;
}

function contextDeleteFile() {
  if (!contextMenu.value) return;
  const path = contextMenu.value.node.path;
  closeContextMenu();
  emit('request-delete-file', path);
}

function menuRestore() {
  if (!deletedMenu.value) return;
  const path = deletedMenu.value.path;
  closeDeletedMenu();
  emit('request-restore-file', path);
}

function menuPermanentDelete() {
  if (!deletedMenu.value) return;
  const path = deletedMenu.value.path;
  closeDeletedMenu();
  emit('request-permanent-delete', path);
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as { message?: unknown }).message ?? error);
  }
  return String(error);
}

const tree = computed(() => props.workspace.state.libraryTree);

function isExpanded(path: string): boolean {
  return props.workspace.state.expandedFolders.has(path);
}

function toggleExpand(path: string) {
  props.workspace.toggleFolderExpansion(path);
}

async function startRename(node: LibraryNode) {
  renamingPath.value = node.path;
  // Show only the base name in the rename input; the .ds extension is
  // re-applied in commitRename. Folders keep their full name.
  renameValue.value = node.kind === 'file' ? displayFileName(node.name) : node.name;
  contextMenu.value = null;
  await nextTick();
  renameInput.value?.focus();
  renameInput.value?.select();
}

async function commitRename(node: LibraryNode) {
  const raw = renameValue.value.trim();
  renamingPath.value = null;
  if (!raw) return;
  const newName = node.kind === 'file' ? normalizeFileRename(raw) : raw;
  if (newName === node.name) return;
  try {
    const newPath = await props.workspace.renameEntry(node.path, newName);
    if (node.kind === 'file') emit('select-file', newPath);
  } catch (error: unknown) {
    emit('error', `Rename failed: ${errorMessage(error)}`);
  }
}

function cancelRename() {
  renamingPath.value = null;
}

function openContextMenu(event: MouseEvent, node: LibraryNode) {
  event.preventDefault();
  contextMenu.value = { node, x: event.clientX, y: event.clientY };
}

function closeContextMenu() {
  contextMenu.value = null;
}

function contextNewFolder(parent: LibraryNode | null) {
  closeContextMenu();
  emit('request-new-folder', parent ? parent.path : '');
}

function onSelectFile(node: LibraryNode) {
  if (renamingPath.value === node.path) return;
  emit('select-file', node.path);
}

function onDragStart(event: DragEvent, node: LibraryNode) {
  draggedPath.value = node.path;
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', node.path);
  }
}

function onDragEnd() {
  draggedPath.value = null;
  dropTarget.value = null;
}

function canDropOn(targetPath: string): boolean {
  if (draggedPath.value === null) return false;
  if (targetPath === draggedPath.value) return false;
  if (targetPath.startsWith(draggedPath.value + '/')) return false;
  const srcParent = draggedPath.value.includes('/')
    ? draggedPath.value.slice(0, draggedPath.value.lastIndexOf('/'))
    : '';
  if (srcParent === targetPath) return false;
  return true;
}

function onDragOver(event: DragEvent, targetPath: string) {
  if (!canDropOn(targetPath)) return;
  event.preventDefault();
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
  dropTarget.value = targetPath;
}

async function onDrop(event: DragEvent, targetPath: string) {
  event.preventDefault();
  const src = draggedPath.value;
  const allowed = src !== null && canDropOn(targetPath);
  dropTarget.value = null;
  draggedPath.value = null;
  if (!allowed || !src) return;
  const name = src.includes('/') ? src.slice(src.lastIndexOf('/') + 1) : src;
  const dst = targetPath ? `${targetPath}/${name}` : name;
  try {
    const newPath = await props.workspace.moveEntry(src, dst);
    emit('info', `Moved to ${newPath}`);
  } catch (error: unknown) {
    emit('error', `Move failed: ${errorMessage(error)}`);
  }
}
</script>

<template>
  <div class="flex-1 overflow-y-auto p-2 custom-scrollbar border-b border-border" @click="() => { closeContextMenu(); closeDeletedMenu(); }">
    <div
      class="flex items-center justify-between px-2 pb-2 rounded transition-colors"
      :class="dropTarget === '' ? 'bg-brass-500/10 ring-1 ring-brass-500/30' : ''"
      @dragover.stop="onDragOver($event, '')"
      @drop.stop="onDrop($event, '')"
    >
      <span class="text-[10px] uppercase tracking-[0.2em] text-text-muted font-bold">Files</span>
      <button
        type="button"
        class="p-1 rounded text-text-muted hover:text-brass-500 hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
        title="New Folder"
        @click.stop="contextNewFolder(null)"
      >
        <FolderPlus class="w-3.5 h-3.5" />
      </button>
    </div>

    <div
      v-if="tree.length === 0"
      class="p-4 text-center"
      @dragover="onDragOver($event, '')"
      @drop="onDrop($event, '')"
      :class="{ 'bg-brass-500/10 ring-1 ring-brass-500/30 rounded-md': dropTarget === '' }"
    >
      <p class="text-xs text-text-muted italic text-balance">This library is empty. Create a play or a folder to get started.</p>
    </div>

    <div
      v-else
      class="space-y-0.5"
      @dragover="onDragOver($event, '')"
      @drop="onDrop($event, '')"
      :class="{ 'bg-brass-500/5 ring-1 ring-brass-500/20 rounded-md': dropTarget === '' }"
    >
      <LibraryTreeNode
        v-for="node in tree"
        :key="node.path"
        :node="node"
        :depth="0"
        :active-file="workspace.state.activeFile"
        :renaming-path="renamingPath"
        :rename-value="renameValue"
        :drop-target="dropTarget"
        :is-expanded="isExpanded"
        :can-drop-on="canDropOn"
        @update:rename-value="renameValue = $event"
        @toggle-expand="toggleExpand"
        @select-file="onSelectFile"
        @context-menu="openContextMenu"
        @drag-start="onDragStart"
        @drag-end="onDragEnd"
        @drag-over="onDragOver"
        @drop="onDrop"
        @commit-rename="commitRename"
        @cancel-rename="cancelRename"
        @set-rename-input="(el) => { renameInput = el; }"
      />
    </div>

    <div
      v-if="contextMenu"
      class="fixed z-50 min-w-[160px] rounded-md border border-border bg-[var(--color-page-surface)] shadow-lg py-1"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      @click.stop
    >
      <button
        v-if="contextMenu.node.kind === 'folder'"
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main inline-flex items-center gap-2"
        @click="() => { const n = contextMenu!.node; closeContextMenu(); contextNewFolder(n); }"
      >
        <FolderPlus class="w-3.5 h-3.5" /> New Folder Here
      </button>
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main inline-flex items-center gap-2"
        @click="() => { const n = contextMenu!.node; closeContextMenu(); startRename(n); }"
      >
        <Edit3 class="w-3.5 h-3.5" /> Rename
      </button>
      <button
        v-if="contextMenu.node.kind === 'file'"
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-red-500/10 text-red-600 dark:text-red-400 inline-flex items-center gap-2"
        @click="contextDeleteFile"
      >
        <Trash2 class="w-3.5 h-3.5" /> Delete…
      </button>
    </div>

    <div
      v-if="deletedFiles.length > 0"
      class="mt-4 pt-2 border-t border-border"
    >
      <div class="flex items-center px-2 pb-2">
        <span class="text-[10px] uppercase tracking-[0.2em] text-text-muted font-bold">
          Deleted ({{ deletedFiles.length }})
        </span>
      </div>
      <div class="space-y-0.5">
        <button
          v-for="dp in deletedFiles"
          :key="dp.path"
          type="button"
          class="w-full text-left px-2 py-1 rounded text-xs flex items-center gap-2 text-text-muted line-through italic opacity-70 hover:opacity-100 hover:bg-black/5 dark:hover:bg-white/5"
          :title="`Deleted: ${dp.path} — right-click to restore`"
          @contextmenu.prevent="openDeletedMenu($event, dp)"
        >
          <Undo2 class="w-3 h-3 shrink-0 opacity-60" />
          <span class="truncate">{{ displayFilePath(dp.path) }}</span>
        </button>
      </div>
    </div>

    <div
      v-if="deletedMenu"
      class="fixed z-50 min-w-[200px] rounded-md border border-border bg-[var(--color-page-surface)] shadow-lg py-1"
      :style="{ left: deletedMenu.x + 'px', top: deletedMenu.y + 'px' }"
      @click.stop
    >
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-brass-500/10 text-text-main inline-flex items-center gap-2"
        @click="menuRestore"
      >
        <Undo2 class="w-3.5 h-3.5" /> Restore from last version
      </button>
      <div class="my-1 border-t border-border" />
      <button
        type="button"
        class="w-full text-left px-3 py-1.5 text-xs hover:bg-red-500/10 text-red-600 dark:text-red-400 inline-flex items-center gap-2"
        @click="menuPermanentDelete"
      >
        <Trash2 class="w-3.5 h-3.5" /> Permanently delete
      </button>
    </div>
  </div>
</template>
