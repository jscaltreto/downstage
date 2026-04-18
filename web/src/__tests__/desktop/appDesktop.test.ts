// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import AppDesktop from "../../AppDesktop.vue";
import type { DesktopCapabilities, LibraryFile, Revision } from "../../desktop/types";
import { dispatchCommand } from "../../desktop/dispatcher-registry";

interface RecordedEnv extends DesktopCapabilities {
  _calls: string[];
  _setOpenReturn: (path: string) => void;
  _setFiles: (files: LibraryFile[]) => void;
  _setContent: (path: string, content: string) => void;
  _writeDelayMs: number;
  _setWriteDelay: (ms: number) => void;
}

function stubDom() {
  const ls = new Map<string, string>();
  Object.defineProperty(globalThis, "localStorage", {
    value: {
      getItem: (k: string) => ls.get(k) ?? null,
      setItem: (k: string, v: string) => { ls.set(k, v); },
      removeItem: (k: string) => { ls.delete(k); },
    },
    configurable: true,
  });

  Object.defineProperty(globalThis, "matchMedia", {
    value: () => ({
      matches: false,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
    }),
    configurable: true,
  });
}

function createEnv(init: { files?: LibraryFile[]; openReturn?: string } = {}): RecordedEnv {
  let openReturn = init.openReturn ?? "/projects/alpha";
  let files: LibraryFile[] = init.files ?? [];
  const contents: Record<string, string> = {};
  const revisions: Revision[] = [];
  const calls: string[] = [];
  let writeDelayMs = 0;

  const record = (name: string) => calls.push(name);

  const env: RecordedEnv = {
    _calls: calls,
    _setOpenReturn: (p) => { openReturn = p; },
    _setFiles: (f) => { files = f; },
    _setContent: (p, c) => { contents[p] = c; },
    _writeDelayMs: 0,
    _setWriteDelay: (ms) => { writeDelayMs = ms; },

    // EditorEnv no-ops
    parse: async () => ({ errors: [] }),
    diagnostics: async () => ({ diagnostics: [] }),
    spellcheckContext: async () => ({ allowWords: [], ignoredRanges: [] }),
    upgradeV1: async () => ({ source: "", changed: false }),
    completion: async () => ({ isIncomplete: false, items: [] }),
    codeActions: async () => ({ uri: "", actions: [] }),
    documentSymbols: async () => ({ symbols: [] }),
    semanticTokens: async () => new Uint32Array(),
    tokenTypeNames: async () => [],
    stats: async () => ({
      acts: 0, scenes: 0, songs: 0, totalWords: 0, dialogueWords: 0,
      lines: 0, stageDirections: 0, stageDirectionWords: 0, characters: [],
      runtime: { preset: "", wordsPerMinute: 0, pauseFactor: 0, dialogueWords: 0, minutes: 0 },
    }),
    renderHTML: async () => "",
    renderPDF: async () => new Uint8Array(),
    loadDrafts: async () => [],
    saveDrafts: async () => {},
    loadActiveDraftId: async () => null,
    saveActiveDraftId: async () => {},
    saveFile: async () => {},
    importLocalFile: async () => null,
    openURL: async () => {},
    getAppVersion: () => "test",

    // LibraryEnv with call recording
    changeLibraryLocation: async () => { record("changeLibraryLocation"); return openReturn; },
    getLibraryFiles: async () => { record("getLibraryFiles"); return files; },
    readLibraryFile: async (p) => { record(`readLibraryFile:${p}`); return contents[p] ?? ""; },
    writeLibraryFile: async (p, c) => {
      record(`writeLibraryFile:${p}`);
      if (writeDelayMs > 0) {
        await new Promise((r) => setTimeout(r, writeDelayMs));
      }
      contents[p] = c;
      record(`writeLibraryFile:${p}:done`);
    },
    createLibraryFile: async (name) => {
      record(`createLibraryFile:${name}`);
      const path = `${name}.ds`;
      files = [...files, { path, name: path, updatedAt: "" }];
      contents[path] = "";
      return path;
    },
    snapshotFile: async (p, m) => { record(`snapshotFile:${p}:${m}`); },
    getRevisions: async (p, _limit) => { record(`getRevisions:${p}`); return revisions; },
    readFileAtRevision: async (p, h) => { record(`readFileAtRevision:${p}:${h}`); return contents[p] ?? ""; },
    getFileGitStatus: async (p) => {
      record(`getFileGitStatus:${p}`);
      return { dirty: false, headAt: "", hasHead: false, untracked: true, missing: false };
    },
    getEditorPreferences: async () => ({ theme: "system", previewHidden: false, spellcheckDisabled: false }),
    setEditorPreferences: async (prefs) => { record(`setEditorPreferences:${JSON.stringify(prefs)}`); },
    getSidebarCollapsed: async () => { record("getSidebarCollapsed"); return false; },
    setSidebarCollapsed: async (c) => { record(`setSidebarCollapsed:${c}`); },
    getSidebarWidth: async () => { record("getSidebarWidth"); return 0; },
    setSidebarWidth: async (px) => { record(`setSidebarWidth:${px}`); },
    getLastDrawerTab: async () => { record("getLastDrawerTab"); return ""; },
    setLastDrawerTab: async (id) => { record(`setLastDrawerTab:${id}`); },
    saveWindowBoundsIfNormal: async () => { record("saveWindowBoundsIfNormal"); },
    getDrawerDock: async () => { record("getDrawerDock"); return "bottom"; },
    setDrawerDock: async (d) => { record(`setDrawerDock:${d}`); },
    getDrawerRightWidth: async () => { record("getDrawerRightWidth"); return 0; },
    setDrawerRightWidth: async (px) => { record(`setDrawerRightWidth:${px}`); },
    showAboutDialog: async () => { record("showAboutDialog"); },
    flushPreferences: async () => { record("flushPreferences"); },
    getCommands: async () => { record("getCommands"); return []; },
    setDisabledCommands: async (ids) => { record(`setDisabledCommands:${ids.join(",")}`); },
    getCurrentLibrary: async () => { record("getCurrentLibrary"); return openReturn; },
    getLastActiveFile: async () => { record("getLastActiveFile"); return ""; },
    setActiveLibraryFile: async (p) => { record(`setActiveLibraryFile:${p}`); },
    getSpellAllowlist: async () => { record("getSpellAllowlist"); return []; },
    addSpellAllowlistWord: async () => { record("addSpellAllowlistWord"); return true; },
    removeSpellAllowlistWord: async () => { record("removeSpellAllowlistWord"); return true; },
  };
  return env;
}

const globalStubs = {
  Editor: {
    name: "Editor",
    props: ["env", "content", "style", "documentKey", "readOnly", "getSpellAllowlist", "addSpellAllowlistWord", "removeSpellAllowlistWord"],
    template: '<div class="editor-stub" />',
  },
  ToastManager: {
    name: "ToastManager",
    template: '<div />',
    methods: { addToast() {} },
  },
  ToolbarButton: {
    name: "ToolbarButton",
    // Declaring emits prevents Vue from double-dispatching the click as
    // both a native fallthrough listener and a component emission (which
    // would cause the parent's @click to fire twice per DOM click).
    emits: ["click"],
    inheritAttrs: false,
    template: '<button @click="$emit(\'click\', $event)"><slot /></button>',
  },
};

// Helper: mount the component and run through the initial onMounted so the
// workspace is populated. Returns the wrapper and the env.
async function mountApp(opts: { files?: LibraryFile[] } = {}) {
  stubDom();
  const env = createEnv({ files: opts.files ?? [{ path: "play.ds", name: "play.ds", updatedAt: "" }] });
  env._setContent("play.ds", "old");
  const wrapper = mount(AppDesktop, {
    props: { env },
    global: { stubs: globalStubs },
  });
  await flushPromises();
  return { wrapper, env };
}

// Simulate the user typing by updating the content prop on the Editor stub.
// This propagates via v-model:content to activeContent on AppDesktop, which
// kicks its debounced save.
async function typeInto(wrapper: ReturnType<typeof mount>, content: string) {
  const editor = wrapper.findComponent({ name: "Editor" });
  editor.vm.$emit("update:content", content);
  await flushPromises();
}

describe("AppDesktop flush ordering", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("handleOpenFolder flushes the pending save BEFORE opening the new folder", async () => {
    const { wrapper, env } = await mountApp();

    await typeInto(wrapper, "edit-A");

    // The debounce has not fired yet.
    expect(env._calls.some((c) => c.startsWith("writeLibraryFile:play.ds"))).toBe(false);

    // Slow the write so we can prove it completes before changeLibraryLocation.
    env._setWriteDelay(20);

    env._setFiles([{ path: "other.ds", name: "other.ds", updatedAt: "" }]);
    env._setOpenReturn("/projects/beta");

    const handleOpen = (wrapper.vm as any).handleOpenFolder ?? null;
    // handleOpenFolder is declared in <script setup>; not exposed by default.
    // Instead, invoke by finding the "New Project" / "Open Folder" button —
    // but the welcome screen has them. For robust test, click the sidebar's
    // Open Folder button.
    const openBtn = wrapper.findAll("button").find((b) => b.attributes("title") === "Change library location" || b.text().includes("Open Folder"));
    expect(openBtn).toBeDefined();
    openBtn!.trigger("click");

    // Allow the pending save timer + write delay to complete.
    await vi.advanceTimersByTimeAsync(1500);
    await flushPromises();

    const writeDone = env._calls.indexOf("writeLibraryFile:play.ds:done");
    const openIdx = env._calls.indexOf("changeLibraryLocation");
    expect(writeDone).toBeGreaterThanOrEqual(0);
    expect(openIdx).toBeGreaterThan(writeDone);
  });

  it("file.saveVersion dispatched via command bus flushes before snapshot", async () => {
    const { wrapper, env } = await mountApp();

    await typeInto(wrapper, "edit-snapshot");
    env._setWriteDelay(20);

    // Dispatch the command the way the menu would. `dispatchCommand`
    // reads the dispatcher registered by AppDesktop in onMounted.
    // Under fake timers, the handler's internal flushSave → 20ms write
    // delay won't progress without advanceTimersByTimeAsync, so fire
    // and advance concurrently.
    const dispatchPromise = dispatchCommand("file.saveVersion");
    await vi.advanceTimersByTimeAsync(1500);
    await dispatchPromise;
    await flushPromises();

    const writeDone = env._calls.indexOf("writeLibraryFile:play.ds:done");
    const snapIdx = env._calls.findIndex((c) => c.startsWith("snapshotFile:play.ds"));
    expect(writeDone, `calls: ${env._calls.join("\n  ")}`).toBeGreaterThanOrEqual(0);
    expect(snapIdx, `calls: ${env._calls.join("\n  ")}`).toBeGreaterThan(writeDone);
  });

  it("switching files flushes the pending save before reading the next file", async () => {
    const { wrapper, env } = await mountApp({
      files: [
        { path: "play.ds", name: "play.ds", updatedAt: "" },
        { path: "other.ds", name: "other.ds", updatedAt: "" },
      ],
    });

    await typeInto(wrapper, "edit-switch");
    env._setWriteDelay(20);

    // Click the sidebar entry for "other.ds".
    const otherBtn = wrapper.findAll("button").find((b) => b.text().includes("other.ds"));
    expect(otherBtn).toBeDefined();
    otherBtn!.trigger("click");

    await vi.advanceTimersByTimeAsync(1500);
    await flushPromises();

    const writeDone = env._calls.indexOf("writeLibraryFile:play.ds:done");
    const readOther = env._calls.indexOf("readLibraryFile:other.ds");
    expect(writeDone).toBeGreaterThanOrEqual(0);
    expect(readOther).toBeGreaterThan(writeDone);
  });
});
