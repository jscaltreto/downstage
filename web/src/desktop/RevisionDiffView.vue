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
    <!-- Pin the merge view to all four sides via absolute positioning
         inside a relative flex-1 parent. Mirrors the main Editor's
         pattern (Editor.vue's editorContainer). CodeMirror sizes its
         editors to content by default; without an absolute container
         the panes collapse to a content-height void instead of filling
         the editor area. -->
    <div class="relative flex-1 min-h-0">
      <div ref="hostEl" class="revision-diff-host absolute inset-0 overflow-hidden"></div>
    </div>
  </div>
</template>

<style scoped>
/* @codemirror/merge's design assumes .cm-mergeView is the scrolling
   container with inner editors auto-sized to content. Its baseTheme
   forces `.cm-scroller, .cm-editor { height: auto !important }` inside
   .cm-mergeView (see node_modules/@codemirror/merge/dist/index.cjs
   around the externalTheme/baseTheme exports). Without explicit
   overrides, the editors collapse to content-height and leave a void
   below short documents. We re-impose a top-down height chain so the
   editor canvas paints the full available area. */
.revision-diff-host {
  display: flex;
  flex-direction: column;
}
.revision-diff-host :deep(.cm-mergeView) {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  /* The default `overflow-y: auto` would let the merge view itself
     scroll; we let each side's .cm-scroller handle scroll instead so
     the two panes stay vertically aligned independent of each other. */
  overflow: hidden;
}
.revision-diff-host :deep(.cm-mergeViewEditors) {
  flex: 1 1 0;
  min-height: 0;
  /* `display: flex` is the package default but is re-stated here so
     editors stretch when @codemirror/merge's externalTheme isn't
     applied (e.g., before the first measure). */
  display: flex;
  align-items: stretch;
}
.revision-diff-host :deep(.cm-mergeViewEditor) {
  /* Package defaults flex-grow:1 + flex-basis:0; nothing to add — but
     pin min-width:0 so a long line on one side can't push the column
     wider than its share. */
  min-width: 0;
}
/* Override the baseTheme's `height: auto !important` so editors fill
   the merge view's height instead of collapsing to content. The
   !important is required because the package uses it. */
.revision-diff-host :deep(.cm-mergeView .cm-editor) {
  height: 100% !important;
}
.revision-diff-host :deep(.cm-mergeView .cm-scroller) {
  height: 100% !important;
  overflow: auto !important;
}
/* The content layer (where text and gutters paint) is still content-
   sized inside .cm-scroller; min-height makes it claim the full
   scroller height so the editor's background extends past the last
   line for short documents. */
.revision-diff-host :deep(.cm-content) {
  min-height: 100%;
}
</style>
