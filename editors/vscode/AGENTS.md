## Scope

These instructions apply to `editors/vscode/`.

## Product Direction

- The VS Code extension should feel thin. Let the Go CLI and LSP own document
  semantics whenever possible.
- Keep PDF preview and render commands aligned with the real `downstage`
  command-line behavior. Do not fork option handling or error interpretation in
  TypeScript without a strong reason.
- Favor predictable authoring workflows over aggressive automation. If the
  extension auto-opens, auto-suggests, or auto-renders, make the trigger rules
  explicit and configurable.

## Git Hygiene

- Commit source, config, snippets, and docs.
- Do not commit `node_modules/`, `out/`, `.vsix-stage/`, packaged `.vsix`
  files, or other generated build output.
- Keep workspace debug config useful for local extension development, but do
  not let it become a dumping ground for personal editor preferences.

## Implementation Rules

- Keep TypeScript strict. Avoid `any` unless there is no sane alternative.
- Prefer small helper functions over one large activation function.
- Surface failures clearly through output channels, diagnostics, or user
  messages. Silent extension behavior is bad behavior.
- If a feature depends on `downstage` CLI behavior, document the expectation
  and expose configuration when path or workflow assumptions may vary.

## Validation

- Run `npm run lint`.
- Run `npm run compile`.
- If packaging behavior changed, also run the packaging flow before shipping.
