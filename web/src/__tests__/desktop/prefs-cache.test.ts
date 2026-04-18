import { describe, expect, it, vi } from "vitest";
import { createPrefsCache, type PrefsBackend } from "../../desktop/prefs-cache";

interface TestPrefs {
  theme: string;
  previewHidden: boolean;
  sidebarCollapsed: boolean;
}

function makeBackend(initial: TestPrefs, saveDelayMs = 0) {
  const saves: TestPrefs[] = [];
  const backend: PrefsBackend<TestPrefs> & { saves: TestPrefs[] } = {
    saves,
    load: vi.fn(async () => ({ ...initial })),
    save: vi.fn(async (p) => {
      if (saveDelayMs > 0) {
        await new Promise((r) => setTimeout(r, saveDelayMs));
      }
      saves.push({ ...p });
    }),
  };
  return backend;
}

describe("PrefsCache", () => {
  it("lazy-loads on first read and shares the load promise across concurrent calls", async () => {
    const backend = makeBackend({ theme: "system", previewHidden: false, sidebarCollapsed: false });
    const cache = createPrefsCache(backend);

    // Kick off two reads concurrently — they must share one load call.
    const [a, b] = await Promise.all([cache.get(), cache.get()]);

    expect(a).toEqual({ theme: "system", previewHidden: false, sidebarCollapsed: false });
    expect(b).toEqual(a);
    expect(backend.load).toHaveBeenCalledTimes(1);
  });

  it("returns a shallow copy so callers can't mutate the snapshot out-of-band", async () => {
    const backend = makeBackend({ theme: "system", previewHidden: false, sidebarCollapsed: false });
    const cache = createPrefsCache(backend);

    const s = await cache.get();
    s.theme = "dark";

    const s2 = await cache.get();
    expect(s2.theme).toBe("system");
  });

  it("serializes interleaved updates so both fields survive", async () => {
    const backend = makeBackend({ theme: "system", previewHidden: false, sidebarCollapsed: false }, 10);
    const cache = createPrefsCache(backend);

    // Two writers hitting different fields back-to-back. Previous
    // implementation did a read-modify-write in the frontend and would
    // lose one of these updates. With the cache + chain, both survive.
    const a = cache.update({ theme: "dark" });
    const b = cache.update({ sidebarCollapsed: true });
    await Promise.all([a, b]);

    const final = await cache.get();
    expect(final.theme).toBe("dark");
    expect(final.sidebarCollapsed).toBe(true);

    // Two backend writes, in the order they were enqueued. The first
    // carries the theme change; the second carries both.
    expect(backend.saves.length).toBe(2);
    expect(backend.saves[0].theme).toBe("dark");
    expect(backend.saves[0].sidebarCollapsed).toBe(false);
    expect(backend.saves[1].theme).toBe("dark");
    expect(backend.saves[1].sidebarCollapsed).toBe(true);
  });

  it("flush() resolves only after all pending writes settle", async () => {
    const backend = makeBackend({ theme: "system", previewHidden: false, sidebarCollapsed: false }, 30);
    const cache = createPrefsCache(backend);

    // Fire updates without awaiting — this mimics Store's fire-and-forget
    // watcher. flush() is the shutdown primitive that must wait for them.
    void cache.update({ theme: "dark" });
    void cache.update({ sidebarCollapsed: true });

    await cache.flush();

    expect(backend.saves.length).toBe(2);
    expect(backend.saves[1].theme).toBe("dark");
    expect(backend.saves[1].sidebarCollapsed).toBe(true);
  });

  it("a failed write does not halt the chain; subsequent writes still run", async () => {
    const backend: PrefsBackend<TestPrefs> = {
      load: async () => ({ theme: "system", previewHidden: false, sidebarCollapsed: false }),
      save: vi.fn()
        .mockRejectedValueOnce(new Error("transient backend fail"))
        .mockResolvedValue(undefined),
    };
    const cache = createPrefsCache(backend);

    const first = cache.update({ theme: "dark" });
    await expect(first).rejects.toThrow("transient backend fail");

    // The second write still runs despite the first failing.
    await cache.update({ sidebarCollapsed: true });
    expect(backend.save).toHaveBeenCalledTimes(2);
  });

  it("flush() on a quiescent cache is a no-op and resolves immediately", async () => {
    const backend = makeBackend({ theme: "system", previewHidden: false, sidebarCollapsed: false });
    const cache = createPrefsCache(backend);

    await expect(cache.flush()).resolves.toBeUndefined();
    expect(backend.save).not.toHaveBeenCalled();
  });
});
