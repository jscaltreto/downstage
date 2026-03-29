<p align="center">
  <img src="downstage_logo.png" alt="Downstage logo, markdown for playwrights" width="380">
</p>

# Downstage

A plaintext markup language for stage plays.

## What is Downstage?

Downstage is a plaintext markup language for writing stage plays, inspired by
[Fountain](https://fountain.io/) (for screenplays) and the archived
[TheatreScript](https://github.com/contrapunctus-1/TheatreScript) spec. It
ships with an LSP server for editor integration and CLI tools for parsing,
validation, and PDF rendering. Files use the `.ds` extension.

## Quick Example

```
Title: The Example Play
Author: Jane Smith
Date: 2025

# Dramatis Personae

HAMLET — Prince of Denmark
HORATIO — Friend to Hamlet

## Courtiers

ROSENCRANTZ — A courtier
GUILDENSTERN — A courtier

# The Example Play

## ACT I

### SCENE 1

> The battlements of Elsinore Castle. Night.

HORATIO
Who's there?

HAMLET
(aside)
A piece of work is man, how **noble** in reason,
how *infinite* in faculty.
  In form and moving, how express
  and admirable; in action, how like
  an angel.

// A line comment

> Enter GHOST

===

### SCENE 2

ROSENCRANTZ
Good my lord!

SONG 1: The Wanderer's Lament

HAMLET
  O, that this too, too solid flesh
  Would melt, thaw, and resolve itself
  Into a dew.

SONG END
```

## Features

- Readable plaintext format — scripts look natural without markup noise
- Inline formatting: `*italic*`, `**bold**`, `***bold italic***`, `_underline_`, `~strikethrough~`
- Verse support via indentation (2+ spaces)
- Songs with `SONG`/`SONG END` blocks
- Comments: `// line` and `/* block */`
- Forced elements: `@character` and `.heading` for edge cases
- Character aliases: `HAMLET/HAM` or `[HAMLET/HAM]`
- Page breaks: `===`
- LSP server with:
  - Semantic syntax highlighting
  - Document outline (acts/scenes/characters)
  - Hover info on character names (shows description from dramatis personae)
  - Go-to-definition (jump to character's dramatis personae entry)
  - Diagnostics (parse errors, unknown character warnings)
- Neovim integration out of the box (0.11+)
- CLI tools for parsing and validation

## Installation

```
go install github.com/jscaltreto/downstage@latest
```

### Homebrew

```
brew tap jscaltreto/tap
brew install downstage
```

## CLI Usage

```
downstage parse play.ds       # Output AST as JSON
downstage validate play.ds    # Check for errors
downstage render play.ds      # Render a PDF manuscript
downstage lsp                 # Start LSP server (stdio)
downstage version             # Print version info
```

Use `-v` or `--verbose` to enable debug logging on any command.

`downstage validate` exits non-zero on parse errors. `downstage parse` prints
parse errors to stderr but still emits the AST JSON. `downstage render` exits
non-zero if parsing fails.

## Editor Setup

### Neovim (0.11+)

Copy the contents of `editors/neovim/` into your Neovim config directory
(typically `~/.config/nvim/`). This adds three files:

- `ftdetect/downstage.lua` — registers the `.ds` filetype
- `ftplugin/downstage.lua` — buffer settings (soft wrap, spell check, comment format)
- `plugin/downstage.lua` — LSP client configuration

The LSP config:

```lua
vim.lsp.config("downstage", {
    cmd = { "downstage", "lsp" },
    filetypes = { "downstage" },
    root_markers = { ".git" },
})
vim.lsp.enable("downstage")
```

The `downstage` binary must be on your `PATH`.

### Other Editors

Any LSP-compatible editor can use the Downstage language server. The server
communicates over stdio using JSON-RPC 2.0 (LSP 3.17). Point your editor's LSP
client at the server:

```json
{
  "command": ["downstage", "lsp"],
  "filetypes": ["downstage"],
  "rootPatterns": [".git"]
}
```

## Language Overview

A Downstage document has three sections:

1. **Title Page** — optional `Key: Value` metadata lines at the start of the document, with indented continuation lines allowed
2. **Dramatis Personae** — A `# Dramatis Personae` heading followed by character names and descriptions, optionally organized into groups with `##` subheadings.
3. **Body** — The play itself: acts (`## ACT`), scenes (`### SCENE`), dialogue (ALL CAPS character name followed by speech text), stage directions (`>` prefixed lines), verse (indented 2+ spaces), songs, and comments.

See [SPEC.md](SPEC.md) for the complete language specification.

## Building from Source

```
git clone https://github.com/jscaltreto/downstage.git
cd downstage
make
```

To embed version information:

```
go build -ldflags "\
  -X github.com/jscaltreto/downstage/cmd.version=1.0.0 \
  -X github.com/jscaltreto/downstage/cmd.commit=$(git rev-parse HEAD) \
  -X github.com/jscaltreto/downstage/cmd.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o downstage .
```

## Releases

Release PRs and changelog updates are managed by Release Please. Merging a
release PR creates the Git tag and GitHub release, and GoReleaser then builds
artifacts for macOS, Linux, and Windows on `amd64` and `arm64`.

For local release validation:

```
make release-check
make release-snapshot
```

These targets require [`goreleaser`](https://goreleaser.com/install/) to be
installed locally.

Publishing the Homebrew formula is handled by the release workflow, which
updates `jscaltreto/homebrew-tap` after GoReleaser publishes release assets.
This requires a repository secret named `HOMEBREW_TAP_GITHUB_TOKEN` with push
access to `jscaltreto/homebrew-tap`.

Release Please requires a repository secret named `RELEASE_PLEASE_TOKEN` with
enough access to create release PRs, tags, and releases in
`jscaltreto/downstage`.

## License

Downstage source code is licensed under the [MIT License](LICENSE).

Bundled fonts in [`internal/render/pdf/fonts/`](/home/jake/git/theatrescript/internal/render/pdf/fonts/)
remain under the SIL Open Font License 1.1; see the included `OFL.txt` and
`OFL-LibreBaskerville.txt` files.

## Acknowledgments

- [Fountain](https://fountain.io/) — screenplay markup language that inspired Downstage's plaintext philosophy
- [TheatreScript](https://github.com/contrapunctus-1/TheatreScript) — archived stage play markup spec that Downstage builds on
