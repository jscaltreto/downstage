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

## Validation

- `npm run build` must succeed after any TypeScript or Vue changes.
- WASM changes require rebuilding via `make wasm` and manual browser testing.
- Run `make web-e2e` when touching UI, editor integration, or WASM bindings.
  New contributors need a one-time `npx --prefix web playwright install chromium`
  before the E2E suite will run.
