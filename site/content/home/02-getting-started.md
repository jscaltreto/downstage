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
  title: "Want to skip the install?"
  text: '<a href="../editor/">Try the web editor</a> — it runs entirely in your browser with live preview and PDF export, no install required.'
---
