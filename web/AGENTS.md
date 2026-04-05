## Scope

These instructions apply to the `web/` subtree (the browser-based Downstage editor).

## Architecture

The web editor is a static site that loads a Go WASM binary (`downstage.wasm`)
compiled from `cmd/wasm/main.go`. All parsing, rendering, and semantic analysis
happens in Go via WASM. The JavaScript side handles only UI:

- **CodeMirror 6** for the editor
- **Semantic tokens** from WASM drive syntax highlighting (no Lezer grammar)
- **Live preview** via HTML rendered by WASM, displayed in a sandboxed iframe
- **PDF export** via WASM, downloaded as a Blob

## Change Boundaries

- Do not reimplement parsing or document semantics in TypeScript. The WASM
  bridge is the single source of truth.
- Keep the frontend lightweight. No frameworks — vanilla HTML/CSS/JS with
  CodeMirror as the only substantial dependency.
- Do not check in `node_modules/`, `dist/`, or `*.wasm` files.

## Validation

- `npm run build` must succeed after any TypeScript changes.
- WASM changes require rebuilding via `make wasm` and manual browser testing.
