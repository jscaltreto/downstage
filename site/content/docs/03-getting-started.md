---
permalink: false
id: getting-started
kicker: How
title: Get started without jargon
navTitle: Get Started
order: 3
steps:
  - title: Install Downstage
    text: "If you already use Homebrew, this is the simplest route:"
    code: |-
      brew tap jscaltreto/tap
      brew install downstage
    after: "If you use Go instead:"
    extraCode: go install github.com/jscaltreto/downstage@latest
  - title: Write a file
    text: "Create a file such as `my-play.ds` and put this in it:"
    code: |-
      Title: My Play
      Author: You

      JANE
      I finally started the draft.
  - title: Render a manuscript
    text: "When you want a document you can read or share, render it:"
    code: downstage render my-play.ds
    after: This creates a PDF by default.
callout:
  title: "Important:"
  text: the command line here is small on purpose. Most writers only need a couple of commands to get value out of Downstage.
---
You do not need to learn the whole spec before you begin. Install the
command-line tool once, create a text file ending in `.ds`, and start with a
tiny script.
