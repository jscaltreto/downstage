<script lang="ts">
// Needed for recursive self-reference by name in the template.
export default { name: 'LibraryTreeNode' };
</script>

<script setup lang="ts">
import {
  ChevronDown, ChevronRight, FileText, Folder, FolderOpen,
} from 'lucide-vue-next';
import type { LibraryNode } from './types';

// Recursive tree-row renderer for LibraryTree.vue. Presentational —
// all state (rename target, drop target, expanded set) lives in the
// parent and is passed down as props. Events bubble back up so the
// parent can update its single source of truth.

const props = defineProps<{
  node: LibraryNode;
  depth: number;
  activeFile: string | null;
  renamingPath: string | null;
  renameValue: string;
  dropTarget: string | null;
  isExpanded: (path: string) => boolean;
  canDropOn: (path: string) => boolean;
}>();

const emit = defineEmits<{
  (e: 'update:renameValue', value: string): void;
  (e: 'toggleExpand', path: string): void;
  (e: 'selectFile', node: LibraryNode): void;
  (e: 'contextMenu', event: MouseEvent, node: LibraryNode): void;
  (e: 'dragStart', event: DragEvent, node: LibraryNode): void;
  (e: 'dragEnd'): void;
  (e: 'dragOver', event: DragEvent, path: string): void;
  (e: 'drop', event: DragEvent, path: string): void;
  (e: 'commitRename', node: LibraryNode): void;
  (e: 'cancelRename'): void;
  (e: 'setRenameInput', el: HTMLInputElement | null): void;
}>();

function onRenameInput(e: Event) {
  emit('update:renameValue', (e.target as HTMLInputElement).value);
}

function onRenameKey(e: KeyboardEvent) {
  if (e.key === 'Enter') { e.preventDefault(); emit('commitRename', props.node); }
  if (e.key === 'Escape') { e.preventDefault(); emit('cancelRename'); }
}

function onRowClick(e: MouseEvent) {
  e.stopPropagation();
  if (props.node.kind === 'folder') emit('toggleExpand', props.node.path);
  else emit('selectFile', props.node);
}
</script>

<template>
  <div>
    <div
      :class="[
        'group flex items-center gap-1.5 rounded px-2 py-1 text-sm cursor-pointer select-none transition-colors',
        node.kind === 'file' && activeFile === node.path
          ? 'bg-brass-500/10 text-brass-500 font-bold'
          : 'text-text-main hover:bg-black/5 dark:hover:bg-white/5',
        node.kind === 'folder' && dropTarget === node.path ? 'ring-1 ring-brass-500/60 bg-brass-500/10' : '',
      ]"
      :style="{ paddingLeft: (8 + depth * 12) + 'px' }"
      :draggable="renamingPath !== node.path"
      data-testid="library-tree-row"
      :data-path="node.path"
      :data-kind="node.kind"
      @dragstart="emit('dragStart', $event, node)"
      @dragend="emit('dragEnd')"
      @dragover="node.kind === 'folder' ? emit('dragOver', $event, node.path) : undefined"
      @drop="node.kind === 'folder' ? emit('drop', $event, node.path) : undefined"
      @click="onRowClick"
      @contextmenu.prevent="emit('contextMenu', $event, node)"
    >
      <ChevronDown
        v-if="node.kind === 'folder' && isExpanded(node.path)"
        class="w-3 h-3 shrink-0 text-text-muted"
      />
      <ChevronRight
        v-else-if="node.kind === 'folder'"
        class="w-3 h-3 shrink-0 text-text-muted"
      />
      <span v-else class="w-3 shrink-0" aria-hidden="true"></span>

      <FolderOpen
        v-if="node.kind === 'folder' && isExpanded(node.path)"
        class="w-3.5 h-3.5 shrink-0 text-brass-500/70"
      />
      <Folder
        v-else-if="node.kind === 'folder'"
        class="w-3.5 h-3.5 shrink-0 text-brass-500/70"
      />
      <FileText
        v-else
        class="w-3.5 h-3.5 shrink-0 opacity-50 text-text-muted"
      />

      <input
        v-if="renamingPath === node.path"
        :ref="(el: unknown) => emit('setRenameInput', el as HTMLInputElement | null)"
        type="text"
        :value="renameValue"
        class="flex-1 min-w-0 bg-[var(--color-page-bg)] border border-brass-500/40 rounded px-1 py-0 text-xs text-text-main outline-none"
        @input="onRenameInput"
        @keydown="onRenameKey"
        @blur="emit('commitRename', node)"
        @click.stop
      />
      <span v-else class="truncate">{{ node.name }}</span>
    </div>

    <div
      v-if="node.kind === 'folder' && node.children && node.children.length > 0"
      v-show="isExpanded(node.path)"
      class="space-y-0.5"
    >
      <LibraryTreeNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :depth="depth + 1"
        :active-file="activeFile"
        :renaming-path="renamingPath"
        :rename-value="renameValue"
        :drop-target="dropTarget"
        :is-expanded="isExpanded"
        :can-drop-on="canDropOn"
        @update:rename-value="(v) => emit('update:renameValue', v)"
        @toggle-expand="(p) => emit('toggleExpand', p)"
        @select-file="(n) => emit('selectFile', n)"
        @context-menu="(ev, n) => emit('contextMenu', ev, n)"
        @drag-start="(ev, n) => emit('dragStart', ev, n)"
        @drag-end="() => emit('dragEnd')"
        @drag-over="(ev, p) => emit('dragOver', ev, p)"
        @drop="(ev, p) => emit('drop', ev, p)"
        @commit-rename="(n) => emit('commitRename', n)"
        @cancel-rename="() => emit('cancelRename')"
        @set-rename-input="(el) => emit('setRenameInput', el)"
      />
    </div>
  </div>
</template>
