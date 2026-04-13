---
permalink: false
id: getting-started
kicker: How
title: Try a tiny script
navTitle: Get Started
order: 4
steps:
  - title: Use the Web Editor
    text: "If you want the shortest path to a real script page, start in the Web Editor. It runs in the browser and gives you live preview plus PDF export."
    codeLang: downstage
    codeLabel: downstage
    code: |-
      # My Play
      Author: You

      JANE
      I finally started the draft.
    after: "Paste that into the Web Editor, make a small change, then export a PDF."
    actions:
      - label: Start Writing
        href: /editor/
        kind: primary
  - title: Use Visual Studio Code
    codeLang: text
    codeLabel: first session
    text: "If you already write code in Visual Studio Code, install the Downstage extension and stay in the editor you know. Not a developer? You can skip this one; the Web Editor has everything you need."
    code: |-
      1. Install the Downstage extension
      2. Open or create a .ds file
      3. Draft a scene
      4. Use live preview or render to PDF
    actions:
      - label: Get Extension
        href: https://marketplace.visualstudio.com/items?itemName=jscaltreto.downstage-vscode
        kind: primary
  - title: Use the command line
    codeLang: bash
    codeLabel: shell
    text: "If you prefer terminal workflows, install Downstage and render from the command line."
    code: |-
      brew tap jscaltreto/tap
      brew install downstage

      # Go users can install with:
      # go install github.com/jscaltreto/downstage@latest

      downstage render my-play.ds
callout:
  title: "New here?"
  text: 'Use <a href="../start/">Start Here</a> if you want a quick overview of the Web Editor, plus the Visual Studio Code extension and command-line tools for folks who want them, before you dig into the syntax guide.'
---
You do not need the whole spec before you begin. Start with a tiny script, then
keep the docs open for the parts you actually need.
