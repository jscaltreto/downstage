<script setup lang="ts">
import type { HelpHost } from '../help-sections';

defineProps<{
  openLink: (url: string) => Promise<void>;
  host: HelpHost;
}>();
</script>

<template>
  <div class="space-y-4 text-xs leading-relaxed text-text-muted">
    <section class="space-y-2">
      <h2 class="text-xs font-bold uppercase tracking-[0.14em] text-text-main">Settings</h2>
      <p>
        Open <strong class="text-text-main">File → Settings…</strong> (or press
        <kbd class="rounded border border-border px-1 font-mono text-[10px]">Ctrl/⌘&shy;,</kbd>) to open the
        preferences window. It has four tabs.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Library</h3>
      <p>
        Shows which folder is currently your library. Click
        <strong class="text-text-main">Change…</strong> to point Downstage at a
        different folder; the editor reloads and opens the first script in the
        new library. This is the only place to switch libraries.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Appearance</h3>
      <p>
        Switch between the light and dark themes. Your choice is saved with the
        rest of your preferences and persists across restarts.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Export</h3>
      <p>
        Defaults for the PDF export dialog: page size, style, layout, and
        booklet gutter. These set what the dialog opens with. You can still
        change any of them on a single export without changing the defaults.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Spellcheck</h3>
      <p>
        Turn spellcheck on or off, and review the dictionary of words you've
        added for the current script. Words you added by right-clicking and
        choosing <em>Add to Dictionary</em> appear here, and you can remove any
        you no longer want.
      </p>
    </section>

    <p>
      Quick toggles like showing the preview or the sidebar don't live in
      Settings. They have their own buttons in the toolbar and on the sidebar
      header.
    </p>
  </div>
</template>
