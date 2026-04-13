import { beforeEach, describe, expect, it, vi } from "vitest";
import { Store } from "../core/store";
import type { EditorEnv, SavedDraft } from "../core/types";

function draft(): SavedDraft {
  return {
    id: "draft-1",
    title: "Play",
    content: "# Play",
    updatedAt: "2026-01-01T00:00:00.000Z",
    spellAllowlist: [],
  };
}

function createEnv() {
  return {
    getAppVersion: () => "test",
    loadDrafts: async () => [],
    saveDrafts: vi.fn(async () => {}),
    loadActiveDraftId: async () => null,
    saveActiveDraftId: async () => {},
  } as unknown as EditorEnv & { saveDrafts: ReturnType<typeof vi.fn> };
}

describe("Store spell allowlist", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      value: {
        getItem: vi.fn(() => null),
        setItem: vi.fn(),
      },
      configurable: true,
    });

    Object.defineProperty(globalThis, "window", {
      value: {
        matchMedia: vi.fn(() => ({
          matches: false,
          onchange: null,
        })),
      },
      configurable: true,
    });

    Object.defineProperty(globalThis, "document", {
      value: {
        documentElement: {
          classList: {
            toggle: vi.fn(),
          },
        },
      },
      configurable: true,
    });
  });

  it("adds deduped words to the active draft and persists them", async () => {
    const env = createEnv();
    const store = new Store(env);
    store.state.drafts = [draft()];
    store.state.activeDraftId = "draft-1";

    await expect(store.addSpellAllowlistWord("Nebula")).resolves.toBe(true);
    await expect(store.addSpellAllowlistWord("nebula")).resolves.toBe(false);

    expect(store.activeDraft()?.spellAllowlist).toEqual(["Nebula"]);
    expect(env.saveDrafts).toHaveBeenCalledTimes(1);
  });

  it("removes words case-insensitively and persists the change", async () => {
    const env = createEnv();
    const store = new Store(env);
    store.state.drafts = [{
      ...draft(),
      spellAllowlist: ["Nebula", "Starfall"],
    }];
    store.state.activeDraftId = "draft-1";

    await expect(store.removeSpellAllowlistWord("nebula")).resolves.toBe(true);

    expect(store.activeDraft()?.spellAllowlist).toEqual(["Starfall"]);
    expect(env.saveDrafts).toHaveBeenCalledTimes(1);
  });

  it("returns false when no active draft exists", async () => {
    const env = createEnv();
    const store = new Store(env);

    await expect(store.addSpellAllowlistWord("Nebula")).resolves.toBe(false);
    await expect(store.removeSpellAllowlistWord("Nebula")).resolves.toBe(false);
    expect(env.saveDrafts).not.toHaveBeenCalled();
  });

  it("trims added words before saving", async () => {
    const env = createEnv();
    const store = new Store(env);
    store.state.drafts = [draft()];
    store.state.activeDraftId = "draft-1";

    await expect(store.addSpellAllowlistWord("  Nebula  ")).resolves.toBe(true);

    expect(store.activeDraft()?.spellAllowlist).toEqual(["Nebula"]);
  });
});
