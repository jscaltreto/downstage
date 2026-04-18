import { describe, expect, it, vi } from "vitest";
import { Workspace } from "../../desktop/workspace";
import type { DesktopCapabilities, ProjectFile, Revision } from "../../desktop/types";

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
  _files: ProjectFile[];
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

    // ProjectEnv with call recording.
    openProjectFolder: () => record("openProjectFolder", async () => state._openReturn),
    getProjectFiles: () => record("getProjectFiles", async () => state._files),
    readProjectFile: (p: string) => record(`readProjectFile:${p}`, async () => state._contents[p] ?? ""),
    writeProjectFile: (p: string, c: string) => record(`writeProjectFile:${p}`, async () => {
      state._contents[p] = c;
    }),
    createProjectFile: (name: string, content: string) => record(`createProjectFile:${name}`, async () => {
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
    flushPreferences: () => record("flushPreferences", async () => {}),
    getCommands: () => record("getCommands", async () => []),
    setDisabledCommands: (ids: string[]) => record(`setDisabledCommands:${ids.join(",")}`, async () => {}),
    getCurrentProject: () => record("getCurrentProject", async () => state._openReturn),
    getLastActiveFile: () => record("getLastActiveFile", async () => ""),
    setActiveProjectFile: (p: string) => record(`setActiveProjectFile:${p}`, async () => {}),
    getSpellAllowlist: () => record("getSpellAllowlist", async () => []),
    addSpellAllowlistWord: () => record("addSpellAllowlistWord", async () => true),
    removeSpellAllowlistWord: () => record("removeSpellAllowlistWord", async () => true),
  });

  return state;
}

describe("Workspace", () => {
  it("openFolder cancel (empty path) is a no-op on state", async () => {
    stubLocalStorage();
    const env = createEnv({ _openReturn: "" });
    const ws = new Workspace(env);

    const result = await ws.openFolder();

    expect(result).toBeNull();
    expect(ws.state.projectPath).toBeNull();
    expect(ws.state.activeFile).toBeNull();
  });

  it("openFolder clears activeFile/revisions on successful switch", async () => {
    stubLocalStorage();
    const env = createEnv({
      _files: [{ path: "play.ds", name: "play.ds", updatedAt: "" }],
    });
    const ws = new Workspace(env);
    ws.state.activeFile = "old.ds";
    ws.state.revisions = [{ hash: "abc", message: "m", author: "a", timestamp: "" }];

    await ws.openFolder();

    expect(ws.state.projectPath).toBe("/projects/alpha");
    expect(ws.state.activeFile).toBeNull();
    expect(ws.state.revisions).toEqual([]);
    expect(ws.state.projectFiles.map(f => f.path)).toEqual(["play.ds"]);
  });

  it("selectFile sets activeFile, reads content, and loads revisions", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "hello" },
      _revisions: [{ hash: "abc", message: "m", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);

    const content = await ws.selectFile("play.ds");

    expect(content).toBe("hello");
    expect(ws.state.activeFile).toBe("play.ds");
    expect(ws.state.revisions.length).toBe(1);
    // Ordering matters — activeFile must be set before readProjectFile so
    // debounced saves captured in-flight see the right target file.
    const readIndex = env._calls.indexOf("readProjectFile:play.ds");
    const revisionsIndex = env._calls.indexOf("getRevisions:play.ds");
    expect(readIndex).toBeGreaterThan(-1);
    expect(revisionsIndex).toBeGreaterThan(readIndex);
  });

  it("saveFile is a no-op when no file is active", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);

    await ws.saveFile("ignored");

    expect(env._calls.some((c) => c.startsWith("writeProjectFile:"))).toBe(false);
  });

  it("createFile refreshes project files", async () => {
    stubLocalStorage();
    const env = createEnv();
    const ws = new Workspace(env);

    const path = await ws.createFile("Act One", "body");

    expect(path).toBe("Act One.ds");
    expect(ws.state.projectFiles.map(f => f.path)).toContain("Act One.ds");
  });

  it("viewRevision loads content into preview state without touching the live buffer", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "live", "play.ds@abc": "older" },
      _revisions: [{ hash: "abc", message: "initial draft", author: "a", timestamp: "2026-04-17T00:00:00Z" }],
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
    expect(env._calls.some((c) => c.startsWith("writeProjectFile:"))).toBe(false);
  });

  it("clearRevisionView resets the preview state", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "live", "play.ds@abc": "older" },
      _revisions: [{ hash: "abc", message: "m", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");

    ws.clearRevisionView();

    expect(ws.state.viewingRevisionHash).toBeNull();
    expect(ws.state.viewingRevisionContent).toBeNull();
    expect(ws.state.viewingRevisionMeta).toBeNull();
  });

  it("restoreRevision snapshots the live state, overwrites with the revision, then snapshots the restore", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "originally-saved", "play.ds@abc": "older-content" },
      _revisions: [{ hash: "abc", message: "old draft", author: "a", timestamp: "" }],
    });
    const ws = new Workspace(env);
    await ws.selectFile("play.ds");
    await ws.viewRevision("abc");

    const result = await ws.restoreRevision("abc", "live-unsaved-edits");

    expect(result).toBe("older-content");
    expect(ws.state.viewingRevisionHash).toBeNull();
    // The backup write must land before the backup snapshot.
    const backupWrite = env._calls.indexOf("writeProjectFile:play.ds");
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

  it("restoreRevision swallows 'nothing to snapshot' from the pre-restore backup", async () => {
    stubLocalStorage();
    const env = createEnv({
      _contents: { "play.ds": "clean", "play.ds@abc": "older-content" },
      _revisions: [{ hash: "abc", message: "m", author: "a", timestamp: "" }],
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
});
