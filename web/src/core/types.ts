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
  quickFixes?: string[];
}

export interface LSPPosition {
  line: number;
  character: number;
}

export interface LSPRange {
  start: LSPPosition;
  end: LSPPosition;
}

export interface LSPTextEdit {
  range: LSPRange;
  newText: string;
}

export interface LSPCompletionItem {
  label: string;
  kind?: number;
  detail?: string;
  filterText?: string;
  sortText?: string;
  insertText?: string;
  textEdit?: LSPTextEdit;
}

export interface LSPCompletionList {
  isIncomplete: boolean;
  items: LSPCompletionItem[];
}

export interface LSPWorkspaceEdit {
  changes?: Record<string, LSPTextEdit[]>;
}

export interface LSPCodeAction {
  title: string;
  kind?: string;
  isPreferred?: boolean;
  edit?: LSPWorkspaceEdit;
}

export interface LSPCodeActionsResult {
  uri: string;
  actions: LSPCodeAction[];
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
  completion(source: string, line: number, col: number): Promise<LSPCompletionList>;
  codeActions(source: string, line: number, col: number, codes?: string[]): Promise<LSPCodeActionsResult>;
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
