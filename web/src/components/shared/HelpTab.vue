<script setup lang="ts">
import { ref } from 'vue';
import { Type, Layout, Keyboard, ExternalLink } from 'lucide-vue-next';

const props = defineProps<{
  openLink: (url: string) => Promise<void>;
}>();

type HelpSection = 'syntax' | 'tools' | 'shortcuts';
const activeSection = ref<HelpSection>('syntax');

const shortcuts = [
  { keys: ['Ctrl/⌘', 'F'], desc: 'Open Find' },
  { keys: ['Ctrl/⌘', 'H'], desc: 'Open Find & Replace' },
];

const tools = [
  { name: 'Preview', desc: 'See the printed page side-by-side as you write.' },
  { name: 'Outline', desc: 'Jump between acts, scenes, and characters.' },
  { name: 'Stats', desc: 'Word counts, estimated runtime, and who talks the most.' },
  { name: 'Issues', desc: 'Catch problems — misspelled character names, missing dialogue, formatting mistakes.' },
  { name: 'Find & Replace', desc: 'Search your script and fix names or lines in bulk.' },
  { name: 'Spell Check', desc: 'Underlines misspelled words. You can add names and terms to a per-script allowlist.' },
  { name: 'Manuscript / Acting Edition', desc: 'Switch between standard manuscript format and a compact acting-edition layout.' },
];
</script>

<template>
  <div class="flex h-full flex-col overflow-hidden">
    <div class="flex items-center gap-1 border-b border-border px-4 py-1.5">
      <button
        v-for="section in ([
          { id: 'syntax' as HelpSection, icon: Type, label: 'Writing' },
          { id: 'tools' as HelpSection, icon: Layout, label: 'Tools' },
          { id: 'shortcuts' as HelpSection, icon: Keyboard, label: 'Shortcuts' },
        ])"
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

    <div class="flex-1 overflow-y-auto px-4 py-3">
      <!-- Writing / Syntax -->
      <div v-if="activeSection === 'syntax'" class="space-y-3">
        <p class="text-xs text-text-muted">
          Downstage scripts are plain text. Type naturally — formatting comes from structure, not menus.
        </p>
        <dl class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Title Page</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code># My Play
Subtitle: A Drama
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
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Songs</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>SONG 1: Wanderer's Lament

ALICE
  Lyric line one.

SONG END</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Page Breaks &amp; Comments</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>===

// Note to self: fix this</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Forced Cues</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>@OFFSTAGE VOICE
Help! Is anyone there?</code></pre>
              <p class="mt-1.5 text-[10px] text-text-muted">Use <code class="font-mono">@</code> for one-off speakers not in the cast list.</p>
            </dd>
          </div>
        </dl>
        <p class="text-xs text-text-muted">
          You don't need a title page to start — just dialogue works too. See the
          <button
            class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
            @click="props.openLink('https://www.getdownstage.com/syntax/')"
          >
            full Syntax Guide
            <ExternalLink class="mb-0.5 inline h-3 w-3" />
          </button>
          for everything.
        </p>
      </div>

      <!-- Tools -->
      <div v-if="activeSection === 'tools'" class="space-y-1.5">
        <p class="mb-2 text-xs text-text-muted">
          All of these are in the toolbar above the editor.
        </p>
        <div
          v-for="t in tools"
          :key="t.name"
          class="rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
        >
          <span class="text-xs font-bold text-text-main">{{ t.name }}</span>
          <span class="ml-2 text-xs text-text-muted">{{ t.desc }}</span>
        </div>
      </div>

      <!-- Shortcuts -->
      <div v-if="activeSection === 'shortcuts'" class="space-y-1.5">
        <p class="mb-2 text-xs text-text-muted">
          Everything else is in the toolbar — these are the keyboard shortcuts.
        </p>
        <div
          v-for="s in shortcuts"
          :key="s.desc"
          class="flex items-center justify-between rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
        >
          <span class="text-xs text-text-main">{{ s.desc }}</span>
          <span class="flex items-center gap-1">
            <kbd
              v-for="(k, i) in s.keys"
              :key="i"
              class="rounded border border-border bg-[var(--color-page-surface)] px-1.5 py-0.5 text-[10px] font-mono font-bold text-text-muted shadow-sm"
            >{{ k }}</kbd>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
