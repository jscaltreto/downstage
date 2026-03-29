# Downstage for VS Code

This extension provides basic VS Code support for Downstage scripts.

## Current features

- `.ds` language registration
- TextMate highlighting for metadata, headings, stage directions, songs,
  parentheticals, character cues, and aliases
- Language Server Protocol integration via `downstage lsp`
- Context-aware character cue completions from the Go LSP
- Structural heading completions from the Go LSP
- Snippets for play skeletons, acts, scenes, cues, stage directions, and songs
- Improved folding for title pages, headed sections, songs, and block comments
- Automatic cue suggestion on a fresh line after a blank separator
- Commands to render the current script in standard or compact PDF styles
- Commands to preview the generated PDF inside VS Code
- Render parse failures surfaced in the Problems panel for the current file
- Command to restart the language server

## Requirements

The `downstage` binary must be installed and available on your `PATH`, or you
must set `downstage.server.path` to an explicit executable path.

## Settings

- `downstage.server.path`: executable path for `downstage`
- `downstage.server.trace`: LSP trace verbosity
- `downstage.editor.autoSuggestCharacterCues`: auto-open cue suggestions on a
  new empty line after a blank separator
- `downstage.render.style`: render style for the VS Code render command
- `downstage.render.openAfterRender`: open the generated PDF after rendering

## Development

```bash
cd editors/vscode
npm install
npm run compile
npm run package
```

Press `F5` in VS Code to launch an Extension Development Host.
