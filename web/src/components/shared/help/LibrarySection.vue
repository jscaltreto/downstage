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
        rename it. A folder disappears from the list once its last file moves
        out.
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
