<script setup lang="ts">
import type { Store, Theme } from '../../core/store';
import ButtonRadioGroup from '../../components/shared/ButtonRadioGroup.vue';
import type { ButtonRadioOption } from '../../components/shared/button-radio-group';

// Appearance preferences. Only theme today. Sidebar visibility and
// preview visibility used to live here but they're transient view
// toggles, not persisted preferences — those belong to the main UI
// (sidebar chevron, preview eye button, menu, keyboard shortcuts).

const props = defineProps<{
  store: Store;
}>();

const themeOptions: ButtonRadioOption<Theme>[] = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'system', label: 'System' },
];
</script>

<template>
  <section class="flex flex-col gap-5">
    <header>
      <h3 class="text-base font-semibold text-text-main">Appearance</h3>
      <p class="text-xs text-text-muted mt-0.5">How the app looks across windows.</p>
    </header>

    <div class="flex flex-col gap-1.5">
      <p class="text-[11px] font-semibold uppercase tracking-[0.1em] text-text-muted">Theme</p>
      <ButtonRadioGroup
        :model-value="store.state.theme"
        :options="themeOptions"
        aria-label="Theme"
        :columns="3"
        size="compact"
        class="max-w-[320px]"
        @update:model-value="(v: Theme) => store.setTheme(v)"
      />
    </div>
  </section>
</template>
