---
permalink: false
id: first-workflow
kicker: First Pass
title: Your first end-to-end workflow
navTitle: Your First Script
order: 4
workflow:
  - title: Draft
    text: Write cues, dialogue, headings, and stage directions in a single file.
  - title: Check
    text: Run `downstage validate my-play.ds` to catch structural mistakes.
  - title: Render
    text: Run `downstage render my-play.ds` to make a PDF manuscript.
  - title: Refine
    text: Add editor integration later if you want autocompletion, hover help, or previews.
code: |-
  downstage validate my-play.ds
  downstage render my-play.ds
  downstage render --format html my-play.ds
caption: That last command gives you an HTML version if you want a browser-friendly preview.
---
A typical beginner workflow looks like this: write in a plain text editor,
render when you want to see pages, then add editor support later if you want
live help while drafting.
