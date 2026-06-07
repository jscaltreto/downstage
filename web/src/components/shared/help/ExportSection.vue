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
      <h2 class="text-xs font-bold uppercase tracking-[0.14em] text-text-main">PDF export</h2>
      <p v-if="host === 'desktop'">
        Choose <strong class="text-text-main">File → Export → PDF…</strong> or press
        <kbd class="rounded border border-border px-1 font-mono text-[10px]">Ctrl/⌘&shy;E</kbd> to open the export
        dialog. Pick a page size (US Letter or A4), a layout (single-sided
        manuscript or folded booklet), and a style (Manuscript or Acting
        Edition). Your last choices are remembered, so re-exporting after a
        small edit takes a single keystroke.
      </p>
      <p v-else>
        Click <strong class="text-text-main">Export to PDF</strong> in the toolbar
        to render the script in the browser. Pick a page size and style; the PDF
        downloads to wherever your browser normally puts downloads.
      </p>
    </section>

    <section class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Manuscript vs. Acting Edition</h3>
      <p>
        <strong class="text-text-main">Manuscript</strong> is the full-page
        layout you would send to a literary manager or include in a submission.
        Generous margins, dialogue centered on the page.
      </p>
      <p>
        <strong class="text-text-main">Acting Edition</strong> is the compact
        layout you would hand to a cast. Tighter line spacing and character
        cues in the margin, designed to be read from a music stand.
      </p>
    </section>

    <section v-if="host === 'desktop'" class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Booklet layout</h3>
      <p>
        Booklet mode prints two reading pages on each physical sheet, in the
        order needed to fold the stack in half and bind it down the spine. The
        gutter (the extra space along the inner edge) is adjustable in the
        export dialog, so the binding does not eat into your inner margin.
      </p>
    </section>

    <section v-if="host === 'desktop'" class="space-y-2">
      <h3 class="text-[10px] font-bold uppercase tracking-[0.14em] text-text-main">Exporting the .ds source</h3>
      <p>
        Choose <strong class="text-text-main">File → Export → Downstage File…</strong>
        to save a copy of the raw <code class="rounded bg-black/5 px-1 dark:bg-white/5">.ds</code> text file. Use this
        to send the source to a collaborator who also uses Downstage, or to keep
        a snapshot of the script outside your library folder.
      </p>
    </section>

    <p class="text-text-muted/80">
      The exported PDF uses the same renderer as the live preview. What you see
      in the preview pane is what lands on the page.
    </p>
  </div>
</template>
