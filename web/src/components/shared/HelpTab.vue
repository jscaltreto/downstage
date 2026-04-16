<script setup lang="ts">
import { ref } from 'vue';
import { Keyboard, Type, Clapperboard, ExternalLink } from 'lucide-vue-next';

const props = defineProps<{
  openLink: (url: string) => Promise<void>;
}>();

type HelpSection = 'shortcuts' | 'syntax' | 'features';
const activeSection = ref<HelpSection>('shortcuts');

const shortcuts = [
  { keys: ['Ctrl/⌘', 'B'], desc: 'Bold' },
  { keys: ['Ctrl/⌘', 'I'], desc: 'Italic' },
  { keys: ['Ctrl/⌘', 'U'], desc: 'Underline' },
  { keys: ['Ctrl/⌘', 'F'], desc: 'Find' },
  { keys: ['Ctrl/⌘', 'H'], desc: 'Find & Replace' },
  { keys: ['Ctrl/⌘', 'Alt', 'F'], desc: 'Format Document' },
];

const features = [
  { name: 'Preview', desc: 'Live manuscript rendering alongside your source text.' },
  { name: 'Outline', desc: 'Navigable structure tree — acts, scenes, and characters.' },
  { name: 'Stats', desc: 'Word counts, runtime estimate, and per-character breakdowns.' },
  { name: 'Issues', desc: 'Warnings and errors from the Downstage language server.' },
  { name: 'Find & Replace', desc: 'Search with literal or regex matching, and bulk replace.' },
  { name: 'Spell Check', desc: 'Browser-native spell checking with a per-draft allowlist.' },
  { name: 'Manuscript / Acting Edition', desc: 'Toggle between standard manuscript and condensed acting-edition layout.' },
  { name: 'Dark Mode', desc: 'Switch between light and dark editor themes.' },
];
</script>

<template>
  <div class="flex h-full flex-col overflow-hidden">
    <div class="flex items-center gap-1 border-b border-border px-4 py-1.5">
      <button
        v-for="section in ([
          { id: 'shortcuts' as HelpSection, icon: Keyboard, label: 'Shortcuts' },
          { id: 'syntax' as HelpSection, icon: Type, label: 'Syntax' },
          { id: 'features' as HelpSection, icon: Clapperboard, label: 'Features' },
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
      <!-- Shortcuts -->
      <div v-if="activeSection === 'shortcuts'" class="space-y-1.5">
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

      <!-- Syntax -->
      <div v-if="activeSection === 'syntax'" class="space-y-3">
        <dl class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Play Header</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code># My Play
Subtitle: A Play in One Act
Author: Your Name
Draft: First</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Cue + Dialogue</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>ALICE
I know this looks reckless.</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Stage Direction</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>&gt; The lights cut to black.</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Structure</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>## ACT I
### SCENE 1</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Emphasis</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>**bold**
*italic*
_underline_</code></pre>
            </dd>
          </div>
          <div class="rounded-lg border border-border bg-black/5 p-3 dark:bg-white/5">
            <dt class="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-text-main">Song Block</dt>
            <dd>
              <pre class="overflow-x-auto text-xs leading-relaxed text-text-muted"><code>ALICE (singing)
The night is young and bright.</code></pre>
            </dd>
          </div>
        </dl>
        <p class="text-xs text-text-muted">
          Full spec and examples in the
          <button
            class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
            @click="props.openLink('https://www.getdownstage.com/syntax/')"
          >
            Syntax Guide
            <ExternalLink class="mb-0.5 inline h-3 w-3" />
          </button>
        </p>
      </div>

      <!-- Features -->
      <div v-if="activeSection === 'features'" class="space-y-1.5">
        <div
          v-for="f in features"
          :key="f.name"
          class="rounded-md bg-black/[0.03] px-3 py-2 dark:bg-white/[0.03]"
        >
          <span class="text-xs font-bold text-text-main">{{ f.name }}</span>
          <span class="ml-2 text-xs text-text-muted">{{ f.desc }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
