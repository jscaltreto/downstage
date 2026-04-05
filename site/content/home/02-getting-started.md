---
permalink: false
id: getting-started
kicker: Get Started
title: Start in three steps
navTitle: Get Started
order: 2
steps:
  - title: Install it
    codeLang: bash
    codeLabel: shell
    code: |-
      brew tap jscaltreto/tap
      brew install downstage
  - title: Write a file
    codeLang: downstage
    codeLabel: downstage
    code: |-
      Title: My Play
      Author: You

      JANE
      I finally started the draft.
  - title: Render it
    codeLang: bash
    codeLabel: shell
    code: downstage render my-play.ds
callout:
  title: Need the reference?
  text: The full reference, examples, and syntax guide live in <a href="docs/">the docs</a>.
---
