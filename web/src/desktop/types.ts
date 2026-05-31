import type { EditorEnv, ExportPdfOptions } from "../core/types";

export interface LibraryFile {
  path: string;
  name: string;
  updatedAt: string;
}

export interface LibraryNode {
  path: string;
  name: string;
  kind: 'folder' | 'file' | 'deleted-file';
  children?: LibraryNode[];
  updatedAt?: string;
}

export type DirtyKind = 'untracked' | 'modified' | 'deleted';

export interface DirtyPath {
  path: string;
  kind: DirtyKind;
}

export interface LibraryDirty {
  plays: DirtyPath[];
  sidecars: DirtyPath[];
  other: DirtyPath[];
  count: number;
}

export interface Revision {
  hash: string;
  path: string;
  message: string;
  author: string;
  timestamp: string;
}

export interface FileGitStatus {
  dirty: boolean;
  headAt: string;
  hasHead: boolean;
  untracked: boolean;
  missing: boolean;
}

export interface ExternalFileResult {
  content: string;
  insideLibrary: boolean;
  relativePath: string;
}

export interface LibraryEnv {
  changeLibraryLocation(): Promise<string>;
  revealLibraryInExplorer(): Promise<void>;
  openExternalFileDialog(): Promise<string>;
  readExternalFile(absPath: string): Promise<ExternalFileResult>;
  addExternalFileToLibrary(absSrc: string, destRelDir: string): Promise<string>;
  getLibraryTree(): Promise<LibraryNode[]>;
  createLibraryFolder(relPath: string): Promise<void>;
  moveLibraryEntry(srcRel: string, dstRel: string): Promise<string>;
  renameLibraryEntry(srcRel: string, newName: string): Promise<string>;
  readLibraryFile(path: string): Promise<string>;
  writeLibraryFile(path: string, content: string): Promise<void>;
  createLibraryFile(name: string, content: string): Promise<string>;
  snapshotFile(path: string, message: string): Promise<void>;
  getRevisions(path: string, limit?: number): Promise<Revision[]>;
  readFileAtRevision(path: string, hash: string): Promise<string>;
  getHiddenRevisions(): Promise<string[]>;
  hideRevision(hash: string): Promise<void>;
  unhideRevision(hash: string): Promise<void>;
  getFileGitStatus(path: string): Promise<FileGitStatus>;
  getLibraryDirty(): Promise<LibraryDirty>;
  commitPaths(paths: string[], message: string): Promise<void>;
  discardPaths(paths: string[]): Promise<void>;
  deleteLibraryFile(path: string): Promise<void>;
  restoreLibraryFile(path: string): Promise<void>;
  getCurrentLibrary(): Promise<string>;
  getLastActiveFile(): Promise<string>;
  setActiveLibraryFile(rel: string): Promise<void>;
  getSpellAllowlist(): Promise<string[]>;
  addSpellAllowlistWord(word: string): Promise<boolean>;
  removeSpellAllowlistWord(word: string): Promise<boolean>;
  getSidebarCollapsed(): Promise<boolean>;
  setSidebarCollapsed(collapsed: boolean): Promise<void>;
  getSidebarWidth(): Promise<number>;
  setSidebarWidth(px: number): Promise<void>;
  getLastDrawerTab(): Promise<string>;
  setLastDrawerTab(id: string): Promise<void>;
  saveWindowBoundsIfNormal(): Promise<void>;
  getDrawerDock(): Promise<'bottom' | 'right'>;
  setDrawerDock(dock: 'bottom' | 'right'): Promise<void>;
  getDrawerRightWidth(): Promise<number>;
  setDrawerRightWidth(px: number): Promise<void>;
  getExportPreferences(): Promise<ExportPdfOptions>;
  setExportPreferences(opts: ExportPdfOptions): Promise<void>;
  showAboutDialog(): Promise<void>;
  quit(): Promise<void>;
  flushPreferences(): Promise<void>;
  getCommands(): Promise<CommandMeta[]>;
  setDisabledCommands(ids: string[]): Promise<void>;
}

export interface CommandMeta {
  id: string;
  label: string;
  category: string;
  accelerator?: string;
  paletteHidden?: boolean;
}

export interface DesktopCapabilities extends EditorEnv, LibraryEnv {}
