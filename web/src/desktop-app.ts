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

// @ts-ignore — generated at build time by Wails
import * as App from "./wailsjs/go/desktop/App";

declare const __APP_VERSION__: string;

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
    return App.RenderHTML(source, style || "standard");
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

  async getRevisions(path: string): Promise<Revision[]> {
    return await App.GetRevisions(path);
  }

  async getCurrentProject(): Promise<string> {
    return App.GetCurrentProject();
  }

  async getLastActiveFile(): Promise<string> {
    return App.GetLastActiveFile();
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

async function start() {
  const scripts = ["/wails/ipc.js", "/wails/runtime.js"];
  for (const src of scripts) {
    await new Promise<void>((resolve, reject) => {
      const script = document.createElement("script");
      script.src = src;
      script.onload = () => resolve();
      script.onerror = reject;
      document.head.appendChild(script);
    });
  }

  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    if ((window as any).go?.desktop?.App) break;
    await new Promise(r => setTimeout(r, 50));
  }
  if (!(window as any).go?.desktop?.App) {
    document.body.innerHTML =
      '<div style="color:white;padding:20px;">Failed to initialize Wails runtime</div>';
    return;
  }

  const env = new WailsBridge();
  const app = createApp(AppDesktop, { env });
  app.mount("#app");
}

start();
