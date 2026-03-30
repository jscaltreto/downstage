![Downstage](images/icon.png)

# Downstage for VS Code

Downstage is a VS Code extension for writing stage plays in
[Downstage](https://jscaltreto.github.io/downstage/) markup. It provides full
LSP support, live PDF preview, render commands, snippets, diagnostics, and
TextMate syntax highlighting.

## Quick Start

1. Install this extension from the VS Code Marketplace.
2. Install the [`downstage`](https://github.com/jscaltreto/downstage) binary
   and ensure it is on your `PATH`.
3. Open a `.ds` file and start writing.

## Features

### Language Server

The extension communicates with the Downstage language server (`downstage lsp`)
to provide:

- Context-aware character cue completions
- Structural heading completions
- Diagnostics surfaced in the Problems panel
- Folding for title pages, sections, songs, and block comments

### Live Preview

Open a real-time PDF preview inside VS Code that updates as you type.
The debounce delay is configurable.

### PDF Rendering

Render the current script to a standard or compact PDF directly from the
command palette. The generated file opens automatically (configurable).

### Snippets

Pre-built snippets for common Downstage constructs let you scaffold a play
skeleton, add acts, scenes, cues, stage directions, and songs with a few
keystrokes.

### Syntax Highlighting

A TextMate grammar provides highlighting for all Downstage constructs:
metadata, headings, stage directions, songs, parentheticals, character cues,
aliases, verse, and comments.

## Commands

| Command | Description |
| --- | --- |
| Downstage: Restart Language Server | Restart the LSP connection |
| Downstage: Render Current Script | Render to standard PDF |
| Downstage: Render Compact Script | Render to compact PDF |
| Downstage: Preview Current Script PDF | Preview standard PDF in VS Code |
| Downstage: Preview Compact Script PDF | Preview compact PDF in VS Code |
| Downstage: Live Preview | Live-updating PDF preview |

## Snippets

| Prefix | Description |
| --- | --- |
| `play` | Full play skeleton with title page, cast, act, and scene |
| `act` | Act heading |
| `scene` | Scene heading |
| `cue` | Character cue with dialogue |
| `stage` | Stage direction |
| `song` | Song block |

## Settings

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `downstage.server.path` | string | `"downstage"` | Path to the `downstage` executable |
| `downstage.server.trace` | string | `"off"` | LSP trace verbosity (`off` / `messages` / `verbose`) |
| `downstage.editor.autoSuggestCharacterCues` | boolean | `true` | Auto-trigger cue suggestions on empty lines |
| `downstage.render.style` | string | `"standard"` | Render style (`standard` / `condensed`) |
| `downstage.render.openAfterRender` | boolean | `true` | Open PDF after rendering |
| `downstage.preview.debounceMs` | number | `300` | Delay before re-rendering live preview (ms) |

## Requirements

The `downstage` binary must be installed and available on your `PATH`, or
configured via the `downstage.server.path` setting.

## Related

- [Downstage documentation](https://jscaltreto.github.io/downstage/)
- [GitHub repository](https://github.com/jscaltreto/downstage)

## Development

```bash
cd editors/vscode
npm install
npm run compile
npm run package
```

Press `F5` in VS Code to launch an Extension Development Host.
