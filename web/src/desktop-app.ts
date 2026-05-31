import "./main.css";
import { createApp } from "vue";
import AppDesktop from "./AppDesktop.vue";
import type {
  SavedDraft,
  ParseError,
  WasmDiagnostic,
  LSPCompletionList,
  LSPCodeActionsResult,
  SpellcheckContext,
  DocumentSymbolsResult,
  ManuscriptStats,
  EditorPreferences,
  ExportPdfOptions,
} from "./core/types";
import type { CommandMeta, DesktopCapabilities, ExternalFileResult, FileGitStatus, LibraryFile, LibraryNode, Revision } from "./desktop/types";
import { invokeRegisteredFlushSave } from "./desktop/flush-save";
import { createPrefsCache } from "./desktop/prefs-cache";
import { dispatchCommand } from "./desktop/dispatcher-registry";
interface WailsPreferences {
  theme?: string;
  previewHidden?: boolean;
  spellcheckDisabled?: boolean;
  sidebarCollapsed?: boolean;
  sidebarWidth?: number;
  lastDrawerTab?: string;
  drawerDock?: 'bottom' | 'right';
  drawerRightWidth?: number;
  exportPageSize?: string;
  exportStyle?: string;
  exportLayout?: string;
  exportBookletGutter?: string;
}

import * as App from "./wailsjs/go/desktop/App";
import { EventsOn, EventsEmit } from "./wailsjs/runtime/runtime";

declare const __APP_VERSION__: string;

const EVT_BEFORE_CLOSE = "downstage:before-close";
const EVT_FLUSH_COMPLETE = "downstage:flush-complete";
const EVT_COMMAND_EXECUTE = "command:execute";

// EventsOn returns its unsubscribe function (see wailsjs/runtime/
// runtime.d.ts). We capture both so HMR can dispose of the listeners
// before a fresh module evaluation re-registers them; otherwise dev
// reloads stack subscribers and every menu click dispatches N times.
const cancelCommandEvent = EventsOn(EVT_COMMAND_EXECUTE, (id: unknown) => {
  if (typeof id === "string") {
    void dispatchCommand(id);
  }
});

const cancelBeforeCloseEvent = EventsOn(EVT_BEFORE_CLOSE, async () => {
  try {
    await invokeRegisteredFlushSave();
    await env.flushPreferences();
  } catch (error: unknown) {
    console.error("before-close flush failed:", error);
  } finally {
    EventsEmit(EVT_FLUSH_COMPLETE);
  }
});

// Vite injects import.meta.hot only in dev; the guard makes this a
// no-op in production builds.
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    cancelCommandEvent();
    cancelBeforeCloseEvent();
  });
}

function normalizeLibraryNode(node: unknown): LibraryNode {
  const current = node as {
    kind?: unknown;
    path?: unknown;
    name?: unknown;
    children?: unknown;
    updatedAt?: unknown;
  } | null;
  const rawKind = current?.kind;
  let kind: "folder" | "file" | "deleted-file";
  if (rawKind === "folder") kind = "folder";
  else if (rawKind === "deleted-file") kind = "deleted-file";
  else kind = "file";
  return {
    path: String(current?.path ?? ""),
    name: String(current?.name ?? ""),
    kind,
    children: Array.isArray(current?.children)
      ? current.children.map(normalizeLibraryNode)
      : undefined,
    updatedAt: current?.updatedAt ? String(current.updatedAt) : undefined,
  };
}

function normalizeCommandMeta(command: unknown): CommandMeta {
  const current = command as {
    id?: unknown;
    label?: unknown;
    category?: unknown;
    accelerator?: unknown;
    paletteHidden?: unknown;
  } | null;
  return {
    id: String(current?.id ?? ""),
    label: String(current?.label ?? ""),
    category: String(current?.category ?? ""),
    accelerator: current?.accelerator ? String(current.accelerator) : undefined,
    paletteHidden: !!current?.paletteHidden,
  };
}

class WailsBridge implements DesktopCapabilities {
  private prefs = createPrefsCache<WailsPreferences>({
    load: async () => {
      const all = await App.GetPreferences();
      return (all ?? {}) as WailsPreferences;
    },
    save: async (p) => { await App.SetPreferences(p); },
  });
  async parse(source: string): Promise<{ errors: ParseError[] }> {
    const errors = await App.Parse(source);
    return { errors };
  }

  async diagnostics(source: string): Promise<{ diagnostics: WasmDiagnostic[] }> {
    const diags = await App.Diagnostics(source);
    return {
      diagnostics: diags.map((d: { severity: string } & Omit<WasmDiagnostic, "severity">) => ({
        ...d,
        severity: d.severity as WasmDiagnostic["severity"],
      })),
    };
  }

  async spellcheckContext(source: string): Promise<SpellcheckContext> {
    return await App.SpellcheckContext(source);
  }

  async upgradeV1(source: string): Promise<{ source: string; changed: boolean }> {
    return await App.UpgradeV1(source);
  }

  async completion(source: string, line: number, col: number): Promise<LSPCompletionList> {
    return await App.Completion(source, line, col);
  }

  async codeActions(
    source: string,
    line: number,
    col: number,
    codes?: string[],
  ): Promise<LSPCodeActionsResult> {
    return await App.CodeActions(source, line, col, codes || []);
  }

  async documentSymbols(source: string): Promise<DocumentSymbolsResult> {
    return await App.DocumentSymbols(source);
  }

  async semanticTokens(source: string): Promise<Uint32Array> {
    const tokens = await App.SemanticTokens(source);
    return new Uint32Array(tokens);
  }

  async tokenTypeNames(): Promise<string[]> {
    return App.TokenTypeNames();
  }

  async stats(source: string): Promise<ManuscriptStats> {
    return await App.Stats(source);
  }

  async renderHTML(source: string, style?: string): Promise<string> {
    return await App.RenderHTML(source, style || "standard");
  }

  async renderPDF(source: string, options: ExportPdfOptions): Promise<Uint8Array> {
    const base64 = await App.RenderPDF(source, {
      pageSize: options.pageSize,
      style: options.style,
      layout: options.layout,
      bookletGutter: options.layout === "booklet" ? options.bookletGutter : "",
    });
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes;
  }

  async loadDrafts(): Promise<SavedDraft[]> { return []; }
  async saveDrafts(): Promise<void> {}
  async loadActiveDraftId(): Promise<string | null> { return null; }
  async saveActiveDraftId(): Promise<void> {}

  async changeLibraryLocation(): Promise<string> {
    return App.ChangeLibraryLocation();
  }

  async revealLibraryInExplorer(): Promise<void> {
    await App.RevealLibraryInExplorer();
  }

  async openExternalFileDialog(): Promise<string> {
    return App.OpenExternalFileDialog();
  }

  async readExternalFile(absPath: string): Promise<ExternalFileResult> {
    const raw = await App.ReadExternalFile(absPath);
    return {
      content: raw?.content ?? "",
      insideLibrary: !!raw?.insideLibrary,
      relativePath: raw?.relativePath ? String(raw.relativePath) : "",
    };
  }

  async addExternalFileToLibrary(absSrc: string, destRelDir: string): Promise<string> {
    return App.AddExternalFileToLibrary(absSrc, destRelDir);
  }

  async getLibraryTree(): Promise<LibraryNode[]> {
    const raw = await App.GetLibraryTree();
    return (raw ?? []).map(normalizeLibraryNode);
  }

  async createLibraryFolder(relPath: string): Promise<void> {
    await App.CreateLibraryFolder(relPath);
  }

  async moveLibraryEntry(srcRel: string, dstRel: string): Promise<string> {
    return App.MoveLibraryEntry(srcRel, dstRel);
  }

  async renameLibraryEntry(srcRel: string, newName: string): Promise<string> {
    return App.RenameLibraryEntry(srcRel, newName);
  }

  async readLibraryFile(path: string): Promise<string> {
    return App.ReadLibraryFile(path);
  }

  async writeLibraryFile(path: string, content: string): Promise<void> {
    await App.WriteLibraryFile(path, content);
  }

  async createLibraryFile(name: string, content: string): Promise<string> {
    return await App.CreateLibraryFile(name, content);
  }

  async snapshotFile(path: string, message: string): Promise<void> {
    await App.SnapshotFile(path, message);
  }

  async getRevisions(path: string, limit?: number): Promise<Revision[]> {
    return await App.GetRevisions(path, limit ?? 0);
  }

  async readFileAtRevision(path: string, hash: string): Promise<string> {
    return await App.ReadFileAtRevision(path, hash);
  }

  async getHiddenRevisions(): Promise<string[]> {
    return await App.GetHiddenRevisions();
  }

  async hideRevision(hash: string): Promise<void> {
    await App.HideRevision(hash);
  }

  async unhideRevision(hash: string): Promise<void> {
    await App.UnhideRevision(hash);
  }

  async getFileGitStatus(path: string): Promise<FileGitStatus> {
    const raw = await App.GetFileGitStatus(path);
    return {
      dirty: !!raw?.dirty,
      headAt: raw?.headAt ? String(raw.headAt) : "",
      hasHead: !!raw?.hasHead,
      untracked: !!raw?.untracked,
      missing: !!raw?.missing,
    };
  }

  async getLibraryDirty() {
    const raw = await App.GetLibraryDirty();
    type Kind = "untracked" | "modified" | "deleted";
    const normalizePath = (entry: unknown): { path: string; kind: Kind } => {
      const e = entry as { path?: unknown; kind?: unknown } | null;
      const rawKind = e?.kind;
      const kind: Kind =
        rawKind === "untracked" || rawKind === "modified" || rawKind === "deleted"
          ? rawKind
          : "modified";
      return { path: String(e?.path ?? ""), kind };
    };
    const normList = (arr: unknown): { path: string; kind: Kind }[] =>
      Array.isArray(arr) ? arr.map(normalizePath) : [];
    return {
      plays: normList(raw?.plays),
      sidecars: normList(raw?.sidecars),
      other: normList(raw?.other),
      count: typeof raw?.count === "number" ? raw.count : 0,
    };
  }

  async commitPaths(paths: string[], message: string): Promise<void> {
    await App.CommitPaths(paths, message);
  }

  async discardPaths(paths: string[]): Promise<void> {
    await App.DiscardPaths(paths);
  }

  async deleteLibraryFile(path: string): Promise<void> {
    await App.DeleteLibraryFile(path);
  }

  async restoreLibraryFile(path: string): Promise<void> {
    await App.RestoreLibraryFile(path);
  }

  async getCurrentLibrary(): Promise<string> {
    return App.GetCurrentLibrary();
  }

  async getLastActiveFile(): Promise<string> {
    return App.GetLastActiveFile();
  }

  async setActiveLibraryFile(rel: string): Promise<void> {
    await App.SetActiveLibraryFile(rel);
  }

  async getSpellAllowlist(): Promise<string[]> {
    return App.GetSpellAllowlist();
  }

  async addSpellAllowlistWord(word: string): Promise<boolean> {
    return App.AddSpellAllowlistWord(word);
  }

  async removeSpellAllowlistWord(word: string): Promise<boolean> {
    return App.RemoveSpellAllowlistWord(word);
  }

  async getEditorPreferences(): Promise<EditorPreferences> {
    const all = await this.prefs.get();
    return {
      theme: ((all.theme as EditorPreferences["theme"]) || "system"),
      previewHidden: !!all.previewHidden,
      spellcheckDisabled: !!all.spellcheckDisabled,
    };
  }

  async setEditorPreferences(prefs: EditorPreferences): Promise<void> {
    await this.prefs.update(prefs);
  }

  async getSidebarCollapsed(): Promise<boolean> {
    const all = await this.prefs.get();
    return !!all.sidebarCollapsed;
  }

  async setSidebarCollapsed(collapsed: boolean): Promise<void> {
    await this.prefs.update({ sidebarCollapsed: collapsed });
  }

  async getSidebarWidth(): Promise<number> {
    const all = await this.prefs.get();
    return typeof all.sidebarWidth === "number" ? all.sidebarWidth : 0;
  }

  async setSidebarWidth(px: number): Promise<void> {
    await this.prefs.update({ sidebarWidth: px });
  }

  async getLastDrawerTab(): Promise<string> {
    const all = await this.prefs.get();
    return typeof all.lastDrawerTab === "string" ? all.lastDrawerTab : "";
  }

  async setLastDrawerTab(id: string): Promise<void> {
    await this.prefs.update({ lastDrawerTab: id });
  }

  async saveWindowBoundsIfNormal(): Promise<void> {
    await App.SaveWindowBoundsIfNormal();
  }

  async getDrawerDock(): Promise<'bottom' | 'right'> {
    const all = await this.prefs.get();
    return all.drawerDock === 'right' ? 'right' : 'bottom';
  }

  async setDrawerDock(dock: 'bottom' | 'right'): Promise<void> {
    await this.prefs.update({ drawerDock: dock });
  }

  async getDrawerRightWidth(): Promise<number> {
    const all = await this.prefs.get();
    return typeof all.drawerRightWidth === "number" ? all.drawerRightWidth : 0;
  }

  async setDrawerRightWidth(px: number): Promise<void> {
    await this.prefs.update({ drawerRightWidth: px });
  }

  async getExportPreferences(): Promise<ExportPdfOptions> {
    const all = await this.prefs.get();
    const pageSize = all.exportPageSize === "a4" ? "a4" : "letter";
    const style = all.exportStyle === "condensed" ? "condensed" : "standard";
    const layout =
      all.exportLayout === "2up" || all.exportLayout === "booklet"
        ? all.exportLayout
        : "single";
    const bookletGutter =
      typeof all.exportBookletGutter === "string" && all.exportBookletGutter
        ? all.exportBookletGutter
        : "0.125in";
    return { pageSize, style, layout, bookletGutter };
  }

  async setExportPreferences(opts: ExportPdfOptions): Promise<void> {
    await this.prefs.update({
      exportPageSize: opts.pageSize,
      exportStyle: opts.style,
      exportLayout: opts.layout,
      exportBookletGutter: opts.bookletGutter,
    });
  }

  async showAboutDialog(): Promise<void> {
    await App.ShowAboutDialog();
  }

  async quit(): Promise<void> {
    await App.Quit();
  }

  async flushPreferences(): Promise<void> {
    await this.prefs.flush();
  }

  async getCommands(): Promise<CommandMeta[]> {
    const raw = await App.GetCommands();
    return (raw ?? []).map(normalizeCommandMeta);
  }

  async setDisabledCommands(ids: string[]): Promise<void> {
    await App.SetDisabledCommands(ids);
  }

  async saveFile(filename: string, content: string | Uint8Array, filters?: { displayName: string; pattern: string }[]): Promise<void> {
    const isBinary = content instanceof Uint8Array;
    let payload: string;
    if (isBinary) {
      payload = btoa(Array.from(content).map(b => String.fromCharCode(b)).join(""));
    } else {
      payload = content;
    }

    const wailsFilters = filters || [
      { displayName: "All Files (*.*)", pattern: "*.*" },
    ];

    await App.SaveFile(filename, payload, isBinary, wailsFilters);
  }

  async importLocalFile(): Promise<{ name: string; content: string } | null> {
    return null;
  }

  async openURL(url: string): Promise<void> {
    await App.BrowserOpenURL(url);
  }

  getAppVersion(): string {
    return __APP_VERSION__;
  }
}

const env = new WailsBridge();
const app = createApp(AppDesktop, { env });
app.mount("#app");
