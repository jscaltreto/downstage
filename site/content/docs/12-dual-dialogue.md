---
permalink: false
id: dual-dialogue
kicker: "6"
title: Dual Dialogue
order: 12
code: |-
  HORATIO
  They're here.

  HAMLET ^
  Then let them come.
list:
  - The <code>^</code> must be the last character on the line, preceded by a space.
  - Only the second cue gets the marker.
  - "Forced character cues also work: <code>@narrator ^</code>."
  - If there is no previous dialogue block to pair with, the cue is treated as normal dialogue.
---
To mark simultaneous speech, add `^` to the end of the second character cue.
The first cue stays normal.
