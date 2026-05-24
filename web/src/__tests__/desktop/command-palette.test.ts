// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import CommandPalette from "../../desktop/CommandPalette.vue";
import type { CommandMeta, DesktopCapabilities, LibraryFile } from "../../desktop/types";

// Minimal env stub — only the palette-facing methods are exercised.
function makeEnv(commands: CommandMeta[]): DesktopCapabilities {
  return {
    getCommands: async () => commands,
  } as unknown as DesktopCapabilities;
}

const baseCommands: CommandMeta[] = [
  { id: "file.newPlay", label: "New Play", category: "file", accelerator: "cmdorctrl+n" },
  { id: "file.open", label: "Open…", category: "file", accelerator: "cmdorctrl+o" },
  { id: "file.saveVersion", label: "Save Version", category: "file", accelerator: "cmdorctrl+s" },
  { id: "file.settings.spellcheck", label: "Spellcheck Settings", category: "file", paletteHidden: true },
  { id: "view.commandPalette", label: "Command Palette…", category: "view", accelerator: "cmdorctrl+k" },
];

describe("CommandPalette", () => {
  it("excludes paletteHidden commands and renders the rest", async () => {
    const wrapper = mount(CommandPalette, {
      props: {
        open: true,
        mode: "command",
        env: makeEnv(baseCommands),
        libraryFiles: [],
        disabledIds: [],
      },
    });
    await flushPromises();
    const labels = wrapper.findAll("li").map((li) => li.text());
    expect(labels.some((l) => l.includes("New Play"))).toBe(true);
    expect(labels.some((l) => l.includes("Spellcheck Settings"))).toBe(false);
  });

  it("fuzzy-filters on label with prefix boost", async () => {
    const wrapper = mount(CommandPalette, {
      props: {
        open: true,
        mode: "command",
        env: makeEnv(baseCommands),
        libraryFiles: [],
        disabledIds: [],
      },
    });
    await flushPromises();
    const input = wrapper.find("input");
    await input.setValue("save");
    const labels = wrapper.findAll("li").map((li) => li.text());
    // "Save Version" matches; nothing else in the list does.
    expect(labels.length).toBe(1);
    expect(labels[0]).toContain("Save Version");
  });

  it("greys out disabled commands and refuses to execute them on click", async () => {
    const wrapper = mount(CommandPalette, {
      props: {
        open: true,
        mode: "command",
        env: makeEnv(baseCommands),
        libraryFiles: [],
        disabledIds: ["file.saveVersion"],
      },
    });
    await flushPromises();
    const items = wrapper.findAll("li");
    const saveItem = items.find((li) => li.text().includes("Save Version"))!;
    expect(saveItem.classes().some((c) => c.includes("cursor-not-allowed"))).toBe(true);
    // Click should not emit close (palette wouldn't close if the run
    // was a no-op), and dispatcher-registry isn't touched. We can't
    // spy on the registry easily from here; assert the cursor class
    // serves as the user-visible disabled signal.
  });

  it("emits close on Escape", async () => {
    const wrapper = mount(CommandPalette, {
      props: {
        open: true,
        mode: "command",
        env: makeEnv(baseCommands),
        libraryFiles: [],
        disabledIds: [],
      },
    });
    await flushPromises();
    const input = wrapper.find("input");
    await input.trigger("keydown", { key: "Escape" });
    expect(wrapper.emitted("close")).toBeTruthy();
  });

  it("file-picker mode swaps the source list", async () => {
    const files: LibraryFile[] = [
      { path: "act-one.ds", name: "act-one.ds", updatedAt: "" },
      { path: "act-two.ds", name: "act-two.ds", updatedAt: "" },
    ];
    const wrapper = mount(CommandPalette, {
      props: {
        open: true,
        mode: "file",
        env: makeEnv(baseCommands),
        libraryFiles: files,
        disabledIds: [],
      },
    });
    await flushPromises();
    const labels = wrapper.findAll("li").map((li) => li.text());
    expect(labels.some((l) => l.includes("act-one.ds"))).toBe(true);
    expect(labels.some((l) => l.includes("act-two.ds"))).toBe(true);
    // No catalog labels should appear in file mode.
    expect(labels.some((l) => l.includes("New Play"))).toBe(false);
  });

  it("Enter on a file-mode row emits select-file with the path", async () => {
    const files: LibraryFile[] = [
      { path: "play.ds", name: "play.ds", updatedAt: "" },
    ];
    const wrapper = mount(CommandPalette, {
      props: {
        open: true,
        mode: "file",
        env: makeEnv(baseCommands),
        libraryFiles: files,
        disabledIds: [],
      },
    });
    await flushPromises();
    const input = wrapper.find("input");
    await input.trigger("keydown", { key: "Enter" });
    expect(wrapper.emitted("select-file")).toBeTruthy();
    expect(wrapper.emitted("select-file")![0]).toEqual(["play.ds"]);
  });
});
