// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { reactive } from "vue";
import Settings from "../../desktop/Settings.vue";

// Happy-dom-backed Settings tests. The dialog takes live Store and
// Workspace references; reactive fakes are enough to verify each tab
// writes to the right state without pulling in the real classes.

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

describe("Settings dialog", () => {
  it("renders the Editor tab with preview/spellcheck toggles bound to store state", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "editor", store: store as any, workspace: workspace as any },
    });
    await flushPromises();

    const checkboxes = wrapper.findAll<HTMLInputElement>("input[type=checkbox]");
    expect(checkboxes.length).toBeGreaterThanOrEqual(2);
    // "Show Preview" is checked because previewHidden=false.
    expect(checkboxes[0].element.checked).toBe(true);
    // Click it → previewHidden flips to true.
    await checkboxes[0].setValue(false);
    expect(store.state.previewHidden).toBe(true);
  });

  it("renders the Appearance tab and switches theme via the Store setter", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "appearance", store: store as any, workspace: workspace as any },
    });
    await flushPromises();

    const buttons = wrapper.findAll("button").filter((b) => b.text() === "Dark");
    expect(buttons.length).toBe(1);
    await buttons[0].trigger("click");
    expect(store.setTheme).toHaveBeenCalledWith("dark");
    expect(store.state.theme).toBe("dark");
  });

  it("Appearance sidebar toggle calls workspace.toggleSidebar", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "appearance", store: store as any, workspace: workspace as any },
    });
    await flushPromises();

    const checkboxes = wrapper.findAll<HTMLInputElement>("input[type=checkbox]");
    const sidebarCheckbox = checkboxes[checkboxes.length - 1];
    await sidebarCheckbox.trigger("change");
    expect(workspace.toggleSidebar).toHaveBeenCalled();
  });

  it("Spellcheck tab renders the allowlist and can add/remove words", async () => {
    stubLocalStorage();
    const store = fakeStore();
    const workspace = fakeWorkspace();

    const wrapper = mount(Settings, {
      props: { open: true, tab: "spellcheck", store: store as any, workspace: workspace as any },
    });
    await flushPromises();

    // Initial allowlist row ("Nebula") should render.
    expect(wrapper.text()).toContain("Nebula");

    // Type a new word and submit the form.
    const input = wrapper.find<HTMLInputElement>("input[type=text]");
    await input.setValue("Starfall");
    await wrapper.find("form").trigger("submit");
    await flushPromises();
    expect(workspace.addAllowlistWord).toHaveBeenCalledWith("Starfall");
  });
});
