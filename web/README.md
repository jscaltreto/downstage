# Downstage Web Editor

A browser-based Downstage editor with live preview, syntax highlighting, and
PDF export. The entire parsing and rendering pipeline runs client-side via
WebAssembly — no server required.

## Quick Start

```bash
# From the repository root:
make web
make web-dev
# Open http://localhost:8080
```

## Building

### Prerequisites

- Go 1.23+ (for WASM compilation)
- Node.js 18+ and npm (for bundling)

### Build Steps

```bash
# Build the WASM binary
make wasm

# Bundle the TypeScript
make web

# Or manually:
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/dist/downstage.wasm ./cmd/wasm/
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/dist/
cd web && npm install && npm run build
```

### Development

```bash
# Watch mode for TypeScript changes:
cd web && npm run watch

# In another terminal, serve the files:
make web-dev
```

After Go changes, rebuild with `make wasm` and refresh the browser.

## Architecture

```
cmd/wasm/main.go    → WASM bridge (syscall/js), exposes 4 functions
web/src/main.ts     → Entry point, WASM init, CodeMirror setup
web/src/wasm.ts     → TypeScript bindings for WASM functions
web/src/downstage-lang.ts → Syntax highlighting via WASM semantic tokens
web/src/diagnostics.ts    → Error markers via WASM parser
web/src/preview.ts  → Live HTML preview (debounced)
web/src/pdf-export.ts     → PDF download via WASM renderer
```

All parsing and rendering happens in Go via WASM. The TypeScript code handles
only the editor UI and WASM glue.

### WASM API

The WASM module exposes a global `downstage` object:

| Function | Input | Output |
|----------|-------|--------|
| `parse(source)` | Downstage source string | `{errors: [{message, line, col, endLine, endCol}]}` |
| `renderHTML(source, style?)` | Source + optional style (`"standard"` or `"condensed"`) | HTML string |
| `renderPDF(source, style?)` | Source + optional style | `Uint8Array` (PDF bytes) |
| `semanticTokens(source)` | Source string | `Uint32Array` (delta-encoded LSP tokens) |
| `tokenTypeNames` | — | `string[]` (token type legend) |

### Binary Size

The WASM binary is ~8.5 MB uncompressed due to embedded fonts (Courier Prime +
Libre Baskerville) and the Go runtime. With gzip compression (~3 MB) or brotli
(~2.5 MB), this is acceptable for a tool that loads once per session. Configure
your web server to serve `.wasm` files with compression.
