<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import {
  defaultHelpSection,
  sectionsForHost,
  type HelpHost,
  type HelpSectionId,
  type ShortcutEntry,
} from './help-sections';

const props = withDefaults(
  defineProps<{
    openLink: (url: string) => Promise<void>;
    host?: HelpHost;
    shortcuts?: ShortcutEntry[];
    shortcutsLoading?: boolean;
  }>(),
  {
    host: 'web',
    shortcuts: () => [],
    shortcutsLoading: false,
  },
);

const visibleSections = computed(() => sectionsForHost(props.host));

const activeSection = ref<HelpSectionId>(defaultHelpSection);

// Keep activeSection valid against the current host's visible set. The
// host prop can change (component re-mount in a new host) or the
// registry can grow/shrink; in both cases fall back to the first
// visible section so the content pane never goes blank.
watch(
  [visibleSections, activeSection],
  ([visible, id]) => {
    if (!visible.some((s) => s.id === id)) {
      activeSection.value = visible[0]?.id ?? defaultHelpSection;
    }
  },
  { immediate: true },
);

const currentSection = computed(() =>
  visibleSections.value.find((s) => s.id === activeSection.value)
    ?? visibleSections.value[0],
);
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden @container">
    <!-- Wide layout: left rail + content. Narrow (right-docked drawer at
         ~360px): icon-only top strip + content. The @container threshold
         mirrors HelpTab's pre-existing breakpoint usage. -->
    <div class="flex min-h-0 flex-1 flex-col overflow-hidden @md:flex-row">
      <nav
        class="flex shrink-0 overflow-x-auto border-b border-border bg-[var(--color-toolbar-bg)]
               @md:w-36 @md:flex-col @md:overflow-x-visible @md:overflow-y-auto @md:border-b-0 @md:border-r @md:py-2"
        role="tablist"
        aria-label="Help sections"
      >
        <button
          v-for="section in visibleSections"
          :key="section.id"
          type="button"
          role="tab"
          :aria-selected="activeSection === section.id"
          :title="section.label"
          :aria-label="section.label"
          class="flex items-center gap-2 px-2.5 py-2 text-[10px] font-bold uppercase tracking-[0.14em] transition-colors
                 @md:justify-start @md:rounded-none @md:px-3 @md:py-1.5 @md:text-xs"
          :class="activeSection === section.id
            ? 'bg-brass-500/15 text-accent'
            : 'text-text-muted hover:bg-black/5 hover:text-text-main dark:hover:bg-white/5'"
          @click="activeSection = section.id"
        >
          <component :is="section.icon" class="h-3.5 w-3.5 shrink-0" />
          <span class="hidden normal-case tracking-normal @md:inline">{{ section.label }}</span>
        </button>
      </nav>

      <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
        <component
          :is="currentSection.component"
          :open-link="props.openLink"
          :host="props.host"
          :shortcuts="props.shortcuts"
          :loading="props.shortcutsLoading"
        />
      </div>
    </div>
  </div>
</template>
