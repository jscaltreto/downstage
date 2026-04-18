# Downstage Language Specification

Version: 1.0

## 1. Introduction

Downstage is a plaintext markup language for stage plays. Files use the `.ds` extension.

Design principles:

- **Human-readable plaintext.** A `.ds` file is legible without any tooling.
- **Minimal markup.** Structure is inferred from conventions (ALL CAPS names, indentation, blank lines) rather than verbose tags.
- **Theatre-specific.** Built-in constructs for acts, scenes, dialogue, stage directions, verse, songs, and dramatis personae.

Downstage is inspired by [Fountain](https://fountain.io/) (screenplays) and [TheatreScript](https://github.com/contrapunctus-1/TheatreScript) (stage plays). It is not compatible with either.

See `README.md` for installation and CLI usage.

## 2. Document Structure

A Downstage document is organised as one or more top-level `#` sections.

Every `#` heading begins a new top-level section. In the common case, that section is a play. In compilation files, a top-level section can also act as a title-page-only section for the collection.

The heading level determines the structure:

- `# <title>` -- a top-level play section or title-page-only section
- `## Dramatis Personae`, `## Cast of Characters`, or `## Characters` -- character list for the enclosing top-level section
- `## ACT I` -- act heading (detected by "ACT" prefix)
- `### SCENE 1` -- scene heading (detected by "SCENE" prefix or position within an act)
- `## Notes` / `### Notes` -- generic nested prose section

Headings are structural, not presentational. Tooling SHOULD treat each `#` section as an independent top-level scope for metadata, dramatis personae, and numbering.

A top-level `#` heading is REQUIRED only when the document carries metadata, Dramatis Personae, or acts/scenes — that is, when there is something that needs a scope. Body-only fragments are valid without one:

```
ALICE
Hello, world!
```

A complete play with a title page and scope-owning metadata uses a heading:

```
# A Play
Author: You

ALICE
Hello, world!
```

## 3. Breaking Changes

This version intentionally breaks earlier Downstage behaviour in order to simplify authoring rules.

The following older patterns are no longer part of the preferred model:

- document-level `Key: Value` frontmatter before the first `#` heading
- document-level `# Dramatis Personae`
- ALL-CAPS cues without a preceding blank line (cues need a blank line or the start of the document; use `@NAME` to force one)

Instead:

- metadata lives directly under a `#` heading
- dramatis personae lives in `## Dramatis Personae` under that same `#` section

Tooling SHOULD call this out as a breaking change and provide a migration guide in the documentation.

For dramatis personae entries specifically, migration SHOULD replace Unicode dash separators such as ` — ` and ` – ` with ASCII ` - `, and rewrite standalone alias lines such as `[HAMLET/HAM]` into inline entries such as `HAMLET/HAM`.

## 4. Top-Level Sections

A file MAY contain multiple plays. Each play begins with a `#` heading whose content is that section's title.

Every `#` heading creates a top-level section. A top-level section is considered a play section when it contains play-body content such as:

- `## Dramatis Personae`
- acts or scenes
- dialogue
- songs
- stage directions or callouts
- verse blocks or other authored body content

A top-level section that contains metadata but no play-body content MAY be used as a title-page-only section for a compilation or collection. Renderers SHOULD treat such a section as frontmatter for that compilation block rather than as a play body.

Generic nested prose sections such as `## Notes` do not by themselves turn a top-level section into a play section.

### Local Metadata

Metadata is written directly under the `#` heading using `Key: Value` lines.

Each key-value pair uses one line. Indented lines continue the value for the most recent key.

Common keys: `Title`, `Subtitle`, `Author`, `Date`, `Draft`, `Copyright`, `Contact`, `Notes`.

The metadata block is only recognised when it appears immediately after the `#` heading, before any other body content.

If a section has both a `#` heading title and a `Title:` field, the `Title:` value is authoritative metadata. The `#` heading still begins the structural section.

### Example

```text
# The Example Play
Subtitle: A play in one act
Author: Your Name
Date: 2024
Draft: First
```

### Compilation Example

```text
# My Compilation
Author: Me

## First Production
My Compilation was first produced in 2026 at the Faketown Fringe Festival in Anytown, US.

# The First Play
Subtitle: A play in one act
Author: Your Name
Date: 2024
Draft: First
```

## 5. Dramatis Personae

The dramatis personae section begins with one of these headings inside a top-level section:

- `## Dramatis Personae`
- `## Cast of Characters`
- `## Characters`

Heading matching is case-insensitive.
Renderers use the author's chosen heading wording.

`## Dramatis Personae` applies only to the enclosing `#` section.

- a `## Dramatis Personae` section ends at the next `#` or `##` heading

### Character Entries

Format: `NAME` or `NAME - Description`

The name and description are separated by exactly space-hyphen-space (` - `).

Unicode dash separators such as em-dash (`—`) and en-dash (`–`) are not part of the entry syntax.

```
MARGARET - An aging actress, once famous
HENRY - Stage manager, loyal to Margaret
```

### Character Aliases

Aliases let a character be referenced by a shorter name in dialogue.

Aliases are written inline as `NAME/ALIAS` within the character entry line.

```
JAMES/JIM - Her estranged son
```

When `JIM` appears as a character name in dialogue, it is resolved to `JAMES`.

### Character Groups

Characters can be organized into groups using nested subheadings within the dramatis personae section.

- inside `## Dramatis Personae`, groups use `###`

```
# A Play

## Dramatis Personae

MARGARET - An aging actress
JAMES - Her son

### The Crew

STAGEHAND 1 - A quiet worker
STAGEHAND 2 - Talkative and nervous
```

Characters before any group heading are ungrouped. The group name is available in LSP hover information.

## 6. Acts and Scenes

### Acts

Acts are defined with `##` (H2) headings where the text after `##` is `ACT`, starts with `ACT `, or starts with `ACT:` (case-insensitive).

```
## ACT I
## ACT ONE
## ACT 1
## ACT 1: The Beginning
```

The act number is the text between "ACT" and the first colon (if any). The act title is the text after the colon (if any).
Acts SHOULD be numbered with Roman numerals (`ACT I`, `ACT II`, ...). Unnumbered acts remain valid, but they are discouraged and may be flagged by tooling.

### Scenes

A heading is recognised as a scene when any of the following apply:

- The heading text is `SCENE`, starts with `SCENE `, or starts with `SCENE:` (case-insensitive) — at any heading level.
- The heading is a `##` (H2) or `###` (H3) inside an act and does not match the act keyword pattern.

Headings that do not match the `SCENE` keyword and appear **outside** an act are treated as generic sections, not scenes.

```
### SCENE 1
### SCENE 1: The Palace
## SCENE 2: The Garden
```

When the text matches the `SCENE` keyword form, the scene number is the text between `SCENE` and the first colon, and the scene title is the text after the colon. When a heading is implicitly treated as a scene (inside an act, without the keyword), the entire text is the scene title and the number is empty.
Scenes SHOULD be numbered with Arabic numerals (`SCENE 1`, `SCENE 2`, ...). Unnumbered scenes remain valid, but they are discouraged and may be flagged by tooling.

### Without Acts or Scenes

Acts and scenes are optional. Content can appear directly in the top-level section body:

```
# A Short Play

Author: Jane Smith

ALICE
Hello!

BOB
Goodbye!
```

### Example with Acts and Scenes

```
# The Last Curtain Call

Subtitle: A drama in two acts
Author: Eleanor Vance

## Dramatis Personae

ALICE - A young actor
BOB - Her brother

## ACT I

### SCENE 1

> A bare stage.

ALICE
Hello!

### SCENE 2

> A garden.

BOB
Goodbye!

## ACT II

### SCENE 1

ALICE
We meet again.
```

In a multi-play file, act and scene numbering is scoped to the enclosing top-level `#` section. Tooling SHOULD restart act numbering at each new `#` section, and SHOULD restart scene numbering within each act in that section.

## 7. Dialogue

Dialogue is the primary content type. A dialogue block consists of a cue, an optional parenthetical, and one or more lines of dialogue text.

An ALL-CAPS line is only a cue at the start of a document or after a blank line. Comments do not reset cue context. So `WHAT` under `JIM` stays dialogue, not a new character.

```
JIM
WHAT
```

A parenthetical between the cue and the shouted line does not change this.

Use `@NAME` when you need a cue without a blank line (see Section 14).

```
HAMLET
To be, or not to be, that is the question.
```

With a parenthetical:

```
HAMLET
(aside)
To be, or not to be, that is the question.
```

Multi-line dialogue:

```
MARGARET
They used to fill every seat, you know.
Every last one.
```

A single blank line within dialogue creates a paragraph break — additional vertical space between lines of the same speech. In standard layout this renders as a full blank line; in condensed layout the gap is smaller.

Dialogue ends when a blank line is followed by another structural element (another character name, heading, song, etc.) or when a double blank line is encountered.

### Dual Dialogue

Dual dialogue marks two dialogue blocks for side-by-side rendering, representing simultaneous speech. Append ` ^` (space + caret) to the second character's name:

```
BRICK
Screw retirement.

STEEL ^
Screw retirement.
```

Rules:

- The `^` must be the last character on the character name line, preceded by a space.
- Only the **second** character gets the `^`. The first character's block is normal dialogue.
- The two blocks are rendered side-by-side (left and right columns).
- Parentheticals, verse, and inline formatting work normally within dual dialogue.
- Forced characters work with dual dialogue: `@narrator ^`
- If a `^`-marked character has no preceding dialogue block to pair with, it is treated as regular dialogue.

Multi-character names like `JOE AND JANE` or `ALL` are valid character names and work in dual dialogue:

```
JOE
We should go.

JOE AND JANE ^
We should go.
```

### Character Name Rules

Character names in dialogue must satisfy all of the following:

- ALL CAPS (no lowercase letters)
- At least 1 character long
- Allowed characters: uppercase letters `A-Z`, digits, spaces, periods (`.`), commas (`,`), hyphens (`-`), apostrophes (`'`), and slashes (`/`)
- No underscores or other punctuation

Valid names: `Q`, `HAMLET`, `STAGE HAND`, `MARY-JANE`, `MR. SMITH`, `O'BRIEN`, `GUARD 1`

Invalid names: `Hamlet` (lowercase), `ROBOT_3` (underscore)

When these rules are too restrictive, use a forced character (see Section 14).

## 8. Stage Directions

### Standalone Stage Directions

A line prefixed with `>`:

```
> The lights dim. MARGARET crosses downstage.
```

Standalone stage directions can appear anywhere in the body -- between dialogue blocks, within scenes, at the top of the document. They are rendered as italic text without the `>` prefix.

Consecutive `>` lines with no blank lines between them are treated as a continuation — rendered with regular line spacing (no extra gap). Consecutive `>` lines separated by blank lines are treated as separate blocks with a paragraph break between them.

### Callouts

A line prefixed with `>>` creates a non-structural callout:

```
>> Midwinter. The room has not been heated for days.
```

Callouts can appear anywhere standalone stage directions can appear, but they do not create or close sections. Consecutive `>>` lines with no blank lines between them are treated as a continuation. Callouts render in bold and are indented further than standalone stage directions.

### Inline Stage Directions

Parenthetical text within a dialogue line is treated as an inline stage direction:

```
HAMLET
To be (pause) or not to be.
```

### Character Parentheticals

A stage direction immediately after a character name (before dialogue text) is a character direction:

```
CLAIRE
(entering briskly)
Ms. Thornton, we have three hours.
```

## 9. Verse

Lines indented with 2 or more spaces within dialogue are treated as verse. Verse preserves line breaks, which is important for poetry, song lyrics, and heightened speech.

```
MARGARET
  To stand upon this stage once more,
  to feel the boards beneath my feet,
  to hear the hush before the roar,
  to know this moment, bittersweet.
```

Verse and prose can be mixed within the same dialogue block:

```
HAMLET
A piece of work is man, how noble in reason.
  In form and moving, how express
  and admirable; in action, how like
  an angel.
```

The first line is prose. The indented lines are verse.

## 10. Songs

Song blocks mark musical numbers within the play.

### Starting a Song

A song begins with a `SONG` marker on its own line:

- `SONG` -- unnamed song
- `SONG: Title` -- titled song
- `SONG 1: Title` -- numbered and titled song
- `SONG 1` -- numbered song without title

### Ending a Song

`SONG END` on its own line.

### Song Content

Songs can contain dialogue (characters singing), verse, and stage directions:

```
SONG 1: The Stagehand's Lament

JIM
  Under the lights we never stand,
  behind the curtain, close at hand.

HENRY
  And when the crowd gives its applause,
  we take no bow, we get no cause.

SONG END
```

Songs can appear within scenes or directly in the body.

## 11. Inline Formatting

Formatting markers work within dialogue, stage directions, verse, character
descriptions in the dramatis personae, and frontmatter/section-metadata values
such as `Subtitle`.

| Syntax | Result |
|--------|--------|
| `*text*` | Italic |
| `**text**` | Bold |
| `***text***` | Bold italic |
| `_text_` | Underline |
| `~text~` | Strikethrough |

### Examples

```
CLAIRE
I need you in **costume** and ready for the _technical rehearsal_.

MARGARET
I was doing tech rehearsals before you were ***born***.

HAMLET
I think it was to see my mother's ~wedding~ marriage.
```

### Edge Cases

- Formatting does **not** nest. Inner markers within formatted text are treated as plain text.
- Unclosed formatting markers are treated as literal characters.

## 12. Comments

Comments are preserved in the AST but are not part of the play's content.

### Line Comments

```
// This is a line comment
```

### Block Comments

```
/* This is a block comment
   spanning multiple lines */
```

Block comments can also be single-line: `/* single line comment */`

Comments can appear anywhere -- between scenes, within dialogue blocks, at the top of the document.

## 13. Page Breaks

Three equals signs on their own line:

```
===
```

Page breaks indicate a page separation in rendered output. They can appear anywhere in the body.

## 14. Forced Elements

When the normal parsing rules produce the wrong result, forced elements override them.

### Forced Character (`@`)

Prefix a name with `@` to force it to be treated as a character name, even if it violates the ALL CAPS rules:

```
@horatio
My lord, I came to see your father's funeral.
```

Without the `@`, `horatio` would be treated as plain text. The `@` is stripped from the name in the AST.

A forced cue also opts the character out of Dramatis Personae membership checks: tooling SHOULD NOT flag a `@name` cue as unknown when the enclosing play has a `## Dramatis Personae` section. This makes `@` a concise way to mark brief character appearances without adding them to the roster. The suppression is scoped to the individual cue — subsequent non-forced uses of the same name are still checked.

### Forced Heading (`.`)

Prefix text with `.` to force it to be treated as a heading, without needing `#` markup:

```
.The Next Evening
```

This creates a structural heading. The `.` prefix requires the next character to be uppercase. The `.` is stripped from the heading text in the AST.

## 15. Rendering

The `downstage render` command converts `.ds` files to output formats.

### Formats

| Format | Flag | Output |
|--------|------|--------|
| PDF | `--format pdf` (default) | Paginated manuscript PDF |
| HTML | `--format html` | Self-contained HTML document |

### Styles

Both formats support `--style standard` (default) and `--style condensed`:

- **standard**: manuscript-oriented layout. Monospace font, centered character names above dialogue, indented dialogue margins.
- **condensed**: compact reading layout. Serif font, inline character names (bold, followed by dialogue on the same line), tighter spacing.

### PDF Page Size

PDF output supports `--page-size letter` (default) and `--page-size a4`.
Manuscript layout renders on the selected physical sheet. Condensed layout
derives its logical page from that sheet: half-letter for Letter, A5 for A4.

### HTML Output

HTML rendering produces a single self-contained `.html` file with an embedded stylesheet. The output uses semantic HTML with stable CSS class names for all major structures:

| Element | CSS Class |
|---------|-----------|
| Document wrapper | `.downstage-document` |
| Title page | `.downstage-title-page` |
| Dramatis personae | `.downstage-dramatis-personae` |
| Act | `.downstage-act` |
| Scene | `.downstage-scene` |
| Dialogue block | `.downstage-dialogue` |
| Character name | `.downstage-character` |
| Dialogue line | `.downstage-line` |
| Verse dialogue line | `.downstage-line.downstage-verse` |
| Stage direction | `.downstage-stage-direction` |
| Callout | `.downstage-callout` |
| Inline direction | `.downstage-inline-direction` |
| Song | `.downstage-song` |
| Verse block | `.downstage-verse-block` |
| Verse line | `.downstage-verse-line` |
| Page break | `.downstage-page-break` |
| Dual dialogue | `.downstage-dual-dialogue` |

### CLI Examples

```bash
downstage render play.ds                          # PDF, standard style
downstage render --format html play.ds            # HTML, standard style
downstage render --format html --style condensed play.ds
downstage render --format html --output play.html play.ds
```

## 16. Divergences from TheatreScript

Downstage is inspired by the archived [TheatreScript](https://github.com/contrapunctus-1/TheatreScript) specification but diverges in these ways:

1. **Comments.** Downstage adds `// line comment` and `/* block comment */` syntax. The original TheatreScript spec had no comment support.
2. **Forced elements.** `@CHARACTER` forces a character name, `.HEADING` forces a heading. Borrowed from Fountain's similar conventions.
3. **Page breaks.** `===` on its own line marks a page break. Not present in the original spec.
4. **Character aliases.** `HAMLET/HAM` in the dramatis personae defines short-form character names. The original spec had no alias mechanism.
5. **Arbitrary metadata.** Top-level section metadata accepts any `Key: Value` pairs. The original spec had a fixed set of metadata keys.
6. **Separate lines.** Character names and dialogue are always on separate lines. The original spec supported inline `NAME: dialogue` format, which Downstage does not.
7. **Dual dialogue.** `CHARACTER ^` marks simultaneous speech for side-by-side rendering. Inspired by [Fountain's dual dialogue](https://fountain.io/syntax/#dual-dialogue). Not present in the original spec.

## 17. Complete Example

The following is `testdata/full_play.ds`, demonstrating every Downstage feature in context.

```
# The Last Curtain Call

Subtitle: A Drama in Two Acts
Author: Eleanor Vance
Date: 2025
Draft: Third Draft
Copyright: 2025 Eleanor Vance
Contact: evance@example.com
Notes: Inspired by true events

// This is a full play demonstrating all Downstage features.

/*
   Block comment at the top level.
   This should be ignored by the parser.
*/

## Dramatis Personae

MARGARET - An aging actress, once famous
JAMES/JIM - Her estranged son, a stagehand
CLAIRE - The new director, ambitious and sharp
HENRY - Stage manager, loyal to Margaret

### The Crew
STAGEHAND 1 - A quiet worker
STAGEHAND 2 - Talkative and nervous

## ACT I

### SCENE 1

> The stage is bare except for a single chair downstage center. Dim lighting. The sound of an empty theatre.

MARGARET
(sitting in the chair, staring out)
They used to fill every seat, you know.
Every last one.
  And the balcony too,
  row after row of faces,
  all watching me.

JAMES
(entering from stage left)
Mother, the crew is waiting.

MARGARET
Let them wait. I've earned that much.

// A comment between dialogue lines.

JAMES
(sighing)
You've earned a lot of things. Not all of them good.

> MARGARET stands abruptly. She crosses downstage.

CLAIRE
(entering briskly)
Ms. Thornton, we have exactly *three hours* before curtain.
I need you in **costume** and ready for the _technical rehearsal_.

MARGARET
Don't lecture me, dear. I was doing tech rehearsals
before you were ***born***.

CLAIRE
(to JAMES, aside)
Is she always like this?

JAMES
(quietly)
Worse, usually.

> MARGARET turns and catches them whispering.

CLAIRE
(covering)
I was just saying how wonderful you look tonight.

JAMES ^
(quickly)
I was just saying how wonderful you look tonight.

===

### SCENE 2

> The backstage area. HENRY is checking props.

HENRY
(into headset)
All set for Act One. Standing by.

@STAGEHAND 1
The backdrop won't hold. Rigging's shot.

HENRY
(alarmed)
What do you mean it ~won't hold~?

STAGEHAND 2
(rushing in)
I told you! I *told* you the cables were frayed!

HENRY
Get me new cables. Now. Both of you, move.

> STAGEHAND 1 and STAGEHAND 2 exit hurriedly.

JIM
(to HENRY)
Need a hand?

HENRY
Always. Grab the number three fly line.

SONG 1: The Stagehand's Lament

JIM
  Under the lights we never stand,
  behind the curtain, close at hand.
  We pull the ropes and shift the scene,
  the unseen hands, the great machine.

HENRY
  And when the crowd gives its applause,
  we take no bow, we get no cause.

SONG END

/*
   End of Act I.
   The intermission should be approximately 15 minutes.
*/

## ACT II

### SCENE 1

.The Next Evening

> The theatre is fully lit. The set is complete. MARGARET stands center stage in full costume.

MARGARET
(to herself)
One more night. Just one more.

CLAIRE
(from the wings)
Places, everyone! This is it!

JAMES
(approaching MARGARET)
Mother, I wanted to say...

MARGARET
(cutting him off)
Save it for after. We have a show to do.

> The lights dim. A spotlight finds MARGARET.

MARGARET
  To stand upon this stage once more,
  to feel the boards beneath my feet,
  to hear the hush before the roar,
  to know this moment, bittersweet.

> Silence. Then, from the darkness, applause.

===

### SCENE 2

> After the show. The stage is empty. MARGARET sits alone in the chair from Act I.

SONG: Curtain Call

MARGARET
  The lights go down, the curtain falls,
  and silence fills these empty halls.
  But somewhere in the dark out there,
  a memory floats upon the air.

SONG END

JAMES
(entering quietly)
That was your best performance yet.

MARGARET
(smiling)
I know.

> They share a long look. JAMES extends his hand. MARGARET takes it. They exit together. The stage lights fade to black.
```
