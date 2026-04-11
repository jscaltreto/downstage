# Downstage Web Editor

A browser-based Downstage editor with live preview, syntax highlighting,
browser-local draft storage, an Open Draft picker, and PDF export. The entire parsing and
rendering pipeline runs client-side via
WebAssembly — no server required.

## Quick Start

```bash
# From the repository root:
npm install
npm --prefix web install
make web
make web-dev
# Open http://localhost:5173/editor/
```

## Tech Stack

- **Vue 3**: Component-based UI shell
- **Tailwind CSS v4**: Adaptive theatrical design system (Dark/Light mode)
- **CodeMirror 6**: Plaintext editor core
- **Lucide**: Consistent iconography
- **WebAssembly**: Go-powered parsing and rendering

## Building

### Prerequisites

- Go 1.24+ (for WASM compilation)
- Node.js 22+ and npm (for bundling)

### Build Steps

```bash
# Build the WASM runtime and editor bundle
make web

# Or manually:
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/build/downstage.wasm ./cmd/wasm/
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/build/
cd web && npm install && npm run build
```

### Development

```bash
# Start the editor dev server after rebuilding WASM:
make web-dev
```

After Go changes, rebuild with `make wasm` and refresh the browser.

## Architecture

```
web/src/core/       → Pure TypeScript logic (Store, Engine, Types)
web/src/components/ → Vue Single File Components
web/src/web-app.ts  → Web-specific entry point and environment implementation
web/src/wasm.ts     → TypeScript bindings for WASM functions
```

All parsing and rendering happens in Go via WASM. The TypeScript core handles
the editor lifecycle, while Vue handles the presentation layer.

### WASM API

The WASM module exposes a global `downstage` object:

| Function | Input | Output |
|----------|-------|--------|
| `parse(source)` | Downstage source string | `{errors: [{message, line, col, endLine, endCol}]}` |
| `diagnostics(source)` | Downstage source string | `{diagnostics: [{message, severity, line, col, endLine, endCol, code?}]}` |
| `renderHTML(source, style?)` | Source + optional style (`"standard"`/Manuscript or `"condensed"`/Acting Edition) | HTML string |
| `renderPDF(source, style?)` | Source + optional style | `Uint8Array` (PDF bytes) |
| `semanticTokens(source)` | Source string | `Uint32Array` (delta-encoded LSP tokens) |
| `tokenTypeNames` | — | `string[]` (token type legend) |
