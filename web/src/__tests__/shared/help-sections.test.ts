import { describe, expect, it } from "vitest";
import {
  defaultHelpSection,
  helpSections,
  sectionsForHost,
  shortcutGroupOrder,
  sortShortcuts,
  type HelpSectionId,
  type ShortcutEntry,
} from "../../components/shared/help-sections";

describe("helpSections registry", () => {
  it("exports exactly seven sections in the documented order", () => {
    const ids = helpSections.map((s) => s.id);
    expect(ids).toEqual<HelpSectionId[]>([
      "getting-started",
      "writing",
      "versions",
      "library",
      "export",
      "shortcuts",
      "settings",
    ]);
  });

  it("every section has a non-empty label, an icon, and a component", () => {
    for (const section of helpSections) {
      expect(section.label).not.toBe("");
      expect(section.icon).toBeTruthy();
      expect(section.component).toBeTruthy();
    }
  });

  it("default section is one of the registered IDs", () => {
    expect(helpSections.map((s) => s.id)).toContain(defaultHelpSection);
  });

  it("every section advertises at least one host", () => {
    for (const section of helpSections) {
      expect(section.hosts.length).toBeGreaterThan(0);
    }
  });
});

describe("sectionsForHost", () => {
  it("desktop sees the full set", () => {
    expect(sectionsForHost("desktop").map((s) => s.id)).toEqual<HelpSectionId[]>([
      "getting-started",
      "writing",
      "versions",
      "library",
      "export",
      "shortcuts",
      "settings",
    ]);
  });

  it("web hides desktop-only sections (versions, library, settings)", () => {
    expect(sectionsForHost("web").map((s) => s.id)).toEqual<HelpSectionId[]>([
      "getting-started",
      "writing",
      "export",
      "shortcuts",
    ]);
  });

  it("the default section is visible to both hosts", () => {
    expect(sectionsForHost("desktop").map((s) => s.id)).toContain(defaultHelpSection);
    expect(sectionsForHost("web").map((s) => s.id)).toContain(defaultHelpSection);
  });
});

describe("sortShortcuts", () => {
  function entry(id: string, group: string): ShortcutEntry {
    return { id, label: id, keys: "X", group };
  }

  it("orders entries by shortcutGroupOrder, preserving within-group input order", () => {
    const input: ShortcutEntry[] = [
      entry("help.toggle", "help"),
      entry("file.save", "file"),
      entry("format.bold", "format"),
      entry("file.open", "file"),
      entry("edit.find", "edit"),
    ];
    const sorted = sortShortcuts(input);
    expect(sorted.map((s) => s.id)).toEqual([
      "file.save",
      "file.open",
      "format.bold",
      "help.toggle",
      "edit.find",
    ].sort((a, b) => {
      // Verify against the documented ordering manually:
      // file (save, open) → edit → format → help
      const order = ["file.save", "file.open", "edit.find", "format.bold", "help.toggle"];
      return order.indexOf(a) - order.indexOf(b);
    }));
  });

  it("places unknown-group entries last", () => {
    const input: ShortcutEntry[] = [
      entry("custom.x", "mystery"),
      entry("file.save", "file"),
    ];
    const sorted = sortShortcuts(input);
    expect(sorted.map((s) => s.id)).toEqual(["file.save", "custom.x"]);
  });

  it("does not mutate the input array", () => {
    const input: ShortcutEntry[] = [
      entry("help.toggle", "help"),
      entry("file.save", "file"),
    ];
    const before = [...input];
    sortShortcuts(input);
    expect(input).toEqual(before);
  });
});

describe("shortcutGroupOrder", () => {
  it("matches the native menu's left-to-right layout", () => {
    expect(shortcutGroupOrder).toEqual([
      "file",
      "edit",
      "view",
      "navigate",
      "insert",
      "format",
      "help",
    ]);
  });
});
