---
permalink: false
id: acts-scenes
kicker: "4"
title: Acts and Scenes
order: 10
examplePairs:
  - text: Acts are usually written as <code>##</code> headings.
    lang: downstage
    label: downstage
    code: |-
      ## ACT I
      ## ACT 2
      ## ACT: Finale
  - text: Scenes are usually written as <code>###</code> headings, though a non-act <code>##</code> heading inside an act also becomes a scene.
    lang: downstage
    label: downstage
    code: |-
      ## ACT I

      ### SCENE 1
      ### SCENE 1: The Palace
      ### A Bare Stage
  - text: One-acts and short-form pieces often skip acts entirely. Write scenes as <code>##</code> headings and Downstage treats them as direct children of the play.
    lang: downstage
    label: downstage
    code: |-
      ## SCENE 1: Opening

      ALICE
      Hello.

      ## SCENE 2: Closing

      ALICE
      Goodbye.
---
If you do not need acts or scenes, skip them. Downstage does not force a
structure you are not using.
