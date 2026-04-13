<script setup lang="ts">
import BaseModal from './BaseModal.vue';
import { AlertTriangle, BookOpen, Download } from 'lucide-vue-next';

defineProps<{
  open: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'open-quick-reference'): void;
}>();
</script>

<template>
  <BaseModal
    :open="open"
    title="Welcome to Downstage"
    kicker="Web Editor"
    @close="emit('close')"
  >
    <div class="flex flex-col gap-6 py-1">
      <div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4">
        <div class="flex items-start gap-3">
          <div class="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-amber-500/15">
            <AlertTriangle class="h-5 w-5 text-amber-500" />
          </div>
          <div class="space-y-2">
            <p class="text-sm font-bold text-text-main">Your work stays yours.</p>
            <p class="text-sm leading-relaxed text-text-muted">
              Downstage never sends your work to a server, and we don't use tracking cookies. Note that
              this means if you clear browser data, switch devices, or lose this browser profile, those
              drafts can disappear.
            </p>
          </div>
        </div>
      </div>

      <div class="rounded-xl border border-border bg-black/5 p-4 dark:bg-white/5">
        <div class="flex items-start gap-3">
          <div class="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-brass-500/15">
            <Download class="h-5 w-5 text-brass-500" />
          </div>
          <div class="space-y-2">
            <p class="text-sm font-bold text-text-main">Save drafts you care about.</p>
            <p class="text-sm leading-relaxed text-text-muted">
              Use <strong class="text-text-main">Save .ds</strong> regularly to keep a real copy on your
              device. Treat the browser draft list as convenient scratch storage, not your only backup.
            </p>
          </div>
        </div>
      </div>

      <div class="rounded-xl border border-border/80 bg-[var(--color-page-surface)] p-4">
        <div class="flex items-start gap-3">
          <div class="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-black/5 dark:bg-white/5">
            <BookOpen class="h-5 w-5 text-text-main" />
          </div>
          <div class="space-y-2">
            <p class="text-sm font-bold text-text-main">Need a quick syntax refresher?</p>
            <p class="text-sm leading-relaxed text-text-muted">
              Open the quick reference whenever you need it, or view the
              <a
                class="font-bold text-brass-500 underline decoration-brass-500/40 underline-offset-2 hover:text-brass-400"
                href="https://www.getdownstage.com/syntax/"
                target="_blank"
                rel="noreferrer"
              >
                Syntax Guide
              </a>
              for the full spec with examples.
            </p>
          </div>
        </div>
      </div>

      <div class="flex flex-col gap-3 pt-1 sm:flex-row">
        <button
          class="flex-1 rounded-lg border border-border px-4 py-2.5 text-sm font-bold text-text-main transition-colors hover:bg-black/5 dark:hover:bg-white/5"
          @click="emit('open-quick-reference')"
        >
          Open Quick Reference
        </button>
        <button
          class="flex-1 rounded-lg bg-brass-500 px-4 py-2.5 text-sm font-bold text-black transition-colors hover:bg-brass-400"
          @click="emit('close')"
        >
          Start Writing
        </button>
      </div>
    </div>
  </BaseModal>
</template>
