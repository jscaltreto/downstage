// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import CommandPalette from "../../desktop/CommandPalette.vue";
import type { CommandMeta, DesktopCapabilities, LibraryFile } from "../../desktop/types";

// CommandPalette.vue imports dispatchCommand directly from
// dispatcher-registry (not via env), so the spy has to live at the
// module level. Mock it once for the whole file; tests that need to
// assert dispatch behavior reach in via `mockDispatchCommand`.
vi.mock("../../desktop/dispatcher-registry", () => ({
  dispatchCommand: vi.fn(),
}));
import { dispatchCommand as mockDispatchCommand } from "../../desktop/dispatcher-registry";

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
  beforeEach(() => {
    vi.mocked(mockDispatchCommand).mockClear();
  });

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

  it("greys out disabled commands and refuses to execute them on click or Enter", async () => {
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

    // (a) User-visible disabled signal.
    expect(saveItem.classes().some((c) => c.includes("cursor-not-allowed"))).toBe(true);

    // (b) Click on the disabled row must NOT dispatch and must NOT
    // close the palette.
    await saveItem.trigger("click");
    await flushPromises();
    expect(mockDispatchCommand).not.toHaveBeenCalled();
    expect(wrapper.emitted("close")).toBeFalsy();

    // (c) Drive Enter against the disabled row specifically. The
    // palette opens with the first row selected; ArrowDown to reach
    // "Save Version" (third visible row in baseCommands order). Then
    // press Enter on the input.
    const input = wrapper.find("input");
    await input.trigger("keydown", { key: "ArrowDown" });
    await input.trigger("keydown", { key: "ArrowDown" });
    // Cross-check selection landed on the disabled row before
    // pressing Enter — otherwise the Enter assertion proves nothing.
    // The palette flags the selected row with bg-brass-500/15.
    const selected = wrapper.findAll("li").find((li) =>
      li.classes().some((c) => c.includes("bg-brass-500/15")),
    );
    expect(selected, "no row appears visually selected").toBeTruthy();
    expect(selected!.text()).toContain("Save Version");

    await input.trigger("keydown", { key: "Enter" });
    await flushPromises();
    expect(mockDispatchCommand).not.toHaveBeenCalled();
    expect(wrapper.emitted("close")).toBeFalsy();
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
