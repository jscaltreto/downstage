---
permalink: false
id: comments-breaks
kicker: "11"
title: Comments and Page Breaks
order: 12
examplePairs:
  - text: Comments are ignored when rendering, so they will not appear in the PDF.
    lang: downstage
    label: downstage
    code: |-
      // A line comment

      /* A block comment
         spanning multiple lines */
  - text: <code>===</code> inserts a page break in rendered output.
    lang: downstage
    label: downstage
    code: |-
      ### SCENE 1

      ===

      ### SCENE 2
---
These elements affect the working file and rendered pages differently.
