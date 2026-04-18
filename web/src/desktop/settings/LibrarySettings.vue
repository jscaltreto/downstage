<script setup lang="ts">
import { FolderOpen, ExternalLink } from 'lucide-vue-next';
import type { Workspace } from '../workspace';
import type { DesktopCapabilities } from '../types';

// Library settings. The library is a single per-user directory (default
// ~/Documents/Downstage Plays) where all plays live. It is relocatable
// but only one library is active at a time. Reveal opens the folder in
// the host's file explorer; Change… is the only location-management
// entrypoint left in the app since the File menu no longer has
// "Open Folder…".

defineProps<{
  workspace: Workspace;
  env: DesktopCapabilities;
}>();

const emit = defineEmits<{
  (e: 'change-library'): void;
}>();

async function reveal(env: DesktopCapabilities) {
  try {
    await env.revealLibraryInExplorer();
  } catch {
    // No-op on user-visible channel — the host will surface backend
    // errors via toast if the call fails.
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <h3 class="text-sm font-bold text-text-main">Library</h3>

    <div class="rounded-lg border border-border bg-black/5 p-4 dark:bg-white/5">
      <p class="text-xs font-bold uppercase tracking-[0.15em] text-text-muted mb-2">Current location</p>
      <p class="text-sm text-text-main font-mono truncate" :title="workspace.state.libraryPath ?? ''">
        {{ workspace.state.libraryPath || 'No library set' }}
      </p>
    </div>

    <div class="flex gap-2">
      <button
        type="button"
        class="flex-1 inline-flex items-center justify-center gap-2 px-3 py-2 rounded-md border border-border text-sm font-bold text-text-main hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
        @click="emit('change-library')"
      >
        <FolderOpen class="w-4 h-4" />
        Change…
      </button>
      <button
        type="button"
        class="flex-1 inline-flex items-center justify-center gap-2 px-3 py-2 rounded-md border border-border text-sm font-bold text-text-main hover:bg-black/5 dark:hover:bg-white/5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="!workspace.state.libraryPath"
        @click="reveal(env)"
      >
        <ExternalLink class="w-4 h-4" />
        Reveal in File Explorer
      </button>
    </div>

    <p class="text-xs text-text-muted leading-relaxed">
      Your library is one folder that holds every play. The default is
      <code class="px-1 py-0.5 rounded bg-black/5 dark:bg-white/5 font-mono">~/Documents/Downstage Plays</code>.
      Change the location if you prefer a different folder — all plays in the
      new folder appear in the sidebar, and Downstage creates a Git repo there
      to version your work.
    </p>
  </div>
</template>
