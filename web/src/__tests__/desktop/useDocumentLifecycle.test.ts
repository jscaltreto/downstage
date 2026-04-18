import { describe, expect, it, vi } from "vitest";
import { nextTick, ref } from "vue";
import { useDocumentLifecycle } from "../../core/useDocumentLifecycle";

describe("useDocumentLifecycle", () => {
  it("does not fire onChange at setup", async () => {
    const key = ref<string | null>("a");
    const onChange = vi.fn();
    useDocumentLifecycle(key, onChange);
    await nextTick();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("fires onChange when the key changes", async () => {
    const key = ref<string | null>("a");
    const onChange = vi.fn();
    useDocumentLifecycle(key, onChange);
    key.value = "b";
    await nextTick();
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("fires onChange when the key becomes null", async () => {
    const key = ref<string | null>("a");
    const onChange = vi.fn();
    useDocumentLifecycle(key, onChange);
    key.value = null;
    await nextTick();
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("tracks V1 suppression per key", async () => {
    const key = ref<string | null>("a");
    const { isV1Suppressed, suppressV1 } = useDocumentLifecycle(key, () => {});

    expect(isV1Suppressed.value).toBe(false);
    suppressV1();
    expect(isV1Suppressed.value).toBe(true);

    key.value = "b";
    await nextTick();
    // New document — dismissal must not carry over.
    expect(isV1Suppressed.value).toBe(false);
  });

  it("clearV1Suppression resets suppression immediately", () => {
    const key = ref<string | null>("a");
    const { isV1Suppressed, suppressV1, clearV1Suppression } =
      useDocumentLifecycle(key, () => {});

    suppressV1();
    expect(isV1Suppressed.value).toBe(true);
    clearV1Suppression();
    expect(isV1Suppressed.value).toBe(false);
  });

  it("suppression with null key is never considered suppressed", () => {
    // Otherwise on desktop with no active file, a prior dismissal would
    // silently hide modals on the next document.
    const key = ref<string | null>(null);
    const { isV1Suppressed, suppressV1 } = useDocumentLifecycle(key, () => {});
    suppressV1();
    // dismissedKey === null && documentKey === null would naively match,
    // but we explicitly guard against null so suppression only counts when
    // there's a real active document.
    expect(isV1Suppressed.value).toBe(false);
  });
});
