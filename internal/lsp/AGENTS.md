## Scope

These instructions apply to `internal/lsp/`.

## Design Rules

- Keep the LSP implementation protocol-first. Add editor-visible behavior
  through standard LSP features before reaching for editor-specific hacks.
- Derive behavior from parsed document structure, not line-based guesswork,
  unless the feature is intentionally heuristic and documented as such.
- Keep diagnostics, completions, hovers, symbols, definitions, and code
  actions internally consistent. A character or heading concept should not be
  interpreted three different ways across features.

## Completions And Diagnostics

- Completion suggestions must be context-aware and low-noise. Do not spam broad
  keyword lists when the document state can narrow the answer.
- Diagnostics should help the writer recover. Prefer actionable wording and
  attach data needed for related code actions.
- If a diagnostic is intentionally suppressed in a common context, preserve
  that behavior with tests.

## Code Actions

- Only offer quick fixes that are safe and unsurprising.
- Workspace edits must preserve surrounding structure and blank-line
  conventions. Do not "fix" one thing by mangling nearby author formatting.

## Testing

- Every behavior change needs regression coverage in `internal/lsp/*_test.go`.
- Add or update tests for both positive and negative cases when changing
  completions, diagnostics, symbols, or code actions.
- Run `go test ./internal/lsp/...` after changes here.
