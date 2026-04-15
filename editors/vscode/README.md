# Downstage for VS Code

Downstage turns VS Code into a plain-text playwriting desk. Start a new play,
watch pages update in live preview, and export a clean PDF manuscript or
acting edition when you are ready.

## Quick Start

1. Install this extension from the VS Code Marketplace.
2. Run `Downstage: New Play`.
3. Write your title and first scene.
4. Run `Downstage: Open Live Preview` to see pages update as you work.
5. Run `Downstage: Export Manuscript PDF` or `Downstage: Export Acting Edition PDF` when you want a shareable file.

You do not need to start in the terminal. The extension opens a start guide on
first launch with direct actions for `New Play`, `Open Sample Play`, and
`Open Live Preview`.

## Features

### Start Writing Fast

- `Downstage: New Play` opens an untitled script with a ready-to-edit play
  skeleton.
- `Downstage: Open Sample Play` opens a fuller example so you can see the
  format in action.
- The cursor lands in the title field so you can start typing immediately.

### Live manuscript preview

Open a live manuscript preview inside VS Code while you write. If you launch
preview before opening a script, Downstage creates a new play first so you are
not blocked on file setup.

### PDF export

Export the current script to either a Manuscript PDF or an Acting Edition PDF
from the command palette. The generated file opens automatically unless you
turn that off.

### Writing Help

Downstage includes:

- Character cue suggestions in scene context
- Structure-aware headings and folding
- Diagnostics in the Problems panel
- Rename Symbol on a character name (F2) updates the dramatis personae
  entry and every matching cue at once
- Snippets for acts, scenes, cues, stage directions, and songs

### Snippets for common structures

Type one of these snippet prefixes inside a Downstage document:

- `play` for a full play skeleton
- `act` for an act heading
- `scene` for a scene heading
- `cue` for a character cue with dialogue
- `stage` for a stage direction
- `song` for a song block

### Syntax highlighting

Highlighting covers metadata, headings, stage directions, songs,
parentheticals, character cues, aliases, verse, and comments.

## Commands

| Command | What it does |
| --- | --- |
| Downstage: New Play | Open a new play with a starter script |
| Downstage: Open Sample Play | Open a richer example play |
| Downstage: Open Start Guide | Reopen the first-run guide |
| Downstage: Open Help | Open Downstage help in your browser |
| Downstage: Restart Downstage | Restart the Downstage background services |
| Downstage: Export Manuscript PDF | Export the current script as a manuscript PDF |
| Downstage: Export Acting Edition PDF | Export the current script as an acting edition PDF |
| Downstage: Open Manuscript PDF Preview | Preview the manuscript PDF in VS Code |
| Downstage: Open Acting Edition PDF Preview | Preview the acting edition PDF in VS Code |
| Downstage: Open Live Preview | Open the live-updating manuscript preview |

## Settings

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `downstage.server.path` | string | `""` | Optional explicit path to the `downstage` app. When empty, the extension first tries its bundled copy and then tries your system path |
| `downstage.server.trace` | string | `"off"` | Extra diagnostic logging for troubleshooting |
| `downstage.editor.autoSuggestCharacterCues` | boolean | `true` | Auto-trigger cue suggestions on empty lines |
| `downstage.render.style` | string | `"standard"` | Default export style. `standard` means Manuscript. `condensed` means Acting Edition |
| `downstage.render.openAfterRender` | boolean | `true` | Open PDF after rendering |
| `downstage.preview.debounceMs` | number | `300` | Delay before re-rendering live preview (ms) |

## If Downstage Does Not Start

Release builds of the extension bundle Downstage for:

- `linux-x64`
- `darwin-x64`
- `darwin-arm64`
- `win32-x64`

If your platform is not listed, or if you want to use a different binary, set
`downstage.server.path` in VS Code settings. If no bundled binary is present,
the extension also tries `downstage` on your system path.

Most users can ignore this, install the extension, open a script, and start
writing.

Release notes for the extension come from the repository root
[`CHANGELOG.md`](../../CHANGELOG.md).

## Release Process

The VS Code extension ships from this repository. It does not maintain a
separate release track.

- Release tags `v*` drive the extension version used for packaged VSIX files.
- `.github/workflows/release.yml` packages one VSIX per supported platform and
  uploads them to the GitHub release.
- If the `VSCE_PAT` GitHub Actions secret is configured, the same workflow also
  publishes those VSIX files to the Visual Studio Marketplace.
- If the `OVSX_TOKEN` GitHub Actions secret is configured, the workflow also
  publishes the same VSIX files to Open VSX.
- The registries are published independently, so one can fail without blocking
  the other.

## Related

- [Start Here](https://www.getdownstage.com/start/)
- [Downstage Syntax Guide](https://www.getdownstage.com/syntax/)
- [Downstage FAQ](https://www.getdownstage.com/faq/)
- [GitHub repository](https://github.com/jscaltreto/downstage)

## Development

```bash
cd editors/vscode
npm install
npm run compile
npm run package
```

Press `F5` in VS Code to launch an Extension Development Host.
