import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { Store } from "../core/store";
import type { EditorEnv, EditorPreferences, SavedDraft } from "../core/types";

function draft(): SavedDraft {
  return {
    id: "draft-1",
    title: "Play",
    content: "# Play",
    updatedAt: "2026-01-01T00:00:00.000Z",
    spellAllowlist: [],
  };
}

function createEnv(prefs?: Partial<EditorPreferences>) {
  const stored: EditorPreferences = {
    theme: "system",
    previewHidden: false,
    spellcheckDisabled: false,
    ...prefs,
  };
  return {
    getAppVersion: () => "test",
    loadDrafts: async () => [],
    saveDrafts: vi.fn(async () => {}),
    loadActiveDraftId: async () => null,
    saveActiveDraftId: async () => {},
    getEditorPreferences: vi.fn(async () => ({ ...stored })),
    setEditorPreferences: vi.fn(async (p: EditorPreferences) => {
      Object.assign(stored, p);
    }),
  } as unknown as EditorEnv & {
    saveDrafts: ReturnType<typeof vi.fn>;
    getEditorPreferences: ReturnType<typeof vi.fn>;
    setEditorPreferences: ReturnType<typeof vi.fn>;
  };
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

describe("Store editor preferences", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      value: { getItem: vi.fn(() => null), setItem: vi.fn() },
      configurable: true,
    });
    Object.defineProperty(globalThis, "window", {
      value: {
        matchMedia: vi.fn(() => ({ matches: false, onchange: null })),
      },
      configurable: true,
    });
    Object.defineProperty(globalThis, "document", {
      value: {
        documentElement: { classList: { toggle: vi.fn() } },
      },
      configurable: true,
    });
  });

  it("loads all three editor prefs from env on init()", async () => {
    const env = createEnv({ theme: "dark", previewHidden: true, spellcheckDisabled: true });
    const store = new Store(env);
    await store.init();

    expect(env.getEditorPreferences).toHaveBeenCalledTimes(1);
    expect(store.state.theme).toBe("dark");
    expect(store.state.previewHidden).toBe(true);
    expect(store.state.spellcheckDisabled).toBe(true);
  });

  it("hydration guard: mutating state before init() does NOT persist", async () => {
    const env = createEnv();
    const store = new Store(env);

    // Pre-init mutation — if the guard were missing, this would call
    // setEditorPreferences with the placeholder default and clobber the
    // real stored value the user had from a previous session.
    store.state.theme = "dark";
    await nextTick();

    expect(env.setEditorPreferences).not.toHaveBeenCalled();
  });

  it("persists full editor-pref snapshot after init() on any field change", async () => {
    const env = createEnv();
    const store = new Store(env);
    await store.init();

    store.state.previewHidden = true;
    await nextTick();

    expect(env.setEditorPreferences).toHaveBeenCalledTimes(1);
    expect(env.setEditorPreferences).toHaveBeenCalledWith({
      theme: "system",
      previewHidden: true,
      spellcheckDisabled: false,
    });
  });
});
