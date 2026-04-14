// @vitest-environment happy-dom
import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import IssuesDrawer from "../components/shared/IssuesDrawer.vue";
import type { EditorDiagnostic } from "../core/types";

function mk(overrides: Partial<EditorDiagnostic>): EditorDiagnostic {
  return {
    from: 0,
    to: 1,
    line: 1,
    col: 1,
    severity: "warning",
    message: "example",
    ...overrides,
  };
}

describe("IssuesDrawer", () => {
  it("shows the empty state when no diagnostics are present and open", () => {
    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics: [], open: true },
    });
    expect(wrapper.text()).toContain("No script issues");
    expect(wrapper.findAll("li")).toHaveLength(0);
  });

  it("renders a row per diagnostic with line:col and message", () => {
    const diagnostics: EditorDiagnostic[] = [
      mk({ line: 3, col: 5, severity: "error", message: "bad act heading" }),
      mk({ line: 7, col: 1, severity: "warning", message: "stray text" }),
    ];

    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics, open: true },
    });

    const rows = wrapper.findAll("li");
    expect(rows).toHaveLength(2);
    expect(rows[0].text()).toContain("3:5");
    expect(rows[0].text()).toContain("bad act heading");
    expect(rows[1].text()).toContain("7:1");
    expect(rows[1].text()).toContain("stray text");
  });

  it("emits jump with the diagnostic when a row is clicked", async () => {
    const diag = mk({ line: 4, col: 2, message: "click me" });
    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics: [diag], open: true },
    });

    await wrapper.find("li").trigger("click");

    const events = wrapper.emitted("jump");
    expect(events).toBeTruthy();
    expect(events?.[0]?.[0]).toEqual(diag);
  });

  it("emits close when the X button is clicked", async () => {
    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics: [], open: true },
    });

    await wrapper.find('button[aria-label="Close script issues"]').trigger("click");

    expect(wrapper.emitted("close")).toBeTruthy();
  });

  it("shows severity count pills for errors and warnings", () => {
    const diagnostics: EditorDiagnostic[] = [
      mk({ severity: "error" }),
      mk({ severity: "error" }),
      mk({ severity: "warning" }),
    ];

    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics, open: true },
    });

    const text = wrapper.text();
    expect(text).toMatch(/2/);
    expect(text).toMatch(/1/);
  });

  it("emits hidden-severities updates when a pill is clicked; hidden pills fade and filter the list", async () => {
    const diagnostics: EditorDiagnostic[] = [
      mk({ severity: "error", message: "boom" }),
      mk({ severity: "warning", message: "stray text" }),
      mk({ severity: "info", message: "a hint" }),
    ];

    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics, open: true, hiddenSeverities: new Set() },
    });

    expect(wrapper.findAll("li")).toHaveLength(3);

    const infoPill = wrapper
      .findAll("button")
      .find((b) => (b.attributes("title") ?? "").includes("info issue"));
    expect(infoPill).toBeTruthy();
    expect(infoPill!.attributes("aria-pressed")).toBe("true");

    await infoPill!.trigger("click");

    const updates = wrapper.emitted("update:hiddenSeverities") as Array<[Set<string>]>;
    expect(updates).toBeTruthy();
    const nextSet = updates[updates.length - 1][0];
    expect(Array.from(nextSet)).toEqual(["info"]);

    // Simulate the parent applying the new value.
    await wrapper.setProps({ hiddenSeverities: nextSet });

    expect(infoPill!.attributes("aria-pressed")).toBe("false");
    expect(infoPill!.classes()).toContain("opacity-40");
    const rowsText = wrapper.findAll("li").map((li) => li.text());
    expect(rowsText.some((t) => t.includes("a hint"))).toBe(false);
    expect(rowsText.some((t) => t.includes("boom"))).toBe(true);
    expect(rowsText.some((t) => t.includes("stray text"))).toBe(true);

    // Clicking again clears the set.
    await infoPill!.trigger("click");
    const allEmits = wrapper.emitted("update:hiddenSeverities") as Array<[Set<string>]>;
    const nextNext = allEmits[allEmits.length - 1][0];
    expect(Array.from(nextNext)).toEqual([]);
    await wrapper.setProps({ hiddenSeverities: nextNext });
    expect(infoPill!.attributes("aria-pressed")).toBe("true");
    expect(wrapper.findAll("li")).toHaveLength(3);
  });

  it("shows a 'all matching issues hidden' state when every pill is toggled off", async () => {
    const diagnostics: EditorDiagnostic[] = [mk({ severity: "warning", message: "x" })];
    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics, open: true, hiddenSeverities: new Set(["warning"]) },
    });

    expect(wrapper.findAll("li")).toHaveLength(0);
    expect(wrapper.text()).toContain("All matching issues hidden");
  });

  it("collapses the section to zero height when closed", () => {
    const wrapper = mount(IssuesDrawer, {
      props: { diagnostics: [], open: false },
    });
    const section = wrapper.find("section");
    expect(section.exists()).toBe(true);
    expect(section.attributes("style")).toContain("height: 0px");
    expect(section.attributes("aria-hidden")).toBe("true");
  });
});
