## Scope

These instructions apply to `web/src/desktop/`.

## Architecture

This directory contains the desktop-only frontend layer for the Wails app.
It is separate from the shared editor core in `web/src/core/`.

- `types.ts` — `ProjectEnv`, `DesktopCapabilities` interfaces, `ProjectFile`
  and `Revision` types. `DesktopCapabilities` extends `EditorEnv` (shared)
  with `ProjectEnv` (desktop-only).
- `workspace.ts` — `Workspace` class with reactive state for project
  files, active file, revisions, sidebar, **spell allowlist**,
  **`isLoadingFile`**, and **revision-view state** (`viewingRevisionHash`,
  `viewingRevisionContent`, `viewingRevisionMeta`). This is the desktop
  equivalent of the web's draft-based state management.
- `flush-save.ts` — a tiny registry so the desktop-only Wails
  `before-close` listener can call `AppDesktop.vue`'s flush function
  without the Vue component having to import the Wails runtime directly.
  This lets tests mount `AppDesktop.vue` without generated Wails bindings
  being present.

The Wails bridge class (`WailsBridge` in `desktop-app.ts`) implements
`DesktopCapabilities`. `AppDesktop.vue` creates both a shared `Store`
(theme, shared editor behavior) and a `Workspace` (project state).

## Shared vs Desktop Boundary

- **`web/src/core/`** is shared between web and desktop. It must not
  import from `web/src/desktop/` or reference project/workspace concepts.
- **`web/src/desktop/`** imports from `web/src/core/` (for `EditorEnv`)
  but not the other way around.
- **`EditorEnv`** is the shared interface that `Editor.vue` depends on.
  Do not add desktop-only methods to it. Desktop capabilities go in
  `ProjectEnv`.
- The shared `Store` manages drafts and theme. The desktop `Workspace`
  manages project files, active file, revisions, and allowlist. Do not
  merge these.
- The web app's draft system (`loadDrafts`, `saveDrafts`, etc.) is not
  used by desktop. `WailsBridge` returns empty/no-op for draft methods.
  Desktop persistence is file-based via `ProjectEnv`.

## `documentKey` Contract

`Editor.vue` takes a `documentKey: string | null` prop. This is the
shared-editor's mechanism for resetting per-document transient state
(diagnostics, search highlights, stats, outline, V1-modal suppression)
when the host swaps the active document. The implementation lives in
`web/src/core/useDocumentLifecycle.ts`.

- The web host passes `store.state.activeDraftId`.
- The desktop host passes `workspace.state.activeFile`.

**Do not** key `Editor.vue` reset logic off `store.state.activeDraftId`
directly — that leaks the draft-system concept into the shared component
and was the root cause of the desktop file-switch state-bleed bug.

V1-modal dismissal is also keyed off `documentKey` (via the composable),
so switching documents automatically invalidates a prior dismissal.

## Auto-Save and Flush Contract

- `AppDesktop.vue` auto-saves to disk via a debounced watcher (1s delay).
- Git snapshots are explicit user actions ("Save Version" button), not
  tied to auto-save. Do not add auto-commit.
- `flushSave()` is **async** and must be awaited. It clears the pending
  timer and awaits the in-flight `writeProjectFile`. Every callsite that
  transitions state in a way that could clobber `workspace.state.activeFile`
  or the project root MUST `await flushSave()` first:
  - folder switch (`handleOpenFolder`) — flush before
    `workspace.openFolder` clears `activeFile`
  - file switch (`selectProjectFile`)
  - snapshot (`handleSnapshot`) — the commit must see just-written contents
  - export (`handleExport`) — kept for contract consistency
  - new file (`handleNewPlay`)
  - component unmount (`onUnmounted`)
- The window-close path is handled by the Go `OnBeforeClose` hook and the
  `downstage:before-close` event handshake (see `internal/desktop/AGENTS.md`
  for the backend side). `AppDesktop.vue` registers its `flushSave` via
  `registerFlushSave` on mount; `desktop-app.ts` subscribes to the event
  and invokes it. Don't rely on `onUnmounted` firing on window-quit —
  browsers/webviews don't guarantee it.

## Revision View and Restore

- Clicking an entry in the Versions sidebar puts the editor in **revision
  view mode**: `workspace.state.viewingRevisionHash` is set, the banner
  appears above `Editor.vue`, and the editor is rendered read-only
  (`readOnly` prop on the shared editor). `activeFile` is unchanged so the
  revisions list and file context stay correct.
- While viewing, `AppDesktop.vue`'s `editorContent` computed routes the
  revision content into the editor; the live buffer (`activeContent`) is
  preserved. Copy and Export use `editorContent` so they operate on what
  the user is actually seeing. Save Version is hidden in revision-view
  mode.
- `documentKey` passed to `Editor.vue` is `${activeFile}@${hash}` while
  viewing a revision so the shared editor resets transient state
  (diagnostics, search, stats, outline, V1-modal suppression) when
  toggling in and out of view.
- **Restore is a "revert, not a rollback."** `Workspace.restoreRevision`
  writes the live buffer to disk → snapshots it as "Auto-save before
  restore" (tolerating `nothing-to-snapshot` when HEAD already matches) →
  writes the revision content to disk → snapshots that as "Restore
  version ${short}". Both the pre-state and post-state are now in git, so
  "undo the restore" is itself a one-click restore of the backup commit.
- `clearRevisionView()` is called on file switch, project switch, and after
  a successful restore so stale preview state never leaks across contexts.

## Spellcheck

- The desktop spellcheck dictionary is stored at `.downstage/dictionary.txt`
  in the project directory, managed by the Go backend.
- `Workspace` owns the in-memory copy as `state.spellAllowlist`. Call
  `workspace.addAllowlistWord` / `workspace.removeAllowlistWord` — not
  the env directly — so the reactive state stays in sync.
- The allowlist is refreshed on project switch (`openFolder`).

## Validation

- `npm run typecheck` must pass.
- `npm run test` must pass (vitest). New desktop tests live under
  `web/src/__tests__/desktop/`.
- `npm run build` must pass. This is the web build only — it no longer
  compiles `src/desktop-app.ts` since the vite split in batch 1.
- The desktop bundle is produced by the Wails toolchain, not by calling
  the npm script directly. Use `make desktop-build` (which invokes
  `wails build`, which generates `web/src/wailsjs/` and then internally
  calls `npm run build:desktop`). Running `npm run build:desktop`
  standalone on a clean checkout will fail on the unresolved
  `./wailsjs/go/desktop/App` import — this is expected and matches the
  Wails workflow.
