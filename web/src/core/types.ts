export interface ParseError {
  message: string;
  line: number;
  col: number;
  endLine: number;
  endCol: number;
}

export interface WasmDiagnostic {
  message: string;
  severity: "error" | "warning" | "info" | "hint";
  line: number;
  col: number;
  endLine: number;
  endCol: number;
  code?: string;
}

export interface SavedDraft {
  id: string;
  title: string;
  content: string;
  updatedAt: string;
}

export interface EditorEnv {
  // Parsing and Diagnostics
  parse(source: string): Promise<{ errors: ParseError[] }>;
  diagnostics(source: string): Promise<{ diagnostics: WasmDiagnostic[] }>;
  upgradeV1(source: string): Promise<{ source: string; changed: boolean }>;
  semanticTokens(source: string): Promise<Uint32Array>;
  tokenTypeNames(): Promise<string[]>;

  // Rendering
  renderHTML(source: string, style?: string): Promise<string>;
  renderPDF(source: string, style?: string): Promise<Uint8Array>;

  // Persistence (Drafts)
  loadDrafts(): Promise<SavedDraft[]>;
  saveDrafts(drafts: SavedDraft[]): Promise<void>;
  loadActiveDraftId(): Promise<string | null>;
  saveActiveDraftId(id: string): Promise<void>;

  // Platform specific
  saveFile(filename: string, content: string | Uint8Array, filters?: { displayName: string; pattern: string }[]): Promise<void>;
  importLocalFile(): Promise<{ name: string; content: string } | null>;
  openURL(url: string): Promise<void>;

  // Metadata
  getAppVersion(): string;
}
