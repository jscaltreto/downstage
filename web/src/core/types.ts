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

// Mirrors go.lsp.dev/protocol SymbolKind values used by the outline.
export const SymbolKind = {
  File: 1,
  Namespace: 3,
  Class: 5,
  Function: 12,
  Struct: 23,
} as const;

export interface DocumentSymbol {
  name: string;
  kind: number;
  range: LSPRange;
  selectionRange: LSPRange;
  children?: DocumentSymbol[];
}

export interface DocumentSymbolsResult {
  symbols: DocumentSymbol[];
}

export interface CharacterStats {
  name: string;
  aliases?: string[];
  lines: number;
  dialogueWords: number;
}

export interface RuntimeEstimate {
  preset: string;
  wordsPerMinute: number;
  pauseFactor: number;
  dialogueWords: number;
  minutes: number;
}

export interface ManuscriptStats {
  acts: number;
  scenes: number;
  songs: number;
  totalWords: number;
  dialogueWords: number;
  lines: number;
  stageDirections: number;
  stageDirectionWords: number;
  characters: CharacterStats[];
  runtime: RuntimeEstimate;
}

export interface SpellcheckRange {
  start: LSPPosition;
  end: LSPPosition;
}

export interface SpellcheckContext {
  allowWords: string[];
  ignoredRanges: SpellcheckRange[];
}

export interface EditorDiagnostic {
  from: number;
  to: number;
  line: number;
  col: number;
  severity: "error" | "warning" | "info" | "hint";
  message: string;
  code?: string;
}

export type PdfPageSize = "letter" | "a4";
export type PdfExportStyle = "standard" | "condensed";

export interface ExportPdfOptions {
  pageSize: PdfPageSize;
  style: PdfExportStyle;
}

export interface SavedDraft {
  id: string;
  title: string;
  content: string;
  updatedAt: string;
  spellAllowlist: string[];
}

export interface EditorEnv {
  // Parsing and Diagnostics
  parse(source: string): Promise<{ errors: ParseError[] }>;
  diagnostics(source: string): Promise<{ diagnostics: WasmDiagnostic[] }>;
  spellcheckContext(source: string): Promise<SpellcheckContext>;
  upgradeV1(source: string): Promise<{ source: string; changed: boolean }>;
  completion(source: string, line: number, col: number): Promise<LSPCompletionList>;
  codeActions(source: string, line: number, col: number, codes?: string[]): Promise<LSPCodeActionsResult>;
  documentSymbols(source: string): Promise<DocumentSymbolsResult>;
  semanticTokens(source: string): Promise<Uint32Array>;
  tokenTypeNames(): Promise<string[]>;
  stats(source: string): Promise<ManuscriptStats>;

  // Rendering
  renderHTML(source: string, style?: string): Promise<string>;
  renderPDF(source: string, style?: string, pageSize?: PdfPageSize): Promise<Uint8Array>;

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
