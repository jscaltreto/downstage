package html

const standardCSS = `
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: "Courier New", Courier, monospace;
  font-size: 12pt;
  line-height: 1.5;
  color: #000;
  background: #fff;
}

.downstage-document {
  max-width: 8.5in;
  margin: 0 auto;
  padding: 1in;
}

/* Title Page */
.downstage-title-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 80vh;
  text-align: center;
  page-break-after: always;
}
.downstage-title-page h1 {
  font-size: 20pt;
  text-transform: uppercase;
  margin-bottom: 0.5em;
}
.downstage-title-page .subtitle {
  font-style: italic;
  font-size: 14pt;
  margin-bottom: 1em;
}
.downstage-title-page .author {
  font-size: 14pt;
  margin-bottom: 0.5em;
}
.downstage-title-page .metadata {
  margin-top: auto;
  font-size: 10pt;
}
.downstage-title-page .metadata dt { display: inline; font-weight: bold; }
.downstage-title-page .metadata dt::after { content: ": "; }
.downstage-title-page .metadata dd { display: inline; margin: 0; }
.downstage-title-page .metadata div { margin-bottom: 0.25em; }

/* Dramatis Personae */
.downstage-dramatis-personae {
  page-break-after: always;
  margin-bottom: 2em;
}
.downstage-dramatis-personae > h2 {
  text-align: center;
  font-size: 14pt;
  text-transform: uppercase;
  margin-bottom: 1.5em;
}
.downstage-dramatis-personae dl { margin-left: 8%; }
.downstage-dramatis-personae dt { font-weight: bold; display: inline; }
.downstage-dramatis-personae dd { display: inline; margin: 0; }
.downstage-dramatis-personae dd::before { content: " \2014 "; }
.downstage-dramatis-personae .character-entry { margin-bottom: 0.25em; }
.downstage-character-group-name {
  text-align: center;
  font-weight: bold;
  margin-top: 1em;
  margin-bottom: 0.5em;
}

/* Acts & Scenes */
.downstage-act {
  margin-top: 2em;
}
.downstage-act > h2 {
  text-align: center;
  text-transform: uppercase;
  text-decoration: underline;
  font-size: 14pt;
  margin-bottom: 1.5em;
}
.downstage-scene {
  margin-top: 1.5em;
}
.downstage-scene > h3 {
  text-align: center;
  text-transform: uppercase;
  font-size: 12pt;
  margin-bottom: 1em;
}

/* Generic Sections */
.downstage-section > h1,
.downstage-section > h2,
.downstage-section > h3 {
  text-align: center;
  text-transform: uppercase;
  margin-bottom: 1.5em;
}
.downstage-section > p { margin-bottom: 0; }

.downstage-forced-heading {
  text-align: center;
  margin-top: 1em;
  margin-bottom: 1em;
}

/* Dialogue */
.downstage-dialogue {
  margin-top: 1em;
  margin-bottom: 0;
}
.downstage-character {
  text-align: center;
  text-transform: uppercase;
  font-weight: bold;
  margin-bottom: 0;
}
.downstage-parenthetical {
  text-align: center;
  font-style: italic;
  font-weight: normal;
  margin-bottom: 0;
}
.downstage-line {
  margin: 0 8%;
}
.downstage-line.downstage-verse {
  margin-left: calc(8% + 2em);
}
.downstage-dialogue-break {
  margin-top: 1em;
}

/* Dual Dialogue */
.downstage-dual-dialogue {
  display: flex;
  gap: 1em;
  margin-top: 1em;
}
.downstage-dual-dialogue > .downstage-dialogue {
  flex: 1;
  margin-top: 0;
}
.downstage-dual-dialogue .downstage-line {
  margin-left: 0;
  margin-right: 0;
}
.downstage-dual-dialogue .downstage-line.downstage-verse {
  margin-left: 2em;
}

/* Stage Direction */
.downstage-stage-direction {
  font-style: italic;
  margin-top: 0.5em;
  margin-bottom: 0.5em;
}
.downstage-stage-direction.downstage-continuation {
  margin-top: 0;
  margin-bottom: 0;
}
.downstage-stage-direction + .downstage-stage-direction:not(.downstage-continuation) {
  margin-top: 1em;
}

/* Song */
.downstage-song {
  margin-top: 1.5em;
  margin-bottom: 1.5em;
}
.downstage-song > h4 {
  text-align: center;
  text-transform: uppercase;
  font-weight: bold;
  margin-bottom: 0.5em;
}
.downstage-song-end {
  text-align: center;
  font-weight: bold;
  margin-top: 0.5em;
}

/* Verse */
.downstage-verse-block {
  margin-top: 0.5em;
  margin-bottom: 0.5em;
}
.downstage-verse-line {
  margin-left: calc(8% + 2em);
  margin-bottom: 0;
}

/* Page Break */
.downstage-page-break {
  border: none;
  page-break-after: always;
  margin: 2em 0;
}

/* Inline */
.downstage-inline-direction {
  font-style: italic;
}

/* Section line spacing */
.downstage-section-break { margin-top: 1.5em; }

@media print {
  .downstage-document { padding: 0; max-width: none; }
  .downstage-title-page { min-height: 100vh; page-break-after: always; }
  .downstage-dramatis-personae { page-break-after: always; }
  .downstage-act { page-break-before: always; }
  .downstage-dialogue { orphans: 3; widows: 2; }
  .downstage-song { page-break-inside: avoid; }
  .downstage-page-break { page-break-after: always; height: 0; margin: 0; }
}
`

const condensedCSS = `
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: Georgia, "Times New Roman", Times, serif;
  font-size: 10pt;
  line-height: 1.4;
  color: #000;
  background: #fff;
}

.downstage-document {
  max-width: 5.5in;
  margin: 0 auto;
  padding: 0.5in;
}

/* Title Page */
.downstage-title-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  text-align: center;
  page-break-after: always;
}
.downstage-title-page h1 {
  font-size: 16pt;
  text-transform: uppercase;
  margin-bottom: 0.5em;
}
.downstage-title-page .subtitle {
  font-style: italic;
  font-size: 11pt;
  margin-bottom: 0.75em;
}
.downstage-title-page .author {
  font-size: 11pt;
  margin-bottom: 0.5em;
}
.downstage-title-page .metadata {
  margin-top: auto;
  font-size: 9pt;
}
.downstage-title-page .metadata dt { display: inline; font-weight: bold; }
.downstage-title-page .metadata dt::after { content: ": "; }
.downstage-title-page .metadata dd { display: inline; margin: 0; }
.downstage-title-page .metadata div { margin-bottom: 0.2em; }

/* Dramatis Personae */
.downstage-dramatis-personae {
  page-break-after: always;
  margin-bottom: 1.5em;
}
.downstage-dramatis-personae > h2 {
  text-align: center;
  font-size: 11pt;
  text-transform: uppercase;
  margin-bottom: 1em;
}
.downstage-dramatis-personae dl { margin-left: 0; }
.downstage-dramatis-personae dt { font-weight: bold; display: inline; }
.downstage-dramatis-personae dd { display: inline; margin: 0; }
.downstage-dramatis-personae dd::before { content: " \2014 "; }
.downstage-dramatis-personae .character-entry { margin-bottom: 0.2em; }
.downstage-character-group-name {
  text-align: center;
  font-weight: bold;
  margin-top: 0.75em;
  margin-bottom: 0.25em;
}

/* Acts & Scenes */
.downstage-act {
  margin-top: 1.5em;
}
.downstage-act > h2 {
  text-align: center;
  text-transform: uppercase;
  font-size: 11pt;
  margin-bottom: 1em;
}
.downstage-scene {
  margin-top: 1em;
}
.downstage-scene > h3 {
  text-align: center;
  text-transform: uppercase;
  font-size: 10pt;
  margin-bottom: 0.75em;
}

/* Generic Sections */
.downstage-section > h1,
.downstage-section > h2,
.downstage-section > h3 {
  text-align: center;
  text-transform: uppercase;
  margin-bottom: 1em;
}
.downstage-section > p { margin-bottom: 0; }

.downstage-forced-heading {
  text-align: center;
  margin-top: 0.75em;
  margin-bottom: 0.75em;
}

/* Dialogue — condensed: inline character name */
.downstage-dialogue {
  margin-top: 0.5em;
  margin-bottom: 0;
}
.downstage-character {
  text-transform: uppercase;
  font-weight: bold;
  display: inline;
  margin-bottom: 0;
}
.downstage-character::after { content: ". "; }
.downstage-parenthetical {
  font-style: italic;
  display: inline;
}
.downstage-line {
  display: inline;
  margin: 0;
}
.downstage-line.downstage-verse {
  display: block;
  margin-left: 2em;
}
.downstage-line-break { display: block; }
.downstage-dialogue-break { display: block; margin-top: 0.3em; }

/* Dual Dialogue */
.downstage-dual-dialogue {
  display: flex;
  gap: 0.5em;
  margin-top: 0.5em;
}
.downstage-dual-dialogue > .downstage-dialogue {
  flex: 1;
  margin-top: 0;
}

/* Stage Direction */
.downstage-stage-direction {
  font-style: italic;
  margin-top: 0.5em;
  margin-bottom: 0.5em;
  margin-left: 1em;
}
.downstage-stage-direction.downstage-continuation {
  margin-top: 0;
  margin-bottom: 0;
}
.downstage-stage-direction + .downstage-stage-direction:not(.downstage-continuation) {
  margin-top: 0.3em;
}

/* Song */
.downstage-song {
  margin-top: 1em;
  margin-bottom: 1em;
}
.downstage-song > h4 {
  text-align: center;
  text-transform: uppercase;
  font-weight: bold;
  font-size: 10pt;
  margin-bottom: 0.25em;
}
.downstage-song-end {
  text-align: center;
  font-weight: bold;
  margin-top: 0.25em;
}

/* Verse */
.downstage-verse-block {
  margin-top: 0.25em;
  margin-bottom: 0.25em;
}
.downstage-verse-line {
  margin-left: 2em;
  margin-bottom: 0;
}

/* Page Break */
.downstage-page-break {
  border: none;
  page-break-after: always;
  margin: 1.5em 0;
}

/* Inline */
.downstage-inline-direction {
  font-style: italic;
}

/* Section line spacing */
.downstage-section-break { margin-top: 1em; }

@media print {
  .downstage-document { padding: 0; max-width: none; }
  .downstage-title-page { min-height: 100vh; page-break-after: always; }
  .downstage-dramatis-personae { page-break-after: always; }
  .downstage-act { page-break-before: always; }
  .downstage-dialogue { orphans: 3; widows: 2; }
  .downstage-song { page-break-inside: avoid; }
  .downstage-page-break { page-break-after: always; height: 0; margin: 0; }
}
`
