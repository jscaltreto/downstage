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
    readFileAtRevision: async (p, h) => {
      record(`readFileAtRevision:${p}:${h}`);
      // Look up by `${path}@${hash}` first so tests can seed per-
      // revision content; fall back to the live path so existing
      // tests that don't care about per-hash distinctness still work.
      const keyed = contents[`${p}@${h}`];
      if (keyed !== undefined) return keyed;
      return contents[p] ?? "";
    },
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
    getLibraryDirty: async () => {
      record("getLibraryDirty");
      return { plays: [], sidecars: [], other: [], count: 0 };
    },
    commitPaths: async (paths, msg) => { record(`commitPaths:${paths.join(",")}:${msg}`); },
    discardPaths: async (paths) => { record(`discardPaths:${paths.join(",")}`); },
    deleteLibraryFile: async (p) => { record(`deleteLibraryFile:${p}`); },
    restoreLibraryFile: async (p) => { record(`restoreLibraryFile:${p}`); },
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

  // M6: a programmatic file switch must NOT trigger the 1s autosave
  // watcher to re-write the just-loaded bytes back to disk. Without the
  // (file, content) sentinel, selecting "other.ds" would land its
  // contents in activeContent, the watcher would fire, and 1s later we'd
  // see a writeLibraryFile:other.ds with identical bytes — wasted I/O
  // and a spurious dirty flicker. The fix: AppDesktop.markProgrammaticLoad
  // suppresses the next tick when (file, content) matches.
  it("programmatic file switch does NOT trigger autosave write to new file", async () => {
    const { wrapper, env } = await mountApp({
      files: [
        { path: "play.ds", name: "play.ds", updatedAt: "" },
        { path: "other.ds", name: "other.ds", updatedAt: "" },
      ],
    });
    env._setContent("other.ds", "other-original-content");

    // Click "other.ds" — a pure programmatic load, no prior user edit.
    const otherRow = wrapper.find('[data-testid="library-tree-row"][data-path="other.ds"]');
    otherRow.trigger("click");

    // Let the autosave debounce (1s) fully fire, then any microtasks.
    await vi.advanceTimersByTimeAsync(2500);
    await flushPromises();

    // The only writeLibraryFile in this scenario should be the initial
    // flush of play.ds (if any). NO write to other.ds should appear —
    // we just loaded it; writing it back would be a no-op disk hit.
    const writesToOther = env._calls.filter((c) => c === "writeLibraryFile:other.ds");
    expect(writesToOther).toEqual([]);
  });

  // Regression for the M7 follow-up: when workspace.selectFile rejects
  // (e.g. the file was deleted out from under the app), the host has to
  // surface a toast. Pre-fix the rejection became an unhandled promise
  // and the UI just silently reverted to the previous file — confusing.
  it("surfaces a toast when selectFile fails for a sidebar click", async () => {
    stubDom();
    const env = createEnv({
      files: [
        { path: "play.ds", name: "play.ds", updatedAt: "" },
        { path: "dead.ds", name: "dead.ds", updatedAt: "" },
      ],
    });
    env._setContent("play.ds", "old");
    const originalRead = env.readLibraryFile;
    env.readLibraryFile = async (p: string) => {
      if (p === "dead.ds") throw new Error("ENOENT: no such file");
      return originalRead(p);
    };

    const addToast = vi.fn();
    const wrapper = mount(AppDesktop, {
      props: { env },
      global: {
        stubs: {
          ...globalStubs,
          ToastManager: {
            name: "ToastManager",
            template: '<div />',
            methods: { addToast },
          },
        },
      },
    });
    await flushPromises();
    addToast.mockClear();

    const deadRow = wrapper.find('[data-testid="library-tree-row"][data-path="dead.ds"]');
    deadRow.trigger("click");
    await vi.advanceTimersByTimeAsync(0);
    await flushPromises();

    const errorCalls = addToast.mock.calls.filter((args) => args[1] === "error");
    expect(errorCalls.length, `toast calls: ${JSON.stringify(addToast.mock.calls)}`).toBeGreaterThan(0);
    expect(errorCalls[0][0]).toMatch(/dead\.ds/);
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
    // T16: seed distinct content per revision so any test that
    // promotes compareTwo can verify the diff component actually
    // received hash-A's content as original and hash-B's content as
    // modified. The readFileAtRevision stub prefers `${path}@${hash}`
    // keys; without these the diff would silently read "live" for
    // both sides and the test wouldn't catch a wrong-hash bug.
    env._setContent("play.ds@abc1234567aaaa", "older-A-content");
    env._setContent("play.ds@def4567890bbbb", "older-B-content");
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
    // T16: the diff component's props prove that A's content went to
    // `original` and B's content went to `modified`. With distinct
    // per-hash content seeded by mountWithRevisions(), a regression
    // that read the wrong revision would flip these.
    expect(diff.props('original')).toBe('older-A-content');
    expect(diff.props('modified')).toBe('older-B-content');
    expect(diff.props('modifiedLabel')).toContain('Saved');
  });

  // M10: re-invoking "Compare to current" on A while already in
  // compareTwo (A vs B) must collapse to compareCurrent (A vs live
  // buffer). Without the stopCompareTwo call inside
  // handleCompareToCurrent, the second invocation is a no-op
  // (revisionViewMode is already 'compare') and the user is stuck.
  it("Compare to current from compareTwo drops B and lands in compareCurrent", async () => {
    const { wrapper } = await mountWithRevisions();

    // Drive into compareTwo via right-click → Compare with… → click B.
    await findRevRow(wrapper, "draft a")!.trigger('contextmenu');
    await flushPromises();
    await wrapper.findAll('button').find(b => b.text() === 'Compare with…')!.trigger('click');
    await flushPromises();
    await findRevRow(wrapper, "draft b")!.trigger('click');
    await flushPromises();
    // Sanity: we're in compareTwo.
    let diff = wrapper.findComponent({ name: "RevisionDiffView" });
    expect(diff.props('modified')).toBe('older-B-content');

    // Now invoke "Compare to current" on A.
    await findRevRow(wrapper, "draft a")!.trigger('contextmenu');
    await flushPromises();
    const compareCurrentBtn = wrapper.findAll('button').find(b => b.text() === 'Compare to current');
    await compareCurrentBtn!.trigger('click');
    await flushPromises();

    // compareTwo banner gone, single-revision banner back, diff's
    // modifiedLabel is "Current" (proving B was dropped and we're
    // diffing against the live buffer, not the stale B).
    expect(wrapper.text()).not.toContain('Comparing versions');
    expect(wrapper.text()).toContain('Viewing older version');
    diff = wrapper.findComponent({ name: "RevisionDiffView" });
    expect(diff.exists()).toBe(true);
    expect(diff.props('modifiedLabel')).toBe('Current');
  });

  // M12: file.exportDs must be enabled in external-file mode. The
  // setDisabledCommands env call records the disabled set; we open
  // an external file and assert file.exportDs is not listed.
  it("file.exportDs is enabled in external-file mode", async () => {
    stubDom();
    const env = createEnv({
      files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
    });
    env._setContent("play.ds", "live");
    // Force the next readExternalFile to return content from outside
    // the library so workspace.openExternalFile sets externalFile.
    const origRead = env.readExternalFile;
    env.readExternalFile = async (p) => {
      const r = await origRead(p);
      return { content: "external-content", insideLibrary: false, relativePath: "" };
    };
    env.openExternalFileDialog = async () => "/tmp/elsewhere.ds";

    const wrapper = mount(AppDesktop, {
      props: { env },
      global: { stubs: globalStubs },
    });
    await flushPromises();

    // Drive into external-file mode via the file.open command.
    await dispatchCommand("file.open");
    await flushPromises();

    // The dispatcher's microtask flush pushed a new disabled set. The
    // most recent setDisabledCommands call shouldn't include
    // file.exportDs because external mode is a valid export source.
    const setDisabledCalls = env._calls.filter(c => c.startsWith("setDisabledCommands:"));
    expect(setDisabledCalls.length).toBeGreaterThan(0);
    const latest = setDisabledCalls[setDisabledCalls.length - 1];
    expect(latest).not.toContain("file.exportDs");

    wrapper.unmount();
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
