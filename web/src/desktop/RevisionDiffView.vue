<script setup lang="ts">
// Side-by-side diff between a historical revision and the live editing
// buffer. Used by the desktop revision banner's "Compare to current"
// toggle. Both panes are read-only; this is a reading tool, not an
// editing surface.
//
// Lifecycle (see web/src/desktop/AGENTS.md, Revision View and Restore):
//   * onMounted constructs the MergeView against hostEl.
//   * Watching props.original / props.modified destroys + recreates the
//     MergeView so a sidebar click on a different revision rebuilds
//     against fresh content.
//   * Watching props.isDark reconfigures the theme compartment on each
//     underlying EditorView in place, avoiding a full-pane flash on
//     light/dark toggle.
//   * onUnmounted destroys the MergeView.
import { onMounted, onUnmounted, ref, watch } from "vue";
import { EditorState, Compartment } from "@codemirror/state";
import { EditorView, lineNumbers } from "@codemirror/view";
import { oneDark } from "@codemirror/theme-one-dark";
import { MergeView } from "@codemirror/merge";
import { lightTheme } from "../core/engine";
import { createDownstageHighlighter } from "../downstage-lang";
import type { EditorEnv } from "../core/types";

const props = defineProps<{
  original: string;
  modified: string;
  originalLabel: string;
  modifiedLabel: string;
  isDark: boolean;
  env: EditorEnv;
}>();

const hostEl = ref<HTMLDivElement | null>(null);
let view: MergeView | null = null;
const themeCompartmentA = new Compartment();
const themeCompartmentB = new Compartment();

function buildExtensions(themeCompartment: Compartment) {
  return [
    lineNumbers(),
    EditorState.readOnly.of(true),
    EditorView.editable.of(false),
    themeCompartment.of(props.isDark ? oneDark : lightTheme),
    createDownstageHighlighter(props.env),
    EditorView.lineWrapping,
  ];
}

function createMergeView() {
  if (!hostEl.value) return;
  view = new MergeView({
    parent: hostEl.value,
    a: {
      doc: props.original,
      extensions: buildExtensions(themeCompartmentA),
    },
    b: {
      doc: props.modified,
      extensions: buildExtensions(themeCompartmentB),
    },
    // Default orientation a-b puts a (historical) on the left and b
    // (current) on the right — matches GitHub / VS Code convention.
    highlightChanges: true,
    gutter: true,
  });
}

function destroyMergeView() {
  view?.destroy();
  view = null;
}

onMounted(() => {
  createMergeView();
});

onUnmounted(() => {
  destroyMergeView();
});

// Revision change → rebuild. For play-length docs the destroy/recreate
// round-trip is cheap and avoids edge cases with mid-diff doc
// replacement. The user-visible cost is a single re-layout when they
// click a different revision in the sidebar.
watch(
  () => [props.original, props.modified],
  () => {
    destroyMergeView();
    createMergeView();
  },
);

// Theme toggle → reconfigure both panes in place. A full rebuild here
// would flash the diff during a normal dark-mode toggle, which is much
// more noticeable than a revision switch (the latter at least swaps
// content).
watch(
  () => props.isDark,
  (isDark) => {
    if (!view) return;
    const next = isDark ? oneDark : lightTheme;
    view.a.dispatch({ effects: themeCompartmentA.reconfigure(next) });
    view.b.dispatch({ effects: themeCompartmentB.reconfigure(next) });
  },
);
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden">
    <div class="flex shrink-0 border-b border-black/10 dark:border-white/10 bg-[var(--color-page-bg)]">
      <div class="flex-1 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[0.18em] text-text-muted border-r border-black/10 dark:border-white/10">
        {{ originalLabel }}
      </div>
      <div class="flex-1 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[0.18em] text-text-muted">
        {{ modifiedLabel }}
      </div>
    </div>
    <div ref="hostEl" class="revision-diff-host flex-1 overflow-auto"></div>
  </div>
</template>

<style scoped>
.revision-diff-host :deep(.cm-mergeView) {
  height: 100%;
}
.revision-diff-host :deep(.cm-mergeViewEditors) {
  height: 100%;
}
.revision-diff-host :deep(.cm-editor) {
  height: 100%;
}
.revision-diff-host :deep(.cm-scroller) {
  overflow: auto;
}
</style>
