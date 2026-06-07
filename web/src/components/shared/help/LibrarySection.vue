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
      <h2 class="text-xs font-bold uppercase tracking-[0.14em] text-text-main">The library folder</h2>
      <p>
        Your library is a single folder on your disk. Every
        <code class="rounded bg-black/5 px-1 dark:bg-white/5">.ds</code> file inside it appears in the file list on
        the left. The first time you open Downstage Write, it asks you to pick
        that folder. Point it at a folder of scripts you already have, or create
        an empty one and start fresh. You can switch to a different library later
        from <strong class="text-text-main">Settings → Library</strong>.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Folders and organization</h3>
      <p>
        Right-click a folder in the file list to add a sub-folder inside it.
        Drag a script onto a folder to move it there. Right-click a script to
        rename it or delete it. A folder disappears from the list once its
        last file moves out.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Deleting and restoring scripts</h3>
      <p>
        Right-click a script and choose <strong class="text-text-main">Delete…</strong>
        to remove it from your library. Deleted scripts move to a
        <strong class="text-text-main">Deleted</strong> section at the bottom
        of the file list, shown with strikethrough. Right-click any row there
        to <strong class="text-text-main">Restore from last version</strong>
        (the file comes back at its most recent saved state) or
        <strong class="text-text-main">Permanently delete</strong> (gone for
        good, can't be undone from inside the app).
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Review Library Changes</h3>
      <p>
        When other files in your library have changes that haven't been
        snapshotted yet, the status bar at the bottom of the window shows
        <strong class="text-text-main">N uncommitted</strong> in amber. Click
        it to open <strong class="text-text-main">Review Library Changes</strong>,
        which groups the drift into <em>Plays</em> (your scripts),
        <em>Library settings</em> (Downstage's own bookkeeping), and
        <em>Other</em>. Commit or discard rows individually, or use
        <strong class="text-text-main">Commit all</strong> /
        <strong class="text-text-main">Discard all</strong> per section.
        This is also where scripts that were edited outside Downstage show
        up so you can fold them into history.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Opening scripts from outside your library</h3>
      <p>
        Choose <strong class="text-text-main">File → Open…</strong> to read a
        <code class="rounded bg-black/5 px-1 dark:bg-white/5">.ds</code> file from anywhere on disk, useful when a
        collaborator emails you a script. Files opened this way are read-only.
        Click <strong class="text-text-main">Add to Library</strong> in the banner
        across the top of the editor to copy the script into your library so you
        can edit it.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">The hidden .downstage folder</h3>
      <p>
        Downstage keeps version history and per-script spellcheck words in a
        hidden <code class="rounded bg-black/5 px-1 dark:bg-white/5">.downstage/</code> folder inside your library.
        Leave it alone. The app reads and writes it for you.
      </p>
    </section>
  </div>
</template>
