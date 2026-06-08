<script setup lang="ts">
import { ExternalLink } from 'lucide-vue-next';
import type { HelpHost } from '../help-sections';
import { helpLinks } from '../../../core/help-links';

defineProps<{
  openLink: (url: string) => Promise<void>;
  host: HelpHost;
}>();
</script>

<template>
  <div class="space-y-4 text-xs leading-relaxed text-text-muted">
    <section class="space-y-2">
      <h2 class="text-xs font-bold uppercase tracking-[0.14em] text-text-main">Welcome to Downstage</h2>
      <p>
        Downstage is a plain-text writing tool for plays and musicals. A script is
        just a text file. Acts, scenes, dialogue, and stage directions follow a
        handful of conventions you can learn in a few minutes.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">The editor</h3>
      <p>
        The large text area on the left is where you write. Type the way you'd
        type any document. The preview on the right updates to show the page as it
        will print. To give the script your full attention, hide the preview with
        the eye icon or
        <kbd class="rounded border border-border px-1 font-mono text-[10px]">Ctrl/⌘\</kbd>.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Where your work lives</h3>
      <p v-if="host === 'desktop'">
        Downstage Write reads from a <em>library folder</em> on your disk. The
        file list on the left shows every <code class="rounded bg-black/5 px-1 dark:bg-white/5">.ds</code> file inside.
        Saved versions of each file are kept in a hidden
        <code class="rounded bg-black/5 px-1 dark:bg-white/5">.downstage/</code> folder next to your scripts. Your work
        stays on your machine, in files you can read and back up yourself.
      </p>
      <p v-else>
        The browser editor keeps drafts in your browser's local storage. Good
        for trying things out or jotting a scene, but anything that clears
        site data (incognito windows, browser cleanups) will wipe them. For
        a script you plan to keep, install
        <strong class="text-text-main">Downstage Write</strong>, the desktop app.
        Same editor, but your scripts live in a folder on your disk, with saved
        versions kept alongside them.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Where to look next</h3>
      <ul class="ml-4 list-disc space-y-1">
        <li><strong class="text-text-main">Writing</strong>. A cheat sheet for dialogue, stage directions, and headings.</li>
        <li v-if="host === 'desktop'"><strong class="text-text-main">Versions</strong>. How to save named snapshots of a script and roll back to one.</li>
        <li v-if="host === 'desktop'"><strong class="text-text-main">Library</strong>. Organizing files into folders and opening scripts from elsewhere on disk.</li>
        <li><strong class="text-text-main">Export</strong>. PDF for readings and submissions, <code class="rounded bg-black/5 px-1 dark:bg-white/5">.ds</code> for handing the source to another Downstage user.</li>
        <li><strong class="text-text-main">Shortcuts</strong>. Every keyboard shortcut in one list.</li>
      </ul>
    </section>

    <p>
      The full syntax reference lives at
      <button
        class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
        @click="openLink(helpLinks.syntax)"
      >
        getdownstage.com/syntax
        <ExternalLink class="mb-0.5 inline h-3 w-3" />
      </button>.
    </p>
  </div>
</template>
