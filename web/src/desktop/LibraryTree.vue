<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { FolderPlus, Edit3 } from 'lucide-vue-next';
import type { Workspace } from './workspace';
import type { LibraryNode } from './types';
import LibraryTreeNode from './LibraryTreeNode.vue';

// Sidebar tree renderer. Expand/collapse, click-to-select, drag-and-
// drop move (file or folder → folder), right-click context menu for
// Rename / New Folder Here, inline rename via Enter/Escape.
//
// v1 scope: no multi-select, no keyboard navigation, no cut/copy/
// paste, no delete. Those can follow.

const props = defineProps<{
  workspace: Workspace;
}>();

const emit = defineEmits<{
  (e: 'select-file', path: string): void;
  (e: 'error', message: string): void;
  (e: 'info', message: string): void;
}>();

const renamingPath = ref<string | null>(null);
const renameValue = ref('');
const renameInput = ref<HTMLInputElement | null>(null);

const contextMenu = ref<{
  node: LibraryNode;
  x: number;
  y: number;
} | null>(null);

const draggedPath = ref<string | null>(null);
const dropTarget = ref<string | null>(null);

const tree = computed(() => props.workspace.state.libraryTree);

function isExpanded(path: string): boolean {
  return props.workspace.state.expandedFolders.has(path);
}

function toggleExpand(path: string) {
  props.workspace.toggleFolderExpansion(path);
}

async function startRename(node: LibraryNode) {
  renamingPath.value = node.path;
  renameValue.value = node.name;
  contextMenu.value = null;
  await nextTick();
  renameInput.value?.focus();
  renameInput.value?.select();
}

async function commitRename(node: LibraryNode) {
  const newName = renameValue.value.trim();
  renamingPath.value = null;
  if (!newName || newName === node.name) return;
  try {
    const newPath = await props.workspace.renameEntry(node.path, newName);
    if (node.kind === 'file') emit('select-file', newPath);
  } catch (e: any) {
    emit('error', `Rename failed: ${e?.message ?? e}`);
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

async function contextNewFolder(parent: LibraryNode | null) {
  closeContextMenu();
  const raw = typeof globalThis.prompt === 'function'
    ? globalThis.prompt('Folder name')
    : null;
  const name = raw?.trim();
  if (!name) return;
  if (name.includes('/') || name.includes('\\')) {
    emit('error', 'Folder names cannot contain slashes');
    return;
  }
  const relPath = parent ? `${parent.path}/${name}` : name;
  try {
    await props.workspace.createFolder(relPath);
  } catch (e: any) {
    emit('error', `Failed to create folder: ${e?.message ?? e}`);
  }
}

function onSelectFile(node: LibraryNode) {
  if (renamingPath.value === node.path) return;
  emit('select-file', node.path);
}

// --- Drag-and-drop ---

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
  // Dropping a folder into its own descendant is a no-op at best and
  // an rm -rf at worst. Block client-side; backend re-validates.
  if (targetPath.startsWith(draggedPath.value + '/')) return false;
  // No-op: the source already lives directly under the target.
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
  dropTarget.value = null;
  draggedPath.value = null;
  if (!src || !canDropOn(targetPath)) return;
  const name = src.includes('/') ? src.slice(src.lastIndexOf('/') + 1) : src;
  const dst = targetPath ? `${targetPath}/${name}` : name;
  try {
    const newPath = await props.workspace.moveEntry(src, dst);
    emit('info', `Moved to ${newPath}`);
  } catch (e: any) {
    emit('error', `Move failed: ${e?.message ?? e}`);
  }
}
</script>

<template>
  <div class="flex-1 overflow-y-auto p-2 custom-scrollbar border-b border-border" @click="closeContextMenu">
    <div class="flex items-center justify-between px-2 pb-2">
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
    </div>
  </div>
</template>
