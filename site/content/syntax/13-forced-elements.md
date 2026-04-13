---
permalink: false
id: forced-elements
kicker: "12"
title: Forced Elements
order: 13
examplePairs:
  - text: <code>@name</code> forces a character cue.
    lang: downstage
    label: downstage
    code: |-
      @horatio
      My lord, I came to see your father's funeral.
  - text: <code>.Heading</code> forces a structural heading.
    lang: downstage
    label: downstage
    code: |-
      .The Next Evening
  - text: A forced cue also opts the character out of the Dramatis Personae check for brief appearances you do not want to list.
    lang: downstage
    label: downstage
    code: |-
      @GUARD 2
      Who goes there?
---
When the parser would otherwise classify a line incorrectly, force it.
