---
permalink: false
id: title-page
kicker: "2"
title: Title Page
order: 8
codeLang: downstage
codeLabel: downstage
code: |-
  # The Last Curtain Call
  Subtitle: A Drama in Two Acts
  Author: Eleanor Vance
  Date: 2025
  Notes: Inspired by true events
    and several missed cues.
---
Title-page metadata lives directly under a top-level `#` heading as
`Key: Value` pairs. Any key name is accepted. Indented lines continue the
previous value.

The `#` heading owns the metadata scope. The metadata block ends at the first
blank line, the next heading, a page break, or a non-indented line that is not
a metadata pair.

If both the heading and a `Title:` entry are present, the `Title:` value wins
when the document is rendered — useful for working titles that shouldn't
appear in the final PDF.
