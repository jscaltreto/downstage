import { computed, ref, watch, type Ref } from "vue";

// useDocumentLifecycle is the shared-editor's mechanism for resetting
// per-document transient state (diagnostics, search highlights, stats,
// outline, V1-modal suppression, etc.) when the host swaps the active
// document.
//
// `documentKey` is an opaque, host-provided identifier — the web host uses
// the active draft ID; the desktop host uses the active file's relative
// path. The shared editor MUST NOT know which identity scheme is in use.
//
// The V1-suppression helpers are exposed here (rather than tracked inside
// Editor.vue) so the dismissal is automatically invalidated whenever the
// document changes, without a second watcher on the same key.
export function useDocumentLifecycle(
  documentKey: Ref<string | null>,
  onChange: () => void,
) {
  const dismissedKey = ref<string | null>(null);

  const isV1Suppressed = computed(
    () =>
      dismissedKey.value !== null &&
      dismissedKey.value === documentKey.value,
  );

  const suppressV1 = () => {
    dismissedKey.value = documentKey.value;
  };

  const clearV1Suppression = () => {
    dismissedKey.value = null;
  };

  watch(documentKey, () => {
    // A new document invalidates the previous suppression — otherwise a
    // dismissal on a prior document could silently suppress the V1 modal
    // on a new one (which has the same null identity on desktop without
    // this composable).
    dismissedKey.value = null;
    onChange();
  });

  return { isV1Suppressed, suppressV1, clearV1Suppression };
}
