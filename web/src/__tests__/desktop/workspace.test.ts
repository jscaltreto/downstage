import { describe, expect, it, vi } from "vitest";
import { Workspace } from "../../desktop/workspace";
import type { DesktopCapabilities, LibraryFile, Revision } from "../../desktop/types";

function stubLocalStorage() {
  const store = new Map<string, string>();
  Object.defineProperty(globalThis, "localStorage", {
    value: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => { store.set(k, v); },
    },
    configurable: true,
  });
}

interface StubEnv extends DesktopCapabilities {
  _calls: string[];
  _files: LibraryFile[];
  _contents: Record<string, string>;
  _revisions: Revision[];
  _openReturn: string;
}

function createEnv(initial?: Partial<StubEnv>): StubEnv {
  const state = {
    _calls: [],
    _files: initial?._files ?? [],
    _contents: initial?._contents ?? {},
    _revisions: initial?._revisions ?? [],
    _openReturn: initial?._openReturn ?? "/projects/alpha",
  } as unknown as StubEnv;

  const record = <T>(name: string, fn: () => T | Promise<T>) => {
    state._calls.push(name);
    return fn();
  };

  // EditorEnv pieces — mostly no-ops in these tests.
  Object.assign(state, {
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

    // LibraryEnv with call recording.
    changeLibraryLocation: () => record("changeLibraryLocation", async () => state._openReturn),
    revealLibraryInExplorer: () => record("revealLibraryInExplorer", async () => {}),
    openExternalFileDialog: () => record("openExternalFileDialog", async () => ""),
    readExternalFile: (absPath: string) => record(`readExternalFile:${absPath}`, async () => ({
      content: (state as any)._externalContent?.[absPath] ?? "",
      insideLibrary: !!(state as any)._externalInsideLibrary?.[absPath],
      relativePath: (state as any)._externalRelativePath?.[absPath] ?? "",
    })),
    addExternalFileToLibrary: (absSrc: string, destRelDir: string) => record(
      `addExternalFileToLibrary:${absSrc}:${destRelDir}`,
      async () => {
        const name = absSrc.split(/[\\/]/).pop() ?? "file.ds";
        return destRelDir ? `${destRelDir}/${name}` : name;
      },
    ),
    getLibraryTree: () => record("getLibraryTree", async () => state._files.map((f) => ({
      path: f.path, name: f.name, kind: "file" as const, updatedAt: f.updatedAt,
    }))),
    createLibraryFolder: (p: string) => record(`createLibraryFolder:${p}`, async () => {}),
    moveLibraryEntry: (src: string, dst: string) => record(`moveLibraryEntry:${src}:${dst}`, async () => dst),
    renameLibraryEntry: (src: string, name: string) => record(`renameLibraryEntry:${src}:${name}`, async () => {
      const parent = src.includes("/") ? src.slice(0, src.lastIndexOf("/")) : "";
      return parent ? `${parent}/${name}` : name;
    }),
    readLibraryFile: (p: string) => record(`readLibraryFile:${p}`, async () => state._contents[p] ?? ""),
    writeLibraryFile: (p: string, c: string) => record(`writeLibraryFile:${p}`, async () => {
      state._contents[p] = c;
    }),
    createLibraryFile: (name: string, content: string) => record(`createLibraryFile:${name}`, async () => {
      const path = `${name}.ds`;
      state._files = [...state._files, { path, name: path, updatedAt: "" }];
      state._contents[path] = content;
      return path;
    }),
    snapshotFile: (p: string, m: string) => record(`snapshotFile:${p}:${m}`, async () => {}),
    getRevisions: (p: string, _limit?: number) => record(`getRevisions:${p}`, async () => state._revisions),
    readFileAtRevision: (p: string, h: string) => record(`readFileAtRevision:${p}:${h}`, async () => {
      const key = `${p}@${h}`;
      return state._contents[key] ?? "";
    }),
    getFileGitStatus: (p: string) => record(`getFileGitStatus:${p}`, async () => (
      (state as any)._gitStatus?.[p] ?? {
        dirty: false, headAt: "2024-01-01T00:00:00Z", hasHead: true, untracked: false, missing: false,
      }
    )),
    getEditorPreferences: () => record("getEditorPreferences", async () => ({
      theme: "system" as const,
      previewHidden: false,
      spellcheckDisabled: false,
    })),
    setEditorPreferences: (prefs: any) => record(`setEditorPreferences:${JSON.stringify(prefs)}`, async () => {}),
    getSidebarCollapsed: () => record("getSidebarCollapsed", async () => (state as any)._sidebarCollapsed ?? false),
    setSidebarCollapsed: (c: boolean) => record(`setSidebarCollapsed:${c}`, async () => {
      (state as any)._sidebarCollapsed = c;
    }),
    getSidebarWidth: () => record("getSidebarWidth", async () => (state as any)._sidebarWidth ?? 0),
    setSidebarWidth: (px: number) => record(`setSidebarWidth:${px}`, async () => {
      (state as any)._sidebarWidth = px;
    }),
    getLastDrawerTab: () => record("getLastDrawerTab", async () => (state as any)._lastDrawerTab ?? ""),
    setLastDrawerTab: (id: string) => record(`setLastDrawerTab:${id}`, async () => {
      (state as any)._lastDrawerTab = id;
    }),
    saveWindowBoundsIfNormal: () => record("saveWindowBoundsIfNormal", async () => {}),
    getDrawerDock: () => record("getDrawerDock", async () => (state as any)._drawerDock ?? "bottom"),
    setDrawerDock: (dock: 'bottom' | 'right') => record(`setDrawerDock:${dock}`, async () => {
      (state as any)._drawerDock = dock;
    }),
    getDrawerRightWidth: () => record("getDrawerRightWidth", async () => (state as any)._drawerRightWidth ?? 0),
    setDrawerRightWidth: (px: number) => record(`setDrawerRightWidth:${px}`, async () => {
      (state as any)._drawerRightWidth = px;
    }),
    showAboutDialog: () => record("showAboutDialog", async () => {}),
    getExportPreferences: () => record("getExportPreferences", async () => ({
      pageSize: "letter" as const,
      style: "standard" as const,
      layout: "single" as const,
      bookletGutter: "0.125in",
    })),
    setExportPreferences: (opts: any) => record(`setExportPreferences:${JSON.stringify(opts)}`, async () => {}),
    quit: () => record("quit", async () => {}),
    flushPreferences: () => record("flushPreferences", async () => {}),
    getCommands: () => record("getCommands", async () => []),
    setDisabledCommands: (ids: string[]) => record(`setDisabledCommands:${ids.join(",")}`, async () => {}),
    getCurrentLibrary: () => record("getCurrentLibrary", async () => state._openReturn),
    getLastActiveFile: () => record("getLastActiveFile", async () => ""),
    setActiveLibraryFile: (p: string) => record(`setActiveLibraryFile:${p}`, async () => {}),
    getSpellAllowlist: () => record("getSpellAllowlist", async () => []),
    addSpellAllowlistWord: () => record("addSpellAllowlistWord", async () => true),
    removeSpellAllowlistWord: () => record("removeSpellAllowlistWord", async () => true),
  });

  return state;
}

describe("Workspace", () => {
  it("changeLibraryLocation cancel (empty path) is a no-op on state", async () => {
    stubLocalStorage();
    const env = createEnv({ _openReturn: "" });
    const ws = new Workspace(env);

    const result = await ws.changeLibraryLocation();

    expect(result).toBeNull();
    expect(ws.state.libraryPath).toBeNull();
    expect(ws.state.activeFile).toBeNull();
  });

  it("changeLibraryLocation clears activeFile/revisions on successful switch", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
    });
    const ws = new Workspace(env);
    ws.state.activeFile = "old.ds";
    ws.state.revisions = [{ hash: "abc", path: "play.ds", message: "m", author: "a", timestamp: "" }];

    await ws.changeLibraryLocation();

    expect(ws.state.libraryPath).toBe("/projects/alpha");
    expect(ws.state.activeFile).toBeNull();
    expect(ws.state.revisions).toEqual([]);
    expect(ws.libraryFiles.value.map((f: { path: string }) => f.path)).toEqual(["play.ds"]);
  });

  it("selectFile sets activeFile, reads content, and loads revisions", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "hello" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "m", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);

    const content = await ws.selectFile("play.ds");

    expect(content).toBe("hello");
    expect(ws.state.activeFile).toBe("play.ds");
    expect(ws.state.revisions.length).toBe(1);
    // Ordering matters — activeFile must be set before readLibraryFile so
    // debounced saves captured in-flight see the right target file.
    const readIndex = env._calls.indexOf("readLibraryFile:play.ds");
    const revisionsIndex = env._calls.indexOf("getRevisions:play.ds");
    expect(readIndex).toBeGreaterThan(-1);
    expect(revisionsIndex).toBeGreaterThan(readIndex);
  });

  it("saveFile is a no-op when no file is active", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);

    await ws.saveFile("ignored");

    expect(env._calls.some((c) => c.startsWith("writeLibraryFile:"))).toBe(false);
  });

  it("createFile refreshes library files", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);

    const path = await ws.createFile("Act One", "body");

    expect(path).toBe("Act One.ds");
    expect(ws.libraryFiles.value.map((f: { path: string }) => f.path)).toContain("Act One.ds");
  });

  it("viewRevision loads content into preview state without touching the live buffer", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "live", "play.ds@abc": "older" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "initial draft", author: "a", timestamp: "2026-04-17T00:00:00Z" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");

    await ws.viewRevision("abc");

    expect(ws.state.viewingRevisionHash).toBe("abc");
    expect(ws.state.viewingRevisionContent).toBe("older");
    expect(ws.state.viewingRevisionMeta?.message).toBe("initial draft");
    // selectFile stays put — the active file (and its revisions list) is
    // still the same file; the banner just overlays the view.
    expect(ws.state.activeFile).toBe("play.ds");
    // No write happens on view.
    expect(env._calls.some((c) => c.startsWith("writeLibraryFile:"))).toBe(false);
  });

  it("clearRevisionView resets the preview state", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "live", "play.ds@abc": "older" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "m", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");

    ws.clearRevisionView();

    expect(ws.state.viewingRevisionHash).toBeNull();
    expect(ws.state.viewingRevisionContent).toBeNull();
    expect(ws.state.viewingRevisionMeta).toBeNull();
    expect(ws.state.revisionViewMode).toBe("preview");
  });

  it("toggleRevisionCompare flips the view mode when a revision is selected", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "live", "play.ds@abc": "older" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "m", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");

    expect(ws.state.revisionViewMode).toBe("preview");
    ws.toggleRevisionCompare();
    expect(ws.state.revisionViewMode).toBe("compare");
    ws.toggleRevisionCompare();
    expect(ws.state.revisionViewMode).toBe("preview");
  });

  it("toggleRevisionCompare is a no-op when no revision is selected", async () => {
    stubLocalStorage();
    const env = createEnv({});
    const ws = new Workspace(env);

    ws.toggleRevisionCompare();

    // No revision in view → mode stays at the initial 'preview' value;
    // we don't enter a mode that has nothing to render.
    expect(ws.state.revisionViewMode).toBe("preview");
  });

  it("viewRevision preserves revisionViewMode when called from compare mode", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: {
        "play.ds": "live",
        "play.ds@abc": "older-a",
        "play.ds@def": "older-b",
      },
      _revisions: [
        { hash: "abc", path: "play.ds", message: "first", author: "a", timestamp: "" },
        { hash: "def", path: "play.ds", message: "second", author: "a", timestamp: "" },
      ],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");
    ws.toggleRevisionCompare();
    expect(ws.state.revisionViewMode).toBe("compare");

    // Clicking another revision in the sidebar while compare is open
    // must update the historical pane content but keep us in compare —
    // otherwise the user gets bounced back to single-pane preview
    // every time they explore.
    await ws.viewRevision("def");

    expect(ws.state.viewingRevisionHash).toBe("def");
    expect(ws.state.viewingRevisionContent).toBe("older-b");
    expect(ws.state.revisionViewMode).toBe("compare");
  });

  it("selectFile resets revisionViewMode via clearRevisionView", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [
        { path: "play.ds", name: "play.ds", updatedAt: "" },
        { path: "other.ds", name: "other.ds", updatedAt: "" },
      ],
      _contents: { "play.ds": "live", "play.ds@abc": "older", "other.ds": "other" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "m", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");
    ws.toggleRevisionCompare();
    expect(ws.state.revisionViewMode).toBe("compare");

    await ws.selectFile("other.ds");

    // File switch routes through clearRevisionView, which also resets
    // the mode. Locks in the "every revision-exit transition resets the
    // mode" rule the design depends on.
    expect(ws.state.viewingRevisionHash).toBeNull();
    expect(ws.state.revisionViewMode).toBe("preview");
  });

  it("restoreRevision snapshots the live state, overwrites with the revision, then snapshots the restore", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "originally-saved", "play.ds@abc": "older-content" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "old draft", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");

    const result = await ws.restoreRevision("abc", "live-unsaved-edits");

    expect(result).toBe("older-content");
    expect(ws.state.viewingRevisionHash).toBeNull();
    // The backup write must land before the backup snapshot.
    const backupWrite = env._calls.indexOf("writeLibraryFile:play.ds");
    const backupSnap = env._calls.findIndex((c) => c === "snapshotFile:play.ds:Auto-save before restore");
    expect(backupWrite).toBeGreaterThanOrEqual(0);
    expect(backupSnap).toBeGreaterThan(backupWrite);
    // The revision content is then read (the restore-time read, not the
    // earlier view-time one) and written over the working copy.
    const readRev = env._calls.lastIndexOf("readFileAtRevision:play.ds:abc");
    expect(readRev).toBeGreaterThan(backupSnap);
    // The restore commit is the final state-mutating step.
    const restoreSnap = env._calls.findIndex((c) => c.startsWith("snapshotFile:play.ds:Restore version"));
    expect(restoreSnap).toBeGreaterThan(readRev);
  });

  it("loads sidebarCollapsed from env and ignores legacy localStorage key", async () => {
    stubLocalStorage();
    // Plant the legacy key — Workspace must NOT read it after the refactor.
    localStorage.setItem("downstage-sidebar-collapsed", "true");
    const env = createEnv();
    (env as any)._sidebarCollapsed = false;

    const ws = new Workspace(env);
    await ws.init();

    // env value wins; legacy localStorage key is ignored.
    expect(ws.state.sidebarCollapsed).toBe(false);
    expect(env._calls).toContain("getSidebarCollapsed");
  });

  it("toggleSidebar persists via env, not localStorage", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);
    await ws.init();

    ws.toggleSidebar();

    expect(ws.state.sidebarCollapsed).toBe(true);
    expect(env._calls).toContain("setSidebarCollapsed:true");
    // Legacy key should not be written.
    expect(localStorage.getItem("downstage-sidebar-collapsed")).toBeNull();
  });

  it("hydration guard: pre-init toggleSidebar does not persist", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);

    // Pre-init mutation — watcher would otherwise fire and overwrite the
    // stored value with the placeholder default.
    ws.toggleSidebar();

    expect(env._calls.some((c) => c.startsWith("setSidebarCollapsed"))).toBe(false);
  });

  it("selectFile refreshes gitStatus from the env and caches it", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
      _contents: { "play.ds": "hello" },
    });
    (env as any)._gitStatus = {
      "play.ds": { dirty: false, headAt: "2024-06-01T00:00:00Z", hasHead: true, untracked: false, missing: false },
    };
    const ws = new Workspace(env);

    await ws.selectFile("play.ds");

    expect(ws.state.gitStatus).toEqual({
      dirty: false, headAt: "2024-06-01T00:00:00Z", hasHead: true, untracked: false, missing: false,
    });
    expect(env._calls).toContain("getFileGitStatus:play.ds");
  });

  it("saveFile flips gitStatus.dirty synchronously via the local fast path", async () => {
    vi.useFakeTimers();
    try {
      stubLocalStorage();
      const env = createEnv({
        _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
        _contents: { "play.ds": "hello" },
      });
      const ws = new Workspace(env);
      await ws.selectFile("play.ds");
      expect(ws.state.gitStatus?.dirty).toBe(false);

      await ws.saveFile("hello world");

      // Fast path — dirty is true without waiting for backend.
      expect(ws.state.gitStatus?.dirty).toBe(true);
      // Backend is not polled on every save — only after the debounced
      // reconcile timer fires (advance past it below).
      const refreshesBeforeDebounce = env._calls.filter((c) => c === "getFileGitStatus:play.ds").length;
      // selectFile itself called refresh once; no additional calls yet.
      expect(refreshesBeforeDebounce).toBe(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("scheduleDirtyReconcile refetches gitStatus after the debounce", async () => {
    vi.useFakeTimers();
    try {
      stubLocalStorage();
      const env = createEnv({
        _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
        _contents: { "play.ds": "hello" },
      });
      const ws = new Workspace(env);
      await ws.selectFile("play.ds");

      // Backend will now report clean (simulating an undo-to-HEAD).
      (env as any)._gitStatus = {
        "play.ds": { dirty: false, headAt: "2024-06-01T00:00:00Z", hasHead: true, untracked: false, missing: false },
      };

      await ws.saveFile("hello world");
      expect(ws.state.gitStatus?.dirty).toBe(true);

      await vi.advanceTimersByTimeAsync(2100);
      // Wait a microtask so the awaited refreshGitStatus resolves.
      await Promise.resolve();

      expect(ws.state.gitStatus?.dirty).toBe(false);
      const refreshCount = env._calls.filter((c) => c === "getFileGitStatus:play.ds").length;
      // Once from selectFile, once from the debounced reconcile.
      expect(refreshCount).toBe(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it("snapshotFile refreshes gitStatus (dirty should clear)", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
      _contents: { "play.ds": "hello" },
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.saveFile("dirty content");
    expect(ws.state.gitStatus?.dirty).toBe(true);

    await ws.snapshotFile("commit msg");

    expect(ws.state.gitStatus?.dirty).toBe(false);
  });

  it("changeLibraryLocation clears gitStatus", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
      _contents: { "play.ds": "hello" },
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    expect(ws.state.gitStatus).not.toBeNull();

    await ws.changeLibraryLocation();

    expect(ws.state.gitStatus).toBeNull();
  });

  it("setSidebarWidth clamps to range and debounces persistence", async () => {
    vi.useFakeTimers();
    try {
      stubLocalStorage();
      const env = createEnv();
      const ws = new Workspace(env);
      await ws.init();

      // Below min clamps up.
      ws.setSidebarWidth(50);
      expect(ws.state.sidebarWidth).toBe(180);
      // Above max clamps down.
      ws.setSidebarWidth(10000);
      expect(ws.state.sidebarWidth).toBe(600);

      // Debounce: no env write during the window.
      const persistCallsBefore = env._calls.filter((c) => c.startsWith("setSidebarWidth:")).length;
      expect(persistCallsBefore).toBe(0);

      await vi.advanceTimersByTimeAsync(400);
      const persistCalls = env._calls.filter((c) => c.startsWith("setSidebarWidth:"));
      expect(persistCalls.length).toBe(1);
      expect(persistCalls[0]).toBe("setSidebarWidth:600");
    } finally {
      vi.useRealTimers();
    }
  });

  it("setSidebarWidth gated on hydrated — pre-init sets state but doesn't persist", async () => {
    vi.useFakeTimers();
    try {
      stubLocalStorage();
      const env = createEnv();
      const ws = new Workspace(env);

      // No init call — hydrated is still false.
      ws.setSidebarWidth(300);
      expect(ws.state.sidebarWidth).toBe(300);

      await vi.advanceTimersByTimeAsync(400);
      expect(env._calls.filter((c) => c.startsWith("setSidebarWidth:"))).toHaveLength(0);
    } finally {
      vi.useRealTimers();
    }
  });

  it("init loads sidebarWidth and lastDrawerTab; defaults apply when stored value is zero/empty", async () => {
    stubLocalStorage();
    const env = createEnv();
    (env as any)._sidebarWidth = 350;
    (env as any)._lastDrawerTab = "stats";

    const ws = new Workspace(env);
    await ws.init();
    expect(ws.state.sidebarWidth).toBe(350);
    expect(ws.state.lastDrawerTab).toBe("stats");

    const ws2 = new Workspace(createEnv());
    await ws2.init();
    expect(ws2.state.sidebarWidth).toBe(256);
    expect(ws2.state.lastDrawerTab).toBe("");
  });

  it("init loads drawerDock + drawerRightWidth; zero width falls back to default", async () => {
    stubLocalStorage();
    const env = createEnv();
    (env as any)._drawerDock = "right";
    (env as any)._drawerRightWidth = 420;

    const ws = new Workspace(env);
    await ws.init();
    expect(ws.state.drawerDock).toBe("right");
    expect(ws.state.drawerRightWidth).toBe(420);

    const ws2 = new Workspace(createEnv());
    await ws2.init();
    expect(ws2.state.drawerDock).toBe("bottom");
    expect(ws2.state.drawerRightWidth).toBe(360);
  });

  it("setDrawerDock mirrors to state and persists via env", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);
    await ws.init();
    ws.setDrawerDock("right");
    expect(ws.state.drawerDock).toBe("right");
    expect(env._calls).toContain("setDrawerDock:right");
  });

  it("setDrawerRightWidth clamps and debounces persistence", async () => {
    vi.useFakeTimers();
    try {
      stubLocalStorage();
      const env = createEnv();
      const ws = new Workspace(env);
      await ws.init();

      ws.setDrawerRightWidth(100); // below min → 240
      expect(ws.state.drawerRightWidth).toBe(240);
      ws.setDrawerRightWidth(9000); // above max → 800
      expect(ws.state.drawerRightWidth).toBe(800);

      expect(env._calls.filter((c) => c.startsWith("setDrawerRightWidth:"))).toHaveLength(0);
      await vi.advanceTimersByTimeAsync(400);
      const calls = env._calls.filter((c) => c.startsWith("setDrawerRightWidth:"));
      expect(calls).toEqual(["setDrawerRightWidth:800"]);
    } finally {
      vi.useRealTimers();
    }
  });

  it("setLastDrawerTab persists and reflects on state", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);
    await ws.init();

    ws.setLastDrawerTab("outline");
    expect(ws.state.lastDrawerTab).toBe("outline");
    expect(env._calls).toContain("setLastDrawerTab:outline");
  });

  it("restoreRevision swallows 'nothing to snapshot' from the pre-restore backup", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "clean", "play.ds@abc": "older-content" },
      _revisions: [{ hash: "abc", path: "play.ds", message: "m", author: "a", timestamp: "" }],
    });
    // Backup snapshot fires "nothing to snapshot". Restore snapshot succeeds.
    let snapCount = 0;
    env.snapshotFile = async (p: string, m: string) => {
      env._calls.push(`snapshotFile:${p}:${m}`);
      snapCount++;
      if (snapCount === 1) {
        throw new Error("downstage: nothing-to-snapshot");
      }
    };
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");

    const result = await ws.restoreRevision("abc", "clean");
    expect(result).toBe("older-content");
    expect(snapCount).toBe(2);
  });

  it("openExternalFile enters external view for paths outside the library", async () => {
    stubLocalStorage();
    const env = createEnv();
    (env as any)._externalContent = { "/outside/foo.ds": "outside content" };
    const ws = new Workspace(env);
    ws.state.activeFile = "previous.ds";

    const content = await ws.openExternalFile("/outside/foo.ds");

    expect(content).toBe("outside content");
    expect(ws.state.externalFile).toEqual({ absPath: "/outside/foo.ds", content: "outside content" });
    // Entering external view clears active-file state so revisions/git
    // status don't point at a ghost.
    expect(ws.state.activeFile).toBeNull();
  });

  it("openExternalFile routes inside-library paths to selectFile", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "hello" },
    });
    (env as any)._externalInsideLibrary = { "/lib/play.ds": true };
    (env as any)._externalRelativePath = { "/lib/play.ds": "play.ds" };
    const ws = new Workspace(env);

    const content = await ws.openExternalFile("/lib/play.ds");

    // Took the in-library path: activeFile is set, externalFile is null.
    expect(content).toBe("hello");
    expect(ws.state.activeFile).toBe("play.ds");
    expect(ws.state.externalFile).toBeNull();
  });

  it("addExternalFileToLibrary copies the file and transitions to editable mode", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "foo.ds": "hello" },
    });
    (env as any)._externalContent = { "/outside/foo.ds": "outside content" };
    const ws = new Workspace(env);

    await ws.openExternalFile("/outside/foo.ds");
    expect(ws.state.externalFile).not.toBeNull();

    const newPath = await ws.addExternalFileToLibrary("");
    expect(newPath).toBe("foo.ds");
    expect(ws.state.externalFile).toBeNull();
    expect(ws.state.activeFile).toBe("foo.ds");
  });

  it("closeExternalFile clears external state", async () => {
    stubLocalStorage();
    const env = createEnv();
    (env as any)._externalContent = { "/outside/foo.ds": "hi" };
    const ws = new Workspace(env);

    await ws.openExternalFile("/outside/foo.ds");
    expect(ws.state.externalFile).not.toBeNull();

    ws.closeExternalFile();
    expect(ws.state.externalFile).toBeNull();
  });

  it("createFolder auto-expands ancestors of the new folder", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);
    await ws.init();

    await ws.createFolder("a/b/c");

    expect(ws.state.expandedFolders.has("a")).toBe(true);
    expect(ws.state.expandedFolders.has("a/b")).toBe(true);
    expect(ws.state.expandedFolders.has("a/b/c")).toBe(true);
  });

  it("moveEntry updates activeFile when the moved path IS the active file", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
      _contents: { "play.ds": "hello" },
    });
    const ws = new Workspace(env);
    await ws.init();
    await ws.selectFile("play.ds");

    await ws.moveEntry("play.ds", "archive/play.ds");

    expect(ws.state.activeFile).toBe("archive/play.ds");
  });

  it("moveEntry prefix-substitutes activeFile when a parent folder moves", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "act-one/scene-one.ds", name: "scene-one.ds", updatedAt: "" }],
      _contents: { "act-one/scene-one.ds": "hello" },
    });
    const ws = new Workspace(env);
    await ws.init();
    await ws.selectFile("act-one/scene-one.ds");

    await ws.moveEntry("act-one", "archive/act-one");

    expect(ws.state.activeFile).toBe("archive/act-one/scene-one.ds");
  });

  it("moveEntry does not touch activeFile on unrelated moves", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [
        { path: "alpha.ds", name: "alpha.ds", updatedAt: "" },
        { path: "beta.ds", name: "beta.ds", updatedAt: "" },
      ],
      _contents: { "alpha.ds": "hello" },
    });
    const ws = new Workspace(env);
    await ws.init();
    await ws.selectFile("alpha.ds");

    await ws.moveEntry("beta.ds", "archive/beta.ds");

    expect(ws.state.activeFile).toBe("alpha.ds");
  });

  it("renameEntry updates activeFile when the renamed path IS the active file", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "sub/old.ds", name: "old.ds", updatedAt: "" }],
      _contents: { "sub/old.ds": "hello" },
    });
    const ws = new Workspace(env);
    await ws.init();
    await ws.selectFile("sub/old.ds");

    await ws.renameEntry("sub/old.ds", "new.ds");

    expect(ws.state.activeFile).toBe("sub/new.ds");
  });

  it("toggleFolderExpansion flips expansion state", () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);

    expect(ws.state.expandedFolders.has("a")).toBe(false);
    ws.toggleFolderExpansion("a");
    expect(ws.state.expandedFolders.has("a")).toBe(true);
    ws.toggleFolderExpansion("a");
    expect(ws.state.expandedFolders.has("a")).toBe(false);
  });
});
