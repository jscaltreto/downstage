import type { EditorEnv } from "../core/types";

export interface LibraryFile {
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

export interface LibraryEnv {
  changeLibraryLocation(): Promise<string>;
  getLibraryFiles(): Promise<LibraryFile[]>;
  readLibraryFile(path: string): Promise<string>;
  writeLibraryFile(path: string, content: string): Promise<void>;
  createLibraryFile(name: string, content: string): Promise<string>;
  snapshotFile(path: string, message: string): Promise<void>;
  // `limit <= 0` falls back to the server's default bound (currently 100)
  // to avoid unbounded payloads on long-lived libraries.
  getRevisions(path: string, limit?: number): Promise<Revision[]>;
  readFileAtRevision(path: string, hash: string): Promise<string>;
  // Per-file status for the desktop status bar: dirty flag + last
  // snapshot time + missing/untracked signals. See FileGitStatus.
  getFileGitStatus(path: string): Promise<FileGitStatus>;
  getCurrentLibrary(): Promise<string>;
  getLastActiveFile(): Promise<string>;
  setActiveLibraryFile(rel: string): Promise<void>;
  getSpellAllowlist(): Promise<string[]>;
  addSpellAllowlistWord(word: string): Promise<boolean>;
  removeSpellAllowlistWord(word: string): Promise<boolean>;
  // Desktop-only pref. Thin bool accessor on top of the same Go Config
  // Preferences blob that backs getEditorPreferences; keeping it as two
  // separate get/set methods avoids forcing the web env to model a field
  // it doesn't have.
  getSidebarCollapsed(): Promise<boolean>;
  setSidebarCollapsed(collapsed: boolean): Promise<void>;
  // Sidebar width (px). 0 → frontend default 256. Rides the same
  // prefs-cache as the other UI prefs.
  getSidebarWidth(): Promise<number>;
  setSidebarWidth(px: number): Promise<void>;
  // Last active drawer tab ID. "" → default 'issues'. Rides the prefs
  // cache. String typed because the concrete WorkbenchTab union lives
  // in the shared editor component; this interface stays host-agnostic.
  getLastDrawerTab(): Promise<string>;
  setLastDrawerTab(id: string): Promise<void>;
  // Live-save the window's current bounds if it is currently unmaximized.
  // The Go side reads IsMaximised + GetSize + GetPosition and only
  // writes when !maximized, so a maximized resize doesn't clobber the
  // last known normal size. Called from a debounced window.resize
  // listener.
  saveWindowBoundsIfNormal(): Promise<void>;
  // Drawer dock persistence. "bottom" (default) keeps the historical
  // layout; "right" docks the workbench drawer as a vertical column
  // between editor and preview.
  getDrawerDock(): Promise<'bottom' | 'right'>;
  setDrawerDock(dock: 'bottom' | 'right'): Promise<void>;
  getDrawerRightWidth(): Promise<number>;
  setDrawerRightWidth(px: number): Promise<void>;
  // Native info dialog carrying the build's version string. One-button
  // "OK" box; callers don't need to await anything beyond "dialog was
  // shown".
  showAboutDialog(): Promise<void>;
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

export interface DesktopCapabilities extends EditorEnv, LibraryEnv {}
