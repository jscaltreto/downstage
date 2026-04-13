---
permalink: false
id: structure
kicker: "1"
title: Document Structure
order: 2
examplePairs:
  - text: A minimal file is valid. No title page or headings required.
    lang: downstage
    label: downstage
    code: |-
      ALICE
      Hello, world!
  - text: A full play opens with a <code>#</code> heading. Metadata, Dramatis Personae, and acts/scenes are scoped to that section.
    lang: downstage
    label: downstage
    code: |-
      # The Last Curtain Call
      Author: Eleanor Vance

      ## Dramatis Personae

      MARGARET - An aging actress
      JAMES - Her son

      ## ACT I

      ### SCENE 1

      MARGARET
      They used to fill every seat.
  - text: A compilation gathers multiple plays in a single file. Each play is its own <code>#</code> section, with independent metadata, Dramatis Personae, and act numbering. A leading <code>#</code> section that has metadata but no body acts as the collection's title page.
    lang: downstage
    label: downstage
    code: |-
      # Short Plays
      Author: Various

      # Waiting
      Author: A. Writer

      ## SCENE 1

      ALICE
      Is this it?

      # Departure
      Author: B. Writer

      ## SCENE 1

      BOB
      It's time.
---
A Downstage document has an optional title page and then the play body. The
body can contain a character list, acts, scenes, prose sections, dialogue,
songs, comments, and page breaks. Multi-play compilations use one
top-level `#` per play; numbering and Dramatis Personae are scoped per play,
not per file.
