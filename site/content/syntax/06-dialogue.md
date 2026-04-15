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

Cues need a blank line before them, or they can start the document. Comments do
not reset cue context, so a shouted "WHAT" under `JIM` stays dialogue. Use
`@NAME` when you need to force a cue without a blank line.
