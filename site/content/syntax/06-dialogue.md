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

A cue must be preceded by a blank line (or another structural break such as a
heading). That way an ALL-CAPS line inside a speech — a shouted "WHAT" directly
below the cue — stays part of the dialogue instead of turning into a new
character. If you need a cue without a blank line before it, prefix the name
with `@` to force it.
