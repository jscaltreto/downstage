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
  _setRevisions: (revs: Revision[]) => void;
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

function createEnv(init: { files?: LibraryFile[]; openReturn?: string; revisions?: Revision[] } = {}): RecordedEnv {
  let openReturn = init.openReturn ?? "/projects/alpha";
  let files: LibraryFile[] = init.files ?? [];
  const contents: Record<string, string> = {};
  let revisions: Revision[] = init.revisions ?? [];
  const calls: string[] = [];
  let writeDelayMs = 0;

  const record = (name: string) => calls.push(name);

  const env: RecordedEnv = {
    _calls: calls,
    _setOpenReturn: (p) => { openReturn = p; },
    _setFiles: (f) => { files = f; },
    _setContent: (p, c) => { contents[p] = c; },
    _setRevisions: (r) => { revisions = r; },
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
    revealLibraryInExplorer: async () => { record("revealLibraryInExplorer"); },
    openExternalFileDialog: async () => { record("openExternalFileDialog"); return ""; },
    readExternalFile: async (p) => {
      record(`readExternalFile:${p}`);
      return { content: "", insideLibrary: false, relativePath: "" };
    },
    addExternalFileToLibrary: async (src, dir) => {
      record(`addExternalFileToLibrary:${src}:${dir}`);
      const path = `${dir ? `${dir}/` : ""}ext.ds`;
      return path;
    },
    getLibraryTree: async () => {
      record("getLibraryTree");
      return files.map(f => ({ path: f.path, name: f.name, kind: "file" as const, updatedAt: f.updatedAt }));
    },
    createLibraryFolder: async (p) => { record(`createLibraryFolder:${p}`); },
    moveLibraryEntry: async (src, dst) => { record(`moveLibraryEntry:${src}:${dst}`); return dst; },
    renameLibraryEntry: async (src, name) => { record(`renameLibraryEntry:${src}:${name}`); return name; },
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
    getHiddenRevisions: async () => { record("getHiddenRevisions"); return []; },
    hideRevision: async (hash) => { record(`hideRevision:${hash}`); },
    unhideRevision: async (hash) => { record(`unhideRevision:${hash}`); },
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
    getExportPreferences: async () => ({
      pageSize: "letter", style: "standard", layout: "single", bookletGutter: "0.125in",
    }),
    setExportPreferences: async (opts) => {
      record(`setExportPreferences:${JSON.stringify(opts)}`);
    },
    quit: async () => { record("quit"); },
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
  RevisionDiffView: {
    name: "RevisionDiffView",
    props: ["original", "modified", "originalLabel", "modifiedLabel", "isDark", "env"],
    template: '<div class="revision-diff-stub" />',
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

  it("Save Version dialog submit flushes before snapshot", async () => {
    // Snapshot creation now goes through the named-version dialog. The
    // command bus opens the prompt; the host's submit handler runs
    // flushSave then workspace.snapshotFile. This test drives that
    // host path: dispatch file.saveVersion to open the dialog, then
    // submit it programmatically and assert the writeLibraryFile
    // round-trip lands before the snapshot.
    const { wrapper, env } = await mountApp();

    await typeInto(wrapper, "edit-snapshot");
    env._setWriteDelay(20);

    // Open the dialog (this just flips a ref — no async work to await).
    const dispatchPromise = dispatchCommand("file.saveVersion");
    await vi.advanceTimersByTimeAsync(0);
    await dispatchPromise;
    await flushPromises();

    // Find the Save Version prompt and submit it. PromptModal renders
    // a form; the test triggers a submit event on it directly so we
    // don't depend on input focus / keyboard simulation under
    // happy-dom.
    const prompts = wrapper.findAllComponents({ name: "PromptModal" });
    const saveVersionPrompt = prompts.find(p => p.props("title") === "Save Version");
    expect(saveVersionPrompt, "Save Version prompt should be open").toBeTruthy();
    saveVersionPrompt!.vm.$emit("submit", "named-version");
    await vi.advanceTimersByTimeAsync(1500);
    await flushPromises();

    const writeDone = env._calls.indexOf("writeLibraryFile:play.ds:done");
    const snapIdx = env._calls.findIndex((c) => c.startsWith("snapshotFile:play.ds:named-version"));
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

    // Click the sidebar entry for "other.ds". The tree renders rows
    // as divs tagged with data-testid/data-path.
    const otherRow = wrapper.find('[data-testid="library-tree-row"][data-path="other.ds"]');
    expect(otherRow.exists()).toBe(true);
    otherRow.trigger("click");

    await vi.advanceTimersByTimeAsync(1500);
    await flushPromises();

    const writeDone = env._calls.indexOf("writeLibraryFile:play.ds:done");
    const readOther = env._calls.indexOf("readLibraryFile:other.ds");
    expect(writeDone).toBeGreaterThanOrEqual(0);
    expect(readOther).toBeGreaterThan(writeDone);
  });
});

describe("AppDesktop revision compare", () => {
  // No fake timers — we're driving DOM interactions, not the debounced
  // save path. The flush-ordering suite above sets fake timers in its
  // own beforeEach, so this describe runs against real timers.

  async function mountWithRevision() {
    stubDom();
    const env = createEnv({
      files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
      revisions: [
        { hash: "abc1234", path: "play.ds", message: "older draft", author: "a", timestamp: "2026-04-17T00:00:00Z" },
      ],
    });
    env._setContent("play.ds", "live-content");
    const wrapper = mount(AppDesktop, {
      props: { env },
      global: { stubs: globalStubs },
    });
    await flushPromises();
    return { wrapper, env };
  }

  it("toggling Compare swaps in the diff view but leaves Editor mounted", async () => {
    const { wrapper } = await mountWithRevision();

    // Click the revision row in the Versions sidebar. The catalog above
    // uses :title="..." on the button; targeting by title is the
    // narrowest selector that doesn't require adding test-only attrs
    // to production markup.
    const revRow = wrapper.findAll('button').find(b =>
      (b.attributes('title') ?? '').startsWith('Preview this version')
    );
    expect(revRow, "revision row should be present in the Versions sidebar").toBeTruthy();
    await revRow!.trigger('click');
    await flushPromises();

    // Banner is now visible — three buttons (Compare, Restore, Return).
    // Locate the compare toggle by its visible label.
    const compareBtn = wrapper.findAll('button').find(b => b.text().includes('Compare to current'));
    expect(compareBtn, "compare toggle should be visible while a revision is in view").toBeTruthy();
    await compareBtn!.trigger('click');
    await flushPromises();

    // Diff stub renders, Editor stub is STILL mounted (hidden via v-show
    // on its wrapper). This is the load-bearing assertion: it locks in
    // the "don't unmount the editor on compare toggle" rule that
    // preserves scroll position, search drawer state, etc.
    expect(wrapper.findComponent({ name: "RevisionDiffView" }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: "Editor" }).exists()).toBe(true);

    // Toggle back via "Hide compare" — same button, label flipped.
    const hideBtn = wrapper.findAll('button').find(b => b.text().includes('Hide compare'));
    expect(hideBtn, "compare toggle should now offer 'Hide compare'").toBeTruthy();
    await hideBtn!.trigger('click');
    await flushPromises();

    expect(wrapper.findComponent({ name: "RevisionDiffView" }).exists()).toBe(false);
    expect(wrapper.findComponent({ name: "Editor" }).exists()).toBe(true);
  });
});

describe("AppDesktop revision context menu", () => {
  async function mountWithRevisions() {
    stubDom();
    const env = createEnv({
      files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
      revisions: [
        { hash: "abc1234567aaaa", path: "play.ds", message: "draft a", author: "x", timestamp: "2026-04-17T00:00:00Z" },
        { hash: "def4567890bbbb", path: "play.ds", message: "draft b", author: "x", timestamp: "2026-04-18T00:00:00Z" },
      ],
    });
    env._setContent("play.ds", "live");
    const wrapper = mount(AppDesktop, {
      props: { env },
      global: { stubs: globalStubs },
    });
    await flushPromises();
    return { wrapper, env };
  }

  // Find a revision row by its title prefix. Stable selector — doesn't
  // require us to leak test-only attrs into production markup.
  function findRevRow(wrapper: ReturnType<typeof mount>, message: string) {
    return wrapper.findAll('button').find(b =>
      b.text().includes(message),
    );
  }

  it("right-click opens the revision context menu with five items", async () => {
    const { wrapper } = await mountWithRevisions();
    const rowA = findRevRow(wrapper, "draft a");
    expect(rowA).toBeTruthy();

    await rowA!.trigger('contextmenu');
    await flushPromises();

    const labels = wrapper.findAll('button').map(b => b.text());
    expect(labels).toContain('View this version');
    expect(labels).toContain('Compare to current');
    expect(labels).toContain('Compare with…');
    expect(labels).toContain('Copy hash');
    expect(labels).toContain('Hide this version');
  });

  it("Hide this version removes the row and records the env call", async () => {
    const { wrapper, env } = await mountWithRevisions();
    const rowA = findRevRow(wrapper, "draft a");
    await rowA!.trigger('contextmenu');
    await flushPromises();

    const hideBtn = wrapper.findAll('button').find(b => b.text() === 'Hide this version');
    expect(hideBtn).toBeTruthy();
    await hideBtn!.trigger('click');
    await flushPromises();

    expect(env._calls).toContain("hideRevision:abc1234567aaaa");
    // Row disappears from the visible list (showHidden is off by
    // default), and "Show hidden (1)" toggle appears.
    expect(findRevRow(wrapper, "draft a")).toBeUndefined();
    expect(wrapper.findAll('button').some(b => b.text().includes('Show hidden (1)'))).toBe(true);
  });

  it("Compare with… enters picking mode; clicking a second row promotes to compareTwo", async () => {
    const { wrapper } = await mountWithRevisions();

    // Open context menu on A and click "Compare with…".
    await findRevRow(wrapper, "draft a")!.trigger('contextmenu');
    await flushPromises();
    const compareWithBtn = wrapper.findAll('button').find(b => b.text() === 'Compare with…');
    await compareWithBtn!.trigger('click');
    await flushPromises();

    // Picking banner appears in the Versions panel header.
    expect(wrapper.text()).toContain('Pick another version');

    // Click the second row → resolves to compareTwo. The compareTwo
    // banner ("Comparing versions") replaces the single-revision one,
    // and RevisionDiffView mounts with the second hash's content as
    // `modified`.
    await findRevRow(wrapper, "draft b")!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Comparing versions');
    const diff = wrapper.findComponent({ name: "RevisionDiffView" });
    expect(diff.exists()).toBe(true);
    // The diff's `modified` prop should be the second-revision content
    // (env stub returns play.ds live content for any readFileAtRevision
    // — what matters here is the prop wiring, not the actual content).
    expect(diff.props('modifiedLabel')).toContain('Saved');
  });

  it("Stop comparing fully exits revision view back to the current buffer", async () => {
    const { wrapper } = await mountWithRevisions();

    // Drive into compareTwo via the same path as the previous test.
    await findRevRow(wrapper, "draft a")!.trigger('contextmenu');
    await flushPromises();
    await wrapper.findAll('button').find(b => b.text() === 'Compare with…')!.trigger('click');
    await flushPromises();
    await findRevRow(wrapper, "draft b")!.trigger('click');
    await flushPromises();

    // Click "Stop comparing" → full exit. The compareTwo banner
    // disappears, the single-revision "Viewing older version" banner
    // does NOT take its place (that would be a clever-not-helpful
    // intermediate state — user wanted to stop comparing, not pivot
    // to a different comparison), and the diff component unmounts.
    const stopBtn = wrapper.findAll('button').find(b => b.text().includes('Stop comparing'));
    expect(stopBtn).toBeTruthy();
    await stopBtn!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).not.toContain('Comparing versions');
    expect(wrapper.text()).not.toContain('Viewing older version');
    expect(wrapper.findComponent({ name: "RevisionDiffView" }).exists()).toBe(false);
  });
});
