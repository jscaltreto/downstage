// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { ref } from "vue";
import { createCommandHandlers, type CommandContext } from "../../desktop/commands";

// Concrete tests against the real command handlers. Each test exercises
// one user-facing outcome: a handler fires, a Workspace/Env method is
// called, an isEnabled predicate reflects current state.

function makeContext(overrides: Partial<CommandContext> = {}): CommandContext {
  const activeContent = ref("");
  const editorContent = ref("");
  const pageStyle = ref("standard");
  const isV1Document = ref(false);
  const isViewingRevision = ref(false);
  const drawerOpen = ref(false);
  const drawerTab = ref<any>("issues");
  const searchRequest = ref<any>({ mode: "find", nonce: 0 });

  const libraryFilesRef = ref<Array<{ path: string; name: string; updatedAt: string }>>([]);
  const workspaceState = {
    activeFile: null as string | null,
    libraryPath: null as string | null,
    viewingRevisionHash: null as string | null,
    externalFile: null as null | { absPath: string; content: string },
  };

  const env: any = {
    renderPDF: vi.fn(async () => new Uint8Array()),
    saveFile: vi.fn(async () => {}),
    openURL: vi.fn(async () => {}),
    openExternalFileDialog: vi.fn(async () => ""),
  };
  const store: any = { state: { previewHidden: false } };
  const workspace: any = {
    state: workspaceState,
    libraryFiles: libraryFilesRef,
    createFile: vi.fn(async (name: string) => `${name}.ds`),
    selectFile: vi.fn(async (_path: string) => ""),
    changeLibraryLocation: vi.fn(async () => "/p/alpha"),
    openExternalFile: vi.fn(async () => ""),
    snapshotFile: vi.fn(async () => {}),
    toggleSidebar: vi.fn(),
    addAllowlistWord: vi.fn(),
    removeAllowlistWord: vi.fn(),
    createFolder: vi.fn(async () => {}),
  };
  const toast = { addToast: vi.fn() };

  return {
    env,
    store,
    workspace,
    toast,
    activeContent,
    editorContent,
    pageStyle,
    isV1Document,
    isViewingRevision,
    flushSave: async () => {},
    editor: { applyFormat: vi.fn() },
    ui: {
      drawerOpen,
      drawerTab,
      searchRequest,
      openPalette: vi.fn(),
      openSettings: vi.fn(),
    },
    ...overrides,
  };
}

function asMap(entries: Array<[string, any]>) {
  return new Map(entries);
}

describe("command handlers", () => {
  it("file.newPlay creates a file when a library is open", async () => {
    const ctx = makeContext();
    ctx.workspace.state.libraryPath = "/p/alpha";
    const cmds = asMap(createCommandHandlers(ctx));
    await cmds.get("file.newPlay")!.handler();
    expect(ctx.workspace.createFile).toHaveBeenCalled();
    expect(ctx.workspace.selectFile).toHaveBeenCalled();
  });

  it("file.saveVersion is disabled when no active file", () => {
    const ctx = makeContext();
    const cmds = asMap(createCommandHandlers(ctx));
    expect(cmds.get("file.saveVersion")!.isEnabled!()).toBe(false);
  });

  it("file.saveVersion is disabled while viewing a revision", () => {
    const ctx = makeContext();
    ctx.workspace.state.activeFile = "play.ds";
    (ctx.isViewingRevision as any).value = true;
    const cmds = asMap(createCommandHandlers(ctx));
    expect(cmds.get("file.saveVersion")!.isEnabled!()).toBe(false);
  });

  it("file.saveVersion snapshots via workspace when enabled", async () => {
    const ctx = makeContext();
    ctx.workspace.state.activeFile = "play.ds";
    const cmds = asMap(createCommandHandlers(ctx));
    expect(cmds.get("file.saveVersion")!.isEnabled!()).toBe(true);
    await cmds.get("file.saveVersion")!.handler();
    expect(ctx.workspace.snapshotFile).toHaveBeenCalledWith("Snapshot play.ds");
  });

  it("file.exportPdf is disabled when the document is V1", () => {
    const ctx = makeContext();
    ctx.workspace.state.activeFile = "play.ds";
    (ctx.isV1Document as any).value = true;
    const cmds = asMap(createCommandHandlers(ctx));
    expect(cmds.get("file.exportPdf")!.isEnabled!()).toBe(false);
  });

  it("file.settings.spellcheck opens Settings on the spellcheck tab", () => {
    const ctx = makeContext();
    const cmds = asMap(createCommandHandlers(ctx));
    cmds.get("file.settings.spellcheck")!.handler();
    expect(ctx.ui.openSettings).toHaveBeenCalledWith("spellcheck");
  });

  it("view.togglePreview flips the store's previewHidden flag", () => {
    const ctx = makeContext();
    const cmds = asMap(createCommandHandlers(ctx));
    cmds.get("view.togglePreview")!.handler();
    expect(ctx.store.state.previewHidden).toBe(true);
    cmds.get("view.togglePreview")!.handler();
    expect(ctx.store.state.previewHidden).toBe(false);
  });

  it("format commands call editor.applyFormat with the right action", () => {
    const ctx = makeContext();
    ctx.workspace.state.activeFile = "play.ds";
    const cmds = asMap(createCommandHandlers(ctx));
    cmds.get("format.bold")!.handler();
    cmds.get("format.scene")!.handler();
    cmds.get("format.pageBreak")!.handler();
    expect(ctx.editor.applyFormat).toHaveBeenCalledWith("bold");
    expect(ctx.editor.applyFormat).toHaveBeenCalledWith("scene");
    expect(ctx.editor.applyFormat).toHaveBeenCalledWith("page-break");
  });

  it("navigate.nextFile cycles forward through the library list", async () => {
    const ctx = makeContext();
    // Assign through the underlying ref (cast is safe — the mock
    // constructs a mutable ref; the production type is a ComputedRef).
    (ctx.workspace.libraryFiles as { value: unknown }).value = [
      { path: "a.ds", name: "a.ds", updatedAt: "" },
      { path: "b.ds", name: "b.ds", updatedAt: "" },
      { path: "c.ds", name: "c.ds", updatedAt: "" },
    ];
    ctx.workspace.state.activeFile = "b.ds";
    const cmds = asMap(createCommandHandlers(ctx));
    cmds.get("navigate.nextFile")!.handler();
    // navigateFile dispatches an async op; wait a microtask.
    await Promise.resolve();
    await Promise.resolve();
    expect(ctx.workspace.selectFile).toHaveBeenCalledWith("c.ds");
  });

  it("edit.find bumps searchRequest nonce (triggers Editor search open)", () => {
    const ctx = makeContext();
    ctx.workspace.state.activeFile = "play.ds";
    const cmds = asMap(createCommandHandlers(ctx));
    const before = ctx.ui.searchRequest.value.nonce;
    cmds.get("edit.find")!.handler();
    expect(ctx.ui.searchRequest.value.nonce).toBe(before + 1);
    expect(ctx.ui.searchRequest.value.mode).toBe("find");
  });

  it("help.about invokes env.showAboutDialog", async () => {
    const ctx = makeContext();
    ctx.env.showAboutDialog = vi.fn(async () => {});
    const cmds = asMap(createCommandHandlers(ctx));
    await cmds.get("help.about")!.handler();
    expect(ctx.env.showAboutDialog).toHaveBeenCalledTimes(1);
  });

  it("view.toggleDrawerDock flips dock between 'bottom' and 'right'", () => {
    const ctx = makeContext();
    (ctx.workspace.state as any).drawerDock = "bottom";
    ctx.workspace.setDrawerDock = vi.fn((d: string) => {
      (ctx.workspace.state as any).drawerDock = d;
    });
    const cmds = asMap(createCommandHandlers(ctx));
    cmds.get("view.toggleDrawerDock")!.handler();
    expect(ctx.workspace.setDrawerDock).toHaveBeenCalledWith("right");
    cmds.get("view.toggleDrawerDock")!.handler();
    expect(ctx.workspace.setDrawerDock).toHaveBeenLastCalledWith("bottom");
  });
});
