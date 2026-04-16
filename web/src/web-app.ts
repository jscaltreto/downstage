import "./main.css";
import { createApp } from "vue";
import AppWeb from "./AppWeb.vue";
import { initWasm, upgradeV1 as upgradeV1Wasm } from "./wasm";
import type {
  EditorEnv,
  SavedDraft,
  ParseError,
  WasmDiagnostic,
  LSPCompletionList,
  LSPCodeActionsResult,
  SpellcheckContext,
  DocumentSymbolsResult,
  ManuscriptStats,
} from "./core/types";

declare const __APP_VERSION__: string;

const draftsStorageKey = "downstage-editor-drafts";
const activeDraftStorageKey = "downstage-editor-active-draft";

function isValidDraft(obj: any): obj is SavedDraft {
  return (
    obj &&
    typeof obj === "object" &&
    typeof obj.id === "string" &&
    typeof obj.title === "string" &&
    typeof obj.content === "string" &&
    typeof obj.updatedAt === "string" &&
    (!("spellAllowlist" in obj) || Array.isArray(obj.spellAllowlist))
  );
}

function normalizeDraft(draft: SavedDraft): SavedDraft {
  return {
    ...draft,
    spellAllowlist: Array.isArray(draft.spellAllowlist) ? draft.spellAllowlist : [],
  };
}

class WebEnv implements EditorEnv {
  async init() {
    await initWasm();
  }

  async parse(source: string): Promise<{ errors: ParseError[] }> {
    return window.downstage.parse(source);
  }

  async diagnostics(source: string): Promise<{ diagnostics: WasmDiagnostic[] }> {
    return window.downstage.diagnostics(source);
  }

  async spellcheckContext(source: string): Promise<SpellcheckContext> {
    return window.downstage.spellcheckContext(source);
  }

  async upgradeV1(source: string): Promise<{ source: string; changed: boolean }> {
    return upgradeV1Wasm(source);
  }

  async completion(source: string, line: number, col: number): Promise<LSPCompletionList> {
    return window.downstage.completion(source, line, col);
  }

  async codeActions(
    source: string,
    line: number,
    col: number,
    codes?: string[],
  ): Promise<LSPCodeActionsResult> {
    return window.downstage.codeActions(source, line, col, codes);
  }

  async documentSymbols(source: string): Promise<DocumentSymbolsResult> {
    return window.downstage.documentSymbols(source);
  }

  async semanticTokens(source: string): Promise<Uint32Array> {
    return window.downstage.semanticTokens(source);
  }

  async tokenTypeNames(): Promise<string[]> {
    return Array.from(window.downstage.tokenTypeNames);
  }

  async stats(source: string): Promise<ManuscriptStats> {
    return window.downstage.stats(source);
  }

  async renderHTML(source: string, style?: string): Promise<string> {
    return window.downstage.renderHTML(source, style);
  }

  async renderPDF(source: string, style?: string): Promise<Uint8Array> {
    return window.downstage.renderPDF(source, style);
  }

  async loadDrafts(): Promise<SavedDraft[]> {
    try {
      const raw = localStorage.getItem(draftsStorageKey);
      if (!raw) return [];
      const parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) return [];
      return parsed.filter(isValidDraft).map(normalizeDraft);
    } catch (e) {
      console.warn("failed to load drafts from storage:", e);
      return [];
    }
  }

  async saveDrafts(drafts: SavedDraft[]): Promise<void> {
    try {
      localStorage.setItem(draftsStorageKey, JSON.stringify(drafts));
    } catch (e) {
      console.error("failed to save drafts to storage:", e);
    }
  }

  async loadActiveDraftId(): Promise<string | null> {
    try {
      return localStorage.getItem(activeDraftStorageKey);
    } catch {
      return null;
    }
  }

  async saveActiveDraftId(id: string): Promise<void> {
    try {
      localStorage.setItem(activeDraftStorageKey, id);
    } catch (e) {
      console.error("failed to save active draft ID to storage:", e);
    }
  }

  async saveFile(filename: string, content: string | Uint8Array, _filters?: { displayName: string; pattern: string }[]): Promise<void> {
    const blob = new Blob([content as BlobPart], { type: typeof content === "string" ? "text/plain" : "application/pdf" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  async importLocalFile(): Promise<{ name: string; content: string } | null> {
    return new Promise((resolve) => {
      const input = document.createElement("input");
      input.type = "file";
      input.accept = ".ds,text/plain";
      input.onchange = async () => {
        const file = input.files?.[0];
        if (!file) {
          resolve(null);
          return;
        }
        const content = await file.text();
        resolve({ name: file.name, content });
      };
      input.oncancel = () => resolve(null);
      input.click();
    });
  }

  async openURL(url: string): Promise<void> {
    window.open(url, "_blank");
  }

  getAppVersion(): string {
    return __APP_VERSION__;
  }
}

const env = new WebEnv();
env.init().then(() => {
    const app = createApp(AppWeb, { env });
    app.mount("#app");
}).catch(err => {
    console.error("Failed to initialize WASM:", err);
    document.body.innerHTML = `<div style="color: white; padding: 20px;">Failed to load Downstage: ${err.message}</div>`;
});
