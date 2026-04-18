import type { EditorEnv } from "../core/types";

export interface ProjectFile {
  path: string;
  name: string;
  updatedAt: string;
}

export interface Revision {
  hash: string;
  message: string;
  author: string;
  timestamp: string;
}

// Mirror of internal/desktop.FileGitStatus — the status-bar's dirty-dot
// and "Last snapshot N ago" readings. The Go side is the source of
// truth; see GetFileGitStatus for the semantics of each field.
export interface FileGitStatus {
  dirty: boolean;
  headAt: string;
  hasHead: boolean;
  untracked: boolean;
  missing: boolean;
}

export interface ProjectEnv {
  openProjectFolder(): Promise<string>;
  getProjectFiles(): Promise<ProjectFile[]>;
  readProjectFile(path: string): Promise<string>;
  writeProjectFile(path: string, content: string): Promise<void>;
  createProjectFile(name: string, content: string): Promise<string>;
  snapshotFile(path: string, message: string): Promise<void>;
  // `limit <= 0` falls back to the server's default bound (currently 100)
  // to avoid unbounded payloads on long-lived projects.
  getRevisions(path: string, limit?: number): Promise<Revision[]>;
  readFileAtRevision(path: string, hash: string): Promise<string>;
  // Per-file status for the desktop status bar: dirty flag + last
  // snapshot time + missing/untracked signals. See FileGitStatus.
  getFileGitStatus(path: string): Promise<FileGitStatus>;
  getCurrentProject(): Promise<string>;
  getLastActiveFile(): Promise<string>;
  setActiveProjectFile(rel: string): Promise<void>;
  getSpellAllowlist(): Promise<string[]>;
  addSpellAllowlistWord(word: string): Promise<boolean>;
  removeSpellAllowlistWord(word: string): Promise<boolean>;
  // Desktop-only pref. Thin bool accessor on top of the same Go Config
  // Preferences blob that backs getEditorPreferences; keeping it as two
  // separate get/set methods avoids forcing the web env to model a field
  // it doesn't have.
  getSidebarCollapsed(): Promise<boolean>;
  setSidebarCollapsed(collapsed: boolean): Promise<void>;
  // Awaits completion of any in-flight preference write. Called on
  // window-close so a debounced toggle isn't lost when the user quits.
  flushPreferences(): Promise<void>;
  // Palette-facing catalog. Labels + categories + accelerators come from
  // the Go catalog so there's no duplicate UI text on the TS side.
  getCommands(): Promise<CommandMeta[]>;
  // Push the latest disabled-command set to the native menu. The host
  // computes this via the CommandDispatcher's microtask-diffed refresh;
  // this method is the wire.
  setDisabledCommands(ids: string[]): Promise<void>;
}

// Palette-facing projection of the Go catalog — no Click callbacks, no
// menu path. Mirrors internal/desktop/commands.go's CommandMeta.
export interface CommandMeta {
  id: string;
  label: string;
  category: string;
  accelerator?: string;
  paletteHidden?: boolean;
}

export interface DesktopCapabilities extends EditorEnv, ProjectEnv {}
