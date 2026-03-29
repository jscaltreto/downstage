## Scope

These instructions apply to `internal/render/` and its subpackages.

## Canonical Output

- PDF is the canonical manuscript output. Treat it as the fidelity target for
  layout and pagination decisions.
- New render targets must share Downstage semantics with PDF. Do not invent a
  separate interpretation of cues, songs, verse, stage directions, or dual
  dialogue.

## Architecture

- Keep parsing and document semantics outside renderer-specific code. Renderers
  should consume a stable representation of the document, not re-parse source
  text ad hoc.
- Share logic for inline formatting and document structure where possible. Do
  not duplicate semantics across renderers unless there is a clear boundary and
  tests prove they stay aligned.
- Keep renderer-specific concerns isolated. Formatting choices belong in the
  target renderer, but document meaning does not.

## Output Discipline

- Any new format must have a clear product reason beyond "it seemed useful."
- If adding HTML or another target, document where it is expected to match PDF
  exactly and where approximation is acceptable.
- Do not bake editor-preview shortcuts into renderer code if they would weaken
  export correctness.

## Testing

- Add focused tests for renderer behavior changes.
- Preserve or extend golden-style coverage where output structure matters.
- Run the relevant render package tests after changes under `internal/render/`.
