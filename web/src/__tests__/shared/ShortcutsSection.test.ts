// @vitest-environment happy-dom
import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import ShortcutsSection from "../../components/shared/help/ShortcutsSection.vue";
import type { ShortcutEntry } from "../../components/shared/help-sections";

const noop = async () => {};

describe("ShortcutsSection", () => {
  it("shows the loading placeholder when loading=true", () => {
    const wrapper = mount(ShortcutsSection, {
      props: {
        openLink: noop,
        host: "desktop",
        shortcuts: [],
        loading: true,
      },
    });
    expect(wrapper.text()).toContain("Loading shortcuts");
    expect(wrapper.findAll("kbd")).toHaveLength(0);
  });

  it("shows the fallback notice when loading=false and no shortcuts", () => {
    const wrapper = mount(ShortcutsSection, {
      props: {
        openLink: noop,
        host: "desktop",
        shortcuts: [],
        loading: false,
      },
    });
    expect(wrapper.text()).toContain("Shortcut list unavailable");
  });

  it("renders grouped shortcuts when a populated list is provided", () => {
    const shortcuts: ShortcutEntry[] = [
      { id: "file.save", label: "Save", keys: "Ctrl+S", group: "file" },
      { id: "file.open", label: "Open…", keys: "Ctrl+O", group: "file" },
      { id: "format.bold", label: "Bold", keys: "Ctrl+B", group: "format" },
      { id: "help.toggle", label: "Help", keys: "Ctrl+Shift+/", group: "help" },
    ];
    const wrapper = mount(ShortcutsSection, {
      props: {
        openLink: noop,
        host: "desktop",
        shortcuts,
        loading: false,
      },
    });
    // group headers
    const headers = wrapper.findAll("h3").map((h) => h.text());
    expect(headers).toEqual(["File", "Format", "Help"]);
    // every shortcut row renders the keys in a <kbd>
    const kbds = wrapper.findAll("kbd").map((k) => k.text());
    expect(kbds).toEqual(["Ctrl+S", "Ctrl+O", "Ctrl+B", "Ctrl+Shift+/"]);
  });

  it("uses different intro copy per host", () => {
    const desktop = mount(ShortcutsSection, {
      props: { openLink: noop, host: "desktop", shortcuts: [], loading: false },
    });
    expect(desktop.text()).toContain("mirror the native menu");

    const web = mount(ShortcutsSection, {
      props: { openLink: noop, host: "web", shortcuts: [], loading: false },
    });
    expect(web.text()).toContain("browser editor");
  });
});
