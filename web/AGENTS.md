## Scope

These instructions apply to the `web/` subtree (the browser-based Downstage editor).

## Architecture

The web editor is a Vue 3 application that loads a Go WASM binary (`downstage.wasm`)
compiled from `cmd/wasm/main.go`. 

- **Pure Logic Core**: All parsing, rendering, and editor engine logic is in pure TypeScript (`web/src/core/`).
- **Vue 3**: Used for the presentation layer (toolbars, modals, workspace shell).
- **Tailwind CSS v4**: Utility-first styling with dual-theme support.
- **CodeMirror 6**: The plaintext editor component.
- **WASM Bridge**: The single source of truth for parsing and rendering.

## Change Boundaries

- Do not reimplement parsing or document semantics in TypeScript. Use the WASM
  bridge.
- Keep business logic in `web/src/core/`. Vue components should focus on
  presentation and state synchronization via the `Store`.
- Leverage Tailwind utility classes for styling.
- Do not check in `node_modules/`, `dist/`, or `*.wasm` files.

## Preferences

- Editor UI preferences (theme, previewHidden, spellcheckDisabled) round-trip
  through `env.getEditorPreferences` / `env.setEditorPreferences`. Both the
  browser env (`WebEnv` in `web-app.ts`) and the desktop env (`WailsBridge`
  in `desktop-app.ts`) implement this pair — the web impl writes a single
  `downstage-editor-prefs` localStorage blob; the desktop impl round-trips
  the Go `Config.Preferences` struct. `Store` consumes the env methods and
  is the single owner of pref state on both hosts.
- `Editor.vue` takes `previewHidden` and `spellcheckDisabled` as v-model
  props and must not touch any storage directly. Adding a new persisted
  pref means: field on `EditorPreferences` (and the Go `Preferences` struct
  mirror), state on `Store`, v-model on `Editor.vue`.
- The parser for the web blob is `parseEditorPreferencesBlob` in
  `web/src/core/editor-prefs.ts`. It is the single defensive boundary —
  corrupt JSON, unknown theme strings, and wrong-typed fields fall back to
  defaults without throwing.

## Validation

- `npm run build` must succeed after any TypeScript or Vue changes.
- WASM changes require rebuilding via `make wasm` and manual browser testing.
- Run `make web-e2e` when touching UI, editor integration, or WASM bindings.
  New contributors need a one-time `npx --prefix web playwright install chromium`
  before the E2E suite will run.
