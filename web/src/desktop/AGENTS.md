## Scope

These instructions apply to `web/src/desktop/`.

## Architecture

This directory contains the desktop-only frontend layer for the Wails app.
It is separate from the shared editor core in `web/src/core/`.

- `types.ts` — `ProjectEnv`, `DesktopCapabilities` interfaces, `ProjectFile`
  and `Revision` types. `DesktopCapabilities` extends `EditorEnv` (shared) with
  `ProjectEnv` (desktop-only).
- `workspace.ts` — `Workspace` class with reactive state for project files,
  active file, revisions, and sidebar. This is the desktop equivalent of the
  web's draft-based state management.

The Wails bridge class (`WailsBridge` in `desktop-app.ts`) implements
`DesktopCapabilities`. `AppDesktop.vue` creates both a shared `Store`
(theme, shared editor behavior) and a `Workspace` (project state).

## Shared vs Desktop Boundary

- **`web/src/core/`** is shared between web and desktop. It must not import
  from `web/src/desktop/` or reference project/workspace concepts.
- **`web/src/desktop/`** imports from `web/src/core/` (for `EditorEnv`) but
  not the other way around.
- **`EditorEnv`** is the shared interface that `Editor.vue` depends on. Do not
  add desktop-only methods to it. Desktop capabilities go in `ProjectEnv`.
- The shared `Store` manages drafts and theme. The desktop `Workspace` manages
  project files, active file, and revisions. Do not merge these.
- The web app's draft system (`loadDrafts`, `saveDrafts`, etc.) is not used by
  desktop. `WailsBridge` returns empty/no-op for draft methods. Desktop
  persistence is file-based via `ProjectEnv`.

## Auto-Save Behavior

- `AppDesktop.vue` auto-saves to disk via a debounced watcher (1s delay).
- Git snapshots are explicit user actions ("Save Version" button), not tied to
  auto-save. Do not add auto-commit.
- When switching files or folders, `flushSave()` must be called before changing
  `activeContent` to avoid losing pending writes.
- Never clear `activeContent` before a folder dialog result is confirmed —
  canceling the dialog must be a no-op.

## Spellcheck

- The desktop spellcheck dictionary is stored at `.downstage/dictionary.txt`
  in the project directory, managed by the Go backend.
- `spellAllowlist` must be refreshed when the project changes (folder switch).

## Validation

- `npm run typecheck` must pass.
- `npm run test` must pass (vitest).
- Changes here should not break the web editor — verify with `npm run build`.
