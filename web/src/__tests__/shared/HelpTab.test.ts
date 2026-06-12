// @vitest-environment happy-dom
import { describe, expect, it } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import HelpTab from "../../components/shared/HelpTab.vue";
import { sectionsForHost } from "../../components/shared/help-sections";

describe("HelpTab", () => {
  it("desktop host shows every desktop-eligible section", async () => {
    const wrapper = mount(HelpTab, {
      props: {
        openLink: async () => {},
        host: "desktop",
        shortcuts: [],
        shortcutsLoading: false,
      },
    });
    await flushPromises();
    const tabs = wrapper.findAll('[role="tab"]');
    const expected = sectionsForHost("desktop");
    expect(tabs).toHaveLength(expected.length);
    expect(tabs.map((t) => t.attributes("title"))).toEqual(expected.map((s) => s.label));
  });

  it("web host hides desktop-only sections", async () => {
    const wrapper = mount(HelpTab, {
      props: {
        openLink: async () => {},
        host: "web",
      },
    });
    await flushPromises();
    const tabs = wrapper.findAll('[role="tab"]');
    const expected = sectionsForHost("web");
    expect(tabs).toHaveLength(expected.length);
    expect(tabs.map((t) => t.attributes("title"))).toEqual(expected.map((s) => s.label));
    // The three sections we deliberately filter on web
    const titles = tabs.map((t) => t.attributes("title"));
    expect(titles).not.toContain("Versions");
    expect(titles).not.toContain("Library");
    expect(titles).not.toContain("Settings");
  });

  it("marks the default section as selected on mount", () => {
    const wrapper = mount(HelpTab, {
      props: { openLink: async () => {}, host: "desktop", shortcuts: [], shortcutsLoading: false },
    });
    const selected = wrapper.findAll('[role="tab"]').filter((t) =>
      t.attributes("aria-selected") === "true",
    );
    expect(selected).toHaveLength(1);
    expect(selected[0].attributes("title")).toBe("Getting Started");
  });

  it("switches active section when a tab is clicked", async () => {
    const wrapper = mount(HelpTab, {
      props: { openLink: async () => {}, host: "desktop", shortcuts: [], shortcutsLoading: false },
    });
    const shortcutsTab = wrapper
      .findAll('[role="tab"]')
      .find((t) => t.attributes("title") === "Shortcuts");
    expect(shortcutsTab).toBeDefined();
    await shortcutsTab!.trigger("click");
    expect(shortcutsTab!.attributes("aria-selected")).toBe("true");
  });

  it("falls back to the first visible section if active becomes invalid after host change", async () => {
    // Start on desktop, switch to Versions (desktop-only), then re-mount as
    // web — the original activeSection ("versions") is no longer visible
    // and should fall back to the first web-visible section.
    const wrapper = mount(HelpTab, {
      props: { openLink: async () => {}, host: "desktop", shortcuts: [], shortcutsLoading: false },
    });
    const versionsTab = wrapper
      .findAll('[role="tab"]')
      .find((t) => t.attributes("title") === "Versions");
    await versionsTab!.trigger("click");
    expect(versionsTab!.attributes("aria-selected")).toBe("true");

    await wrapper.setProps({ host: "web" });
    await flushPromises();

    const selected = wrapper.findAll('[role="tab"]').filter((t) =>
      t.attributes("aria-selected") === "true",
    );
    expect(selected).toHaveLength(1);
    // First visible web section is Getting Started.
    expect(selected[0].attributes("title")).toBe("Getting Started");
  });
});
