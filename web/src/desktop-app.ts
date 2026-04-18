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
} from "./core/types";
import type { DesktopCapabilities, ProjectFile, Revision } from "./desktop/types";
import { invokeRegisteredFlushSave } from "./desktop/flush-save";

// @ts-ignore — generated at build time by Wails
import * as App from "./wailsjs/go/desktop/App";
// @ts-ignore — generated at build time by Wails
import { EventsOn, EventsEmit } from "./wailsjs/runtime/runtime";

declare const __APP_VERSION__: string;

// Event names MUST match internal/desktop/app.go.
const EVT_BEFORE_CLOSE = "downstage:before-close";
const EVT_FLUSH_COMPLETE = "downstage:flush-complete";

// Tie the Wails lifecycle event to the active flushSave registered by
// AppDesktop.vue. Registering happens in `desktop/flush-save.ts` — this
// file only lives on the desktop build graph, so tests that mount the
// component against an in-memory env don't need the Wails runtime loaded.
EventsOn(EVT_BEFORE_CLOSE, async () => {
  try {
    await invokeRegisteredFlushSave();
  } catch (e) {
    console.error("before-close flush failed:", e);
  } finally {
    EventsEmit(EVT_FLUSH_COMPLETE);
  }
});

class WailsBridge implements DesktopCapabilities {
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

  async openProjectFolder(): Promise<string> {
    return App.OpenProjectFolder();
  }

  async getProjectFiles(): Promise<ProjectFile[]> {
    return App.GetProjectFiles();
  }

  async readProjectFile(path: string): Promise<string> {
    return App.ReadProjectFile(path);
  }

  async writeProjectFile(path: string, content: string): Promise<void> {
    await App.WriteProjectFile(path, content);
  }

  async createProjectFile(name: string, content: string): Promise<string> {
    return await App.CreateProjectFile(name, content);
  }

  async snapshotFile(path: string, message: string): Promise<void> {
    await App.SnapshotFile(path, message);
  }

  async getRevisions(path: string, limit?: number): Promise<Revision[]> {
    return await App.GetRevisions(path, limit ?? 0);
  }

  async getCurrentProject(): Promise<string> {
    return App.GetCurrentProject();
  }

  async getLastActiveFile(): Promise<string> {
    return App.GetLastActiveFile();
  }

  async setActiveProjectFile(rel: string): Promise<void> {
    await App.SetActiveProjectFile(rel);
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
