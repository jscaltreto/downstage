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
} from "./core/types";
import type { CommandMeta, DesktopCapabilities, ExternalFileResult, FileGitStatus, LibraryFile, Revision } from "./desktop/types";
import { invokeRegisteredFlushSave } from "./desktop/flush-save";
import { createPrefsCache } from "./desktop/prefs-cache";
import { dispatchCommand } from "./desktop/dispatcher-registry";

// Structural shape of the Go Preferences struct. Intentionally not
// imported from the Wails-generated module — this file keeps a thin
// structural copy so unit tests that don't have wailsjs/ available can
// reason about the type.
interface WailsPreferences {
  theme?: string;
  previewHidden?: boolean;
  spellcheckDisabled?: boolean;
  sidebarCollapsed?: boolean;
  sidebarWidth?: number;
  lastDrawerTab?: string;
  drawerDock?: 'bottom' | 'right';
  drawerRightWidth?: number;
}

// @ts-ignore — generated at build time by Wails
import * as App from "./wailsjs/go/desktop/App";
// @ts-ignore — generated at build time by Wails
import { EventsOn, EventsEmit } from "./wailsjs/runtime/runtime";

declare const __APP_VERSION__: string;

// Event names MUST match internal/desktop/app.go.
const EVT_BEFORE_CLOSE = "downstage:before-close";
const EVT_FLUSH_COMPLETE = "downstage:flush-complete";
const EVT_COMMAND_EXECUTE = "command:execute";

// Tie the Wails lifecycle event to the active flushSave registered by
// AppDesktop.vue. Registering happens in `desktop/flush-save.ts` — this
// file only lives on the desktop build graph, so tests that mount the
// component against an in-memory env don't need the Wails runtime loaded.
// Native-menu click → TS dispatcher. AppDesktop.vue registers the live
// dispatcher via the dispatcher-registry in onMounted. Subscribing at
// module scope means we don't miss a click fired during Vue mount.
EventsOn(EVT_COMMAND_EXECUTE, (id: unknown) => {
  if (typeof id === "string") {
    void dispatchCommand(id);
  }
});

EventsOn(EVT_BEFORE_CLOSE, async () => {
  try {
    // Documents first — losing an unsaved edit is worse than losing a
    // sidebar toggle, so prioritize the file flush if one is stalled.
    await invokeRegisteredFlushSave();
    // Then preferences — the cache's chain may still have unresolved
    // writes from a late toggle; await them before releasing the close.
    await env.flushPreferences();
  } catch (e) {
    console.error("before-close flush failed:", e);
  } finally {
    EventsEmit(EVT_FLUSH_COMPLETE);
  }
});

class WailsBridge implements DesktopCapabilities {
  // All pref reads and writes go through this cache — it maintains the
  // authoritative in-memory snapshot of the Go Preferences struct and
  // serializes writes back through App.SetPreferences so concurrent
  // updates from Store (theme/preview/spellcheck) and Workspace (sidebar)
  // can't race through independent read-modify-write cycles.
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

  async renderPDF(source: string, style?: string): Promise<Uint8Array> {
    const base64 = await App.RenderPDF(source, style || "standard");
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

  async getLibraryFiles(): Promise<LibraryFile[]> {
    return App.GetLibraryFiles();
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

  async getFileGitStatus(path: string): Promise<FileGitStatus> {
    const raw = await App.GetFileGitStatus(path);
    // Defensive reshape so missing/typos in the generated Wails types
    // can't leak undefined fields into the UI.
    return {
      dirty: !!raw?.dirty,
      headAt: raw?.headAt ? String(raw.headAt) : "",
      hasHead: !!raw?.hasHead,
      untracked: !!raw?.untracked,
      missing: !!raw?.missing,
    };
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

  // Preferences — both env projections delegate to the shared prefs cache
  // so interleaved writes can't lose each other's fields. Reads apply
  // the theme default on the frontend too (the Go getter normalizes,
  // but the cache surfaces whatever is on disk and the web env does the
  // same defensive normalization).
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

  async showAboutDialog(): Promise<void> {
    await App.ShowAboutDialog();
  }

  async flushPreferences(): Promise<void> {
    await this.prefs.flush();
  }

  async getCommands(): Promise<CommandMeta[]> {
    const raw = await App.GetCommands();
    // The Go binding returns `desktop.CommandMeta[]`; reshape defensively
    // so TS callers see a consistent structural type even if the
    // generated types drift.
    return (raw ?? []).map((c: any) => ({
      id: String(c?.id ?? ""),
      label: String(c?.label ?? ""),
      category: String(c?.category ?? ""),
      accelerator: c?.accelerator ? String(c.accelerator) : undefined,
      paletteHidden: !!c?.paletteHidden,
    }));
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
