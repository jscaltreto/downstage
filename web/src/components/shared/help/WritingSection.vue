<script setup lang="ts">
import { ExternalLink, Bold, Italic, Underline, MessageSquare, ChevronRight, GalleryVerticalEnd, GalleryVertical, Music, FilePlus2, Eye, ListTree, BarChart3, AlertTriangle, Search, SpellCheck, ScrollText } from 'lucide-vue-next';
import type { Component } from 'vue';
import { helpLinks } from '../../../core/help-links';

defineProps<{
  openLink: (url: string) => Promise<void>;
}>();

const formatTools: { icon: Component; name: string; desc: string }[] = [
  { icon: Bold, name: 'Bold', desc: 'Wrap the selection in **double stars**.' },
  { icon: Italic, name: 'Italic', desc: 'Wrap the selection in *single stars*.' },
  { icon: Underline, name: 'Underline', desc: 'Wrap the selection in _underscores_.' },
  { icon: MessageSquare, name: 'Dialogue', desc: 'Start a new speech with a CHARACTER cue above your cursor.' },
  { icon: ChevronRight, name: 'Stage Direction', desc: 'Add `>` to the start of the line to mark it as a stage direction.' },
  { icon: GalleryVerticalEnd, name: 'Act Heading', desc: 'Insert `## ACT N` to begin a new act.' },
  { icon: GalleryVertical, name: 'Scene Heading', desc: 'Insert `### SCENE N` to begin a new scene.' },
  { icon: Music, name: 'Song Block', desc: 'Wrap lyrics in `SONG` / `SONG END` so they render as a song.' },
  { icon: FilePlus2, name: 'Page Break', desc: 'Insert `===` to force a new page in the printed manuscript.' },
];

const workbenchTools: { icon: Component; name: string; desc: string }[] = [
  { icon: Eye, name: 'Preview', desc: 'Show or hide the rendered page beside your script.' },
  { icon: ListTree, name: 'Outline', desc: 'Jump to any act, scene, or character in the script.' },
  { icon: BarChart3, name: 'Stats', desc: 'Word count, estimated running time, and lines per character.' },
  { icon: AlertTriangle, name: 'Issues', desc: 'Flags typos, broken structure, and formatting mistakes as you write.' },
  { icon: Search, name: 'Find & Replace', desc: 'Search for a phrase and replace it one match at a time or all at once.' },
  { icon: SpellCheck, name: 'Spell Check', desc: 'Underlines misspellings. Right-click to add a character name to this script\'s dictionary.' },
  { icon: ScrollText, name: 'Manuscript / Acting Edition', desc: 'Switch the preview between the full-page manuscript and the compact acting edition.' },
];
</script>

<template>
  <div class="space-y-4">
    <p class="text-xs text-text-muted">
      A Downstage script is plain text. Type the way a script reads, and the
      structure falls out of a few conventions.
    </p>

    <dl class="grid gap-3 @md:grid-cols-2 @2xl:grid-cols-3">
      <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
        <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Title Page</dt>
        <dd>
          <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code># My Play
Subtitle: A Play in One Act
Author: Your Name
Draft: First</code></pre>
        </dd>
      </div>
      <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
        <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Dialogue</dt>
        <dd>
          <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>ALICE
I know this looks reckless.

BOB
(laughing)
You always say that.</code></pre>
        </dd>
      </div>
      <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
        <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Stage Directions</dt>
        <dd>
          <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>&gt; The lights cut to black.

&gt; ALICE crosses to the bench
&gt; and sits.</code></pre>
        </dd>
      </div>
      <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
        <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Acts &amp; Scenes</dt>
        <dd>
          <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>## ACT I
### SCENE 1</code></pre>
        </dd>
      </div>
      <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
        <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Formatting</dt>
        <dd>
          <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>**bold**  *italic*
_underline_  ~strikethrough~</code></pre>
        </dd>
      </div>
      <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
        <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Page Breaks &amp; Comments</dt>
        <dd>
          <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>===

// Note to self: fix this</code></pre>
        </dd>
      </div>
    </dl>

    <p class="text-xs text-text-muted">
      A title page is optional. A file containing nothing but dialogue is a
      valid script. For the full reference, read the
      <button
        class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
        @click="openLink(helpLinks.syntax)"
      >
        Syntax Guide
        <ExternalLink class="mb-0.5 inline h-3 w-3" />
      </button>.
    </p>

    <section class="space-y-2 border-t border-border pt-3">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Formatting buttons</h3>
      <div
        v-for="t in formatTools"
        :key="t.name"
        class="flex items-center gap-2.5 rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
      >
        <component :is="t.icon" class="h-3.5 w-3.5 shrink-0 text-text-muted" />
        <span>
          <span class="text-xs font-bold text-text-main">{{ t.name }}</span>
          <span class="ml-2 text-xs text-text-muted">{{ t.desc }}</span>
        </span>
      </div>
    </section>

    <section class="space-y-2 border-t border-border pt-3">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Workbench</h3>
      <div
        v-for="t in workbenchTools"
        :key="t.name"
        class="flex items-center gap-2.5 rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
      >
        <component :is="t.icon" class="h-3.5 w-3.5 shrink-0 text-text-muted" />
        <span>
          <span class="text-xs font-bold text-text-main">{{ t.name }}</span>
          <span class="ml-2 text-xs text-text-muted">{{ t.desc }}</span>
        </span>
      </div>
    </section>
  </div>
</template>
