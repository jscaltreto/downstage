---
permalink: false
id: dual-dialogue
kicker: "6"
title: Dual Dialogue
order: 12
codeLang: downstage
codeLabel: downstage
code: |-
  HORATIO
  They're here.

  HAMLET ^
  Then let them come.
---
To mark simultaneous speech, add `^` to the end of the second character cue.
The first cue stays normal.

The `^` must be the last character on the line, preceded by a space, and only
the second cue gets the marker. Forced character cues also work, as in
`@narrator ^`. If there is no previous dialogue block to pair with, the cue is
treated as normal dialogue.
