---
permalink: false
id: workflow
kicker: Workflow
title: One draft, several outputs
navTitle: Workflow
order: 3
workflow:
  - title: Draft
    text: Write in plain text.
  - title: Validate
    text: Catch structural mistakes early.
  - title: Render
    text: Generate a PDF manuscript or HTML preview.
  - title: Refine
    text: Add editor support only when it helps.
code: |-
  downstage validate my-play.ds
  downstage render my-play.ds
  downstage render --format html my-play.ds
---
