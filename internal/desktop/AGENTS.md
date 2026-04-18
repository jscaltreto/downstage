## Scope

These instructions apply to `internal/desktop/`.

## Architecture

This package is the Go backend for the Wails-based desktop app ("Downstage
Write"). It exposes methods that the Vue frontend calls via Wails bindings.
All methods live on a single `*App` struct, split across focused files:

- `app.go` — lifecycle, config persistence (including legacy-config
  migration), `safePath` validation, the `OnBeforeClose` hook.
- `language.go` — parser/LSP bridge (parse, diagnostics, completions, etc.).
- `library.go` — file CRUD, library location management, spellcheck dictionary.
- `render.go` — HTML/PDF rendering and save dialogs.
- `git.go` — explicit git snapshotting and revision history.
- `helpers.go` — shared utilities.

## Critical Rules

- **Path safety is non-negotiable.** Every method that accepts a relative
  file path from the frontend must call `safePath()` before touching the
  filesystem. `safePath` rejects:
  - absolute inputs
  - **any leaf symlink**, whether the target is live, dangling, inside, or
    outside the library (this closes a class of TOCTOU / dangling-leaf
    escapes that the previous parent-only check allowed)
  - live symlink chains whose final target escapes the library root

  Tests in `app_test.go` cover each of these explicitly — do not relax the
  leaf-symlink rule without replacing those tests with something that
  provides equivalent coverage.

- **File writes and git commits are separate operations.** `WriteLibraryFile`
  writes to disk only. `SnapshotFile` stages and commits. The frontend
  auto-saves on a debounce timer; git snapshots are explicit user actions.
  Do not re-couple these.

- **`ReadFileAtRevision` is read-only.** It resolves a revision, walks the
  tree, and returns the blob content. It must not mutate state, touch the
  working copy, or create a new commit. The restore flow is orchestrated
  by the frontend across `WriteLibraryFile` + `SnapshotFile` calls.

- **Do not add auto-commit behavior.** Writers auto-save constantly.
  Committing on every save produces useless git history. Snapshots should
  be deliberate.

- **Propagate errors. Do not swallow them.** Methods that represent
  user-visible actions return `(T, error)` so the frontend can toast real
  failures. Early "success-with-nil" returns (for things like "no library
  open") are a bug — return an error and let the UI present it.
  `SnapshotFile` uses `ErrNothingToSnapshot` as a typed sentinel when the
  worktree is clean after staging; the frontend matches on the message
  prefix `"downstage: nothing-to-snapshot"` to show an informational toast
  instead of an error.

- **Config is written explicitly, not on hot paths.** `ReadLibraryFile`
  must not touch the config. Use `SetActiveLibraryFile` from the frontend
  on file selection; library switches mutate only the library fields
  (see `ChangeLibraryLocation`).
- **Legacy config fields migrate via `migrateLegacyConfig`.** The
  pre-rename JSON keys (`lastProjectPath`, `lastActiveProjectFile`) are
  read by `readConfigLocked` and copied into the current-name fields
  whenever the latter are empty. Both legacy keys must migrate together —
  the active file only makes sense in the context of its library.
- **All config writers must go through `updateConfig(func(*Config))`.**
  `updateConfig` holds `configMu` across the full read-modify-write so
  independent subtree writers (Preferences, WindowState,
  LastActiveLibraryFile) can't drop each other's changes. `readConfig`
  and `writeConfig` still exist for external atomic reads and verbatim
  writes (tests), but any in-process RMW must use `updateConfig` —
  otherwise the mutex releases between read and write and two writers
  race. The frontend `prefs-cache` provides a complementary atomicity
  layer for Preferences writes arriving from Store + Workspace.

- **Preferences are the single source of truth for persisted UI state.**
  All desktop-side UI preferences (theme, previewHidden, spellcheckDisabled,
  sidebarCollapsed) live in the nested `Preferences` struct inside
  `Config`. They are exposed via exactly one bound pair:
  `GetPreferences()` and `SetPreferences(prefs)`. Writers always read,
  mutate, and write the full struct; there are no per-field setters by
  design. `GetPreferences` normalizes an empty `Theme` to `"system"` so
  callers never have to know which fields carry sentinels. Do not
  introduce per-field Wails bindings for new preferences — add fields to
  `Preferences` and use the existing pair.

- **Do not reintroduce `localStorage` on the desktop side.** The webview
  has `localStorage`, but everything persisted on desktop must round-trip
  through `Preferences`. If a preference needs to survive restart, it
  belongs in the Go config, not in browser storage.

- **The command catalog in `commands.go` is the single source of truth
  for app-level command metadata.** Every menu label, accelerator, and
  category lives there. `menu.go` builds the native `*menu.Menu` from
  the catalog; `App.GetCommands()` exposes the palette-facing projection
  to the frontend. The frontend's `commands.ts` is a flat
  `Map<id, handler>` — no labels, no accelerators. Adding a command
  is exactly one entry in each file. Changing a label or accelerator
  is a Go-only edit; the frontend never declares those. Do not let
  label/accelerator strings drift into TS.

- **Menu clicks emit `command:execute`.** Each catalog item's Click
  callback publishes the ID on the runtime event bus; the frontend's
  single `EventsOn` subscriber dispatches through its command dispatcher.
  `App.SetDisabledCommands(ids)` rebuilds the menu with `Disabled: true`
  flags for the listed IDs and calls `MenuUpdateApplicationMenu`. The
  frontend dispatcher diffs against its last-sent set before calling,
  so a stable disabled set produces zero wire traffic.

- **Git commit authorship respects the user's global identity.**
  `snapshotAuthor` reads `config.GlobalScope` (merging `~/.gitconfig` and
  `$XDG_CONFIG_HOME/git/config`) and falls back to `Downstage Write
  <hello@getdownstage.com>` only when no identity is configured.

## Shared vs Desktop Boundary

- Parsing, diagnostics, completions, rendering, and all document semantics
  come from `internal/parser`, `internal/lsp`, `internal/render`, etc. This
  package is a thin bridge — it calls those packages and reshapes results
  for Wails JSON serialization. Do not add document logic here.
- The spellcheck dictionary lives at `.downstage/dictionary.txt` in the
  library root. This is a library-level resource, not a per-file or
  per-draft concept.

## `OnBeforeClose` Event Contract

The Wails `OnBeforeClose` hook (`App.BeforeClose`) is how the backend makes
the frontend's debounced auto-save reach disk on window close. The
handshake is a pair of runtime events:

- `downstage:before-close` — Go → frontend. Emitted by `BeforeClose`.
- `downstage:flush-complete` — frontend → Go. Emitted by the frontend's
  `before-close` handler once any pending `flushSave` completes.

Ordering note: `EventsOnce` is registered **before** `EventsEmit` so a fast
frontend cannot reply before Go subscribes. A 2-second timeout bounds the
wait — a broken frontend must not lock the window closed.

The frontend side of this contract lives in `web/src/desktop-app.ts` (the
listener wraps the flush in try/finally so a flush error still releases
the close).

## Generated Files

- Wails generates TypeScript bindings in `web/src/wailsjs/`. These are
  `.gitignored` and must not be committed. They are regenerated by
  `wails dev` and `wails build`.

## Testing

- Run `go test ./internal/desktop/ -race` after changes here. The `-race`
  flag matters because `configMu` guards concurrent reads and writes.
- Security-sensitive tests in `app_test.go` cover: path traversal,
  absolute-path rejection, live symlink escape, **dangling symlink leaf
  (inside and outside root)**, and live leaf symlink. Keep coverage for
  each.
- Snapshot tests cover: no-library error, clean-worktree sentinel, user
  git identity, fallback identity.
- Config tests cover: clear semantics, library-switch clears the active
  file, `ReadLibraryFile` no longer touches config, legacy config
  migration (pre-rename `lastProjectPath`/`lastActiveProjectFile`
  carry across).

## Validation

- `go test ./...` must pass.
- `go vet ./internal/desktop/` must be clean.
- `gofmt` must produce no diff.
