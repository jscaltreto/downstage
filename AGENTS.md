## Scope

These instructions apply to the entire repository unless a deeper `AGENTS.md`
file overrides them for a subtree.

## Repo Priorities

- Preserve Downstage as a plaintext-first writing tool. Favor readable source
  and stable authoring workflows over clever abstractions.
- Keep parser, LSP, renderers, CLI, and editor integrations aligned. Do not
  change one surface and leave the others drifting.
- PDF output is the canonical manuscript rendering. Other output targets may
  exist, but they must not redefine document semantics.

## Change Boundaries

- Keep semantic logic in Go. Editor integrations should consume Go behavior,
  not reimplement parsing or document rules in Lua or TypeScript.
- Prefer small, scoped changes. Avoid drive-by refactors when solving a local
  problem.
- Do not commit generated artifacts, local editor state, packaged VSIX files,
  `node_modules`, or render outputs unless the task explicitly requires them.

## Validation

- Run focused tests for the package you changed at minimum.
- Run broader validation when behavior crosses package boundaries.
- Before shipping editor work, make sure the relevant extension build steps
  pass in addition to Go tests.

## Docs And UX

- Update user-facing docs when commands, editor workflows, or render behavior
  change.
- Prefer explicit, writer-facing behavior over hidden magic. If automation is
  added, make it predictable and easy to disable.

## Design Documents

- Store design documents, epics, and architectural plans in the `.design_docs/`
  directory.
- This directory is gitignored and should not be committed to the repository.
- Use these documents to track long-term goals and complex feature implementations.

## Scoped Files

- `internal/lsp/AGENTS.md` covers language-server work.
- `internal/render/AGENTS.md` covers renderers and output targets.
- `internal/desktop/AGENTS.md` covers the Wails desktop app backend.
- `editors/vscode/AGENTS.md` covers the VS Code extension.
- `web/AGENTS.md` covers the Vue-based web editor.
- `web/src/desktop/AGENTS.md` covers the desktop frontend layer.
