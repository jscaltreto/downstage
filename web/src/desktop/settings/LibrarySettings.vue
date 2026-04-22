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
  <section class="flex flex-col gap-5">
    <header>
      <h3 class="text-base font-semibold text-text-main">Library</h3>
      <p class="text-xs text-text-muted mt-0.5">
        One folder holds every play. Default is
        <code class="px-1 py-0.5 rounded bg-black/5 dark:bg-white/5 font-mono text-[11px]">~/Documents/Downstage Plays</code>.
      </p>
    </header>

    <div>
      <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-text-muted mb-1.5">Current location</p>
      <p class="text-xs text-text-main font-mono truncate mb-2" :title="workspace.state.libraryPath ?? ''">
        {{ workspace.state.libraryPath || 'No library set' }}
      </p>
      <div class="flex gap-1.5">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border text-xs font-semibold text-text-main hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
          @click="emit('change-library')"
        >
          <FolderOpen class="w-3.5 h-3.5" />
          Change…
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border text-xs font-semibold text-text-main hover:bg-black/5 dark:hover:bg-white/5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!workspace.state.libraryPath"
          @click="reveal(env)"
        >
          <ExternalLink class="w-3.5 h-3.5" />
          Reveal in File Explorer
        </button>
      </div>
    </div>
  </section>
</template>
