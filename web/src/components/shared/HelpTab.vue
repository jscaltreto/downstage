<script setup lang="ts">
import { ref } from 'vue';
import {
  Type, Layout, Keyboard, ExternalLink,
  Eye, ListTree, BarChart3, AlertTriangle, Search, SpellCheck, ScrollText,
} from 'lucide-vue-next';
import type { Component } from 'vue';
import { shortcuts as sc } from '../../core/platform';

const { openLink } = defineProps<{
  openLink: (url: string) => Promise<void>;
}>();

type HelpSection = 'syntax' | 'tools' | 'shortcuts';
const activeSection = ref<HelpSection>('syntax');
const sections: { id: HelpSection; icon: Component; label: string }[] = [
  { id: 'syntax', icon: Type, label: 'Writing' },
  { id: 'tools', icon: Layout, label: 'Tools' },
  { id: 'shortcuts', icon: Keyboard, label: 'Shortcuts' },
];

const shortcutList = [
  sc.bold, sc.italic, sc.underline,
  sc.find, sc.findReplace,
  sc.preview, sc.help,
];

const tools: { icon: Component; name: string; desc: string }[] = [
  { icon: Eye, name: 'Preview', desc: 'See the printed page beside your script.' },
  { icon: ListTree, name: 'Outline', desc: 'Jump between acts, scenes, and characters.' },
  { icon: BarChart3, name: 'Stats', desc: 'See word count, runtime estimate, and who speaks most.' },
  { icon: AlertTriangle, name: 'Issues', desc: 'Catch typos, structural errors, and formatting mistakes.' },
  { icon: Search, name: 'Find & Replace', desc: 'Search for a word or phrase, then replace it once or everywhere.' },
  { icon: SpellCheck, name: 'Spell Check', desc: 'Underline misspellings and add names or terms to this script’s allowlist.' },
  { icon: ScrollText, name: 'Manuscript / Acting Edition', desc: 'Switch between standard manuscript and a compact acting-edition layout.' },
];
</script>

<template>
  <div class="flex h-full flex-col overflow-hidden">
    <div class="flex items-center gap-1 border-b border-border px-4 py-1.5">
      <button
        v-for="section in sections"
        :key="section.id"
        class="flex items-center gap-1.5 rounded px-2.5 py-1.5 text-[10px] font-bold uppercase tracking-[0.14em] transition-colors"
        :class="activeSection === section.id
          ? 'bg-brass-500/15 text-accent'
          : 'text-text-muted hover:text-text-main hover:bg-black/5 dark:hover:bg-white/5'"
        @click="activeSection = section.id"
      >
        <component :is="section.icon" class="h-3 w-3" />
        {{ section.label }}
      </button>
    </div>

    <!-- `@container` opts this scroll region into Tailwind v4 container
         queries so the Writing-tab card grid tracks the drawer's width
         (not the viewport). In the right-docked drawer at its default
         360px the cards stack in one column; they spread to 2/3 cols
         as the drawer widens or when docked at the bottom of the
         editor pane. -->
    <div class="@container flex-1 overflow-y-auto px-4 py-3">
      <div v-if="activeSection === 'syntax'" class="space-y-3">
        <p class="text-xs text-text-muted">
          Downstage scripts are plain text. Write naturally. Structure does the work.
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
          You don't need a title page to start. Just dialogue works too. See the
          <button
            class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
            @click="openLink('https://www.getdownstage.com/syntax/')"
          >
            full Syntax Guide
            <ExternalLink class="mb-0.5 inline h-3 w-3" />
          </button>
          for the full reference.
        </p>
      </div>

      <div v-if="activeSection === 'tools'" class="space-y-1.5">
        <p class="mb-2 text-xs text-text-muted">
          All of these are in the toolbar above the editor.
        </p>
        <div
          v-for="t in tools"
          :key="t.name"
          class="flex items-center gap-2.5 rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
        >
          <component :is="t.icon" class="h-3.5 w-3.5 shrink-0 text-text-muted" />
          <span>
            <span class="text-xs font-bold text-text-main">{{ t.name }}</span>
            <span class="ml-2 text-xs text-text-muted">{{ t.desc }}</span>
          </span>
        </div>
      </div>

      <div v-if="activeSection === 'shortcuts'" class="space-y-1.5">
        <p class="mb-2 text-xs text-text-muted">
          These are the keyboard shortcuts. Everything else lives in the toolbar.
        </p>
        <div
          v-for="s in shortcutList"
          :key="s.label"
          class="flex items-center justify-between rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
        >
          <span class="text-xs text-text-main">{{ s.label }}</span>
          <kbd class="rounded border border-border bg-[var(--color-page-surface)] px-1.5 py-0.5 text-[10px] font-mono font-bold text-text-muted shadow-sm">{{ s.keys }}</kbd>
        </div>
      </div>
    </div>
  </div>
</template>
