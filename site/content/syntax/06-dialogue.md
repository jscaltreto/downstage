---
permalink: false
id: dialogue
kicker: "5"
title: Dialogue
order: 6
codeLang: downstage
codeLabel: downstage
code: |-
  HAMLET
  (aside)
  To be, or not to be, that is the question.

  MARGARET
  They used to fill every seat, you know.
  Every last one.
---
Dialogue starts with a character name on its own line, followed by an optional
parenthetical and one or more speech lines.

Character names are uppercase and may contain spaces, punctuation like periods
and apostrophes, digits, and slashes for alias-aware names.

A cue is only recognised at a block boundary — the start of the document, the
line after a blank line, or the line immediately after a heading, page break,
stage direction, callout, or `SONG` / `SONG END` marker. Line comments and
block comments between a cue and its dialogue are transparent and don't break
this rule. That way an ALL-CAPS line inside a speech — a shouted "WHAT"
directly below the cue, even with a parenthetical or comment between them —
stays part of the dialogue instead of turning into a new character. When you
need a cue without a blank line before it, prefix the name with `@` to force
it (see Forced Elements).
