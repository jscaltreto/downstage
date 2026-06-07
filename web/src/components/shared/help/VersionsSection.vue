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
      <h2 class="text-xs font-bold uppercase tracking-[0.14em] text-text-main">What versions are</h2>
      <p>
        A version is a named snapshot of one script at a point in time. Save one
        before a big rewrite, and you can come back to it later. Each script has
        its own history. Saving a version for one file does not touch any other
        file in your library.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Save a version</h3>
      <p>
        Choose <strong class="text-text-main">File → Save Version…</strong> or
        press <kbd class="rounded border border-border px-1 font-mono text-[10px]">Ctrl/⌘&shy;⇧S</kbd>.
        Downstage asks for a short label so you can recognize it later. Something
        like "end of act II rewrite" or "before cutting the duet" works well.
      </p>
      <p>
        The autosave that runs as you type does <em>not</em> create a version.
        Versions are only made when you ask for one.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Browse and compare</h3>
      <p>
        Open the <strong class="text-text-main">Versions</strong> list from the
        sidebar header to see every saved snapshot of the current script. Click a
        row to open that snapshot in the editor (read-only, so you can't edit
        history by accident). The menu on the right of each row lets you:
      </p>
      <ul class="ml-4 list-disc space-y-1">
        <li><strong class="text-text-main">Compare with current</strong>. Show this snapshot side by side with what's in the editor now.</li>
        <li><strong class="text-text-main">Compare with another version</strong>. Pick a second row to compare against.</li>
        <li><strong class="text-text-main">Restore</strong>. Replace the live script with this snapshot. Downstage saves a version of the current state first, so the work you're about to overwrite is not lost.</li>
        <li><strong class="text-text-main">Hide</strong>. Collapse a noisy version out of the list. It is still saved. Unhide it from the filter menu at the top of the panel.</li>
      </ul>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Where versions are stored</h3>
      <p>
        Versions live in a hidden
        <code class="rounded bg-black/5 px-1 dark:bg-white/5">.downstage/</code> folder inside your library. Under the
        hood it's a standard Git repository, so if you're comfortable with Git you
        can back it up to GitHub or another remote from a terminal. Built-in sync
        is on the roadmap.
      </p>
    </section>
  </div>
</template>
