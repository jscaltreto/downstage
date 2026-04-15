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

A cue must be preceded by a blank line, or be the first line of the document.
Comments are transparent: a line or block comment directly above a cue counts,
as long as the comment itself has a blank line (or the start of the document)
before it. That way a shouted "WHAT" directly below a cue — even with a
parenthetical or inline comment between them — stays part of the dialogue
instead of turning into a new character. Headings, page breaks, and stage
directions are not cue boundaries on their own: always put a blank line before
a cue. When you need a cue without a blank line before it, prefix the name
with `@` to force it (see Forced Elements).
