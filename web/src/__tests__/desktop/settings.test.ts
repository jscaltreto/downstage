// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { reactive } from "vue";
import Settings from "../../desktop/Settings.vue";

// Settings dialog tests. Three real tabs — Library (location + reveal),
// Appearance (theme), Spellcheck (enable toggle + custom wordlist).
// Transient view toggles (show preview, show sidebar) live in the main
// UI, not here, so they're not tested from this surface.

function stubLocalStorage() {
  const store = new Map<string, string>();
  Object.defineProperty(globalThis, "localStorage", {
    value: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => { store.set(k, v); },
    },
    configurable: true,
  });
}

function fakeStore() {
  const state = reactive({
    theme: "system",
    previewHidden: false,
    spellcheckDisabled: false,
  });
  return {
    state,
    setTheme: vi.fn((t: string) => { state.theme = t; }),
  };
}

function fakeWorkspace() {
  const state = reactive({
    sidebarCollapsed: false,
    spellAllowlist: ["Nebula"],
    libraryPath: "/home/user/Documents/Downstage Plays",
  });
  return {
    state,
    toggleSidebar: vi.fn(() => { state.sidebarCollapsed = !state.sidebarCollapsed; }),
    addAllowlistWord: vi.fn(async (word: string) => {
      state.spellAllowlist.push(word);
      return true;
    }),
    removeAllowlistWord: vi.fn(async (word: string) => {
      const i = state.spellAllowlist.indexOf(word);
      if (i >= 0) { state.spellAllowlist.splice(i, 1); return true; }
      return false;
    }),
  };
}

function fakeEnv() {
  return {
    revealLibraryInExplorer: vi.fn(async () => {}),
    getExportPreferences: vi.fn(async () => ({
      pageSize: "letter" as const,
      style: "standard" as const,
      layout: "single" as const,
      bookletGutter: "0.125in",
    })),
    setExportPreferences: vi.fn(async () => {}),
  };
}

describe("Settings dialog", () => {
  it("shows Library, Appearance, Export, and Spellcheck tabs", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();
    const env = fakeEnv();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "library", store: store as any, workspace: workspace as any, env: env as any },
    });
    await flushPromises();

    const tabLabels = wrapper.findAll("nav button").map((b) => b.text());
    expect(tabLabels).toEqual(["Library", "Appearance", "Export", "Spellcheck"]);
  });

  it("Appearance tab shows theme buttons and no sidebar/preview toggles", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "appearance", store: store as any, workspace: workspace as any, env: fakeEnv() as any },
    });
    await flushPromises();

    // Theme options present.
    const labels = wrapper.findAll("button").map((b) => b.text());
    expect(labels).toContain("Light");
    expect(labels).toContain("Dark");
    expect(labels).toContain("Follow System");
    // No sidebar/preview labels anywhere in the tab content.
    const body = wrapper.text();
    expect(body).not.toMatch(/sidebar/i);
    expect(body).not.toMatch(/preview/i);
  });

  it("Appearance tab switches theme via the Store setter", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "appearance", store: store as any, workspace: workspace as any, env: fakeEnv() as any },
    });
    await flushPromises();

    const dark = wrapper.findAll("button").find((b) => b.text() === "Dark")!;
    await dark.trigger("click");
    expect(store.setTheme).toHaveBeenCalledWith("dark");
    expect(store.state.theme).toBe("dark");
  });

  it("Spellcheck tab renders the allowlist and can add words", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "spellcheck", store: store as any, workspace: workspace as any, env: fakeEnv() as any },
    });
    await flushPromises();

    // Seed word renders.
    expect(wrapper.text()).toContain("Nebula");

    // Add a new word.
    const input = wrapper.find<HTMLInputElement>("input[type=text]");
    await input.setValue("Starfall");
    await wrapper.find("form").trigger("submit");
    await flushPromises();
    expect(workspace.addAllowlistWord).toHaveBeenCalledWith("Starfall");
  });

  it("Spellcheck tab's enable toggle is a ToggleSwitch (role=switch)", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "spellcheck", store: store as any, workspace: workspace as any, env: fakeEnv() as any },
    });
    await flushPromises();

    // No checkbox inputs anywhere — the Spellcheck tab uses ToggleSwitch
    // so the app's boolean affordance stays visually consistent.
    const checkboxes = wrapper.findAll('input[type="checkbox"]');
    expect(checkboxes.length).toBe(0);

    // A role=switch button exists and reflects the current state.
    const switches = wrapper.findAll('button[role="switch"]');
    expect(switches.length).toBeGreaterThanOrEqual(1);
    const spellSwitch = switches[0];
    expect(spellSwitch.attributes("aria-checked")).toBe("true");

    await spellSwitch.trigger("click");
    expect(store.state.spellcheckDisabled).toBe(true);
  });
});
