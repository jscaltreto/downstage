import { reactive } from "vue";
import type { DesktopCapabilities, FileGitStatus, LibraryFile, Revision } from "./types";

export type DrawerDock = 'bottom' | 'right';

export interface WorkspaceState {
  libraryPath: string | null;
  libraryFiles: LibraryFile[];
  activeFile: string | null;
  revisions: Revision[];
  sidebarCollapsed: boolean;
  sidebarWidth: number;
  lastDrawerTab: string;
  drawerDock: DrawerDock;
  drawerRightWidth: number;
  spellAllowlist: string[];
  isLoadingFile: boolean;
  // Per-file git status for the status bar — null until an active file
  // resolves. Dirty flips via a fast local path on save; HeadAt and the
  // other fields come from the backend on file-switch / snapshot /
  // restore and after a reconcile debounce.
  gitStatus: FileGitStatus | null;
  // Revision-view mode: when a user clicks an older snapshot, these fields
  // carry the read-only preview. `activeFile` still points at the live file;
  // the banner + read-only editor read from viewingRevisionContent. Null
  // here means the editor is showing the live working copy.
  viewingRevisionHash: string | null;
  viewingRevisionContent: string | null;
  viewingRevisionMeta: Revision | null;
  // External-file mode: populated when the user opened a .ds file from
  // outside the library via File → Open. The editor is rendered read-
  // only with an "Add to Library" banner until the user either imports
  // the file or closes the view. `activeFile` is null while this is
  // set, so the revisions panel / git status bar short-circuit cleanly.
  externalFile: { absPath: string; content: string } | null;
}

// How long after the last save the workspace waits before re-querying
// backend git status to confirm the dirty flag. Short enough that an
// undo-to-HEAD clears the dot within a second or two; long enough that
// a burst of keystrokes doesn't hammer go-git.
const dirtyReconcileMs = 2000;

// Sidebar width clamps. 180 is narrow enough to nearly hide the file
// names; 600 is generous without crowding the editor on typical
// laptop displays.
export const minSidebarWidth = 180;
export const maxSidebarWidth = 600;
export const defaultSidebarWidth = 256;

// Drawer right-dock width clamps and default. 240 is narrow enough to
// be a hint pane; 800 is enough to host Find & Replace comfortably
// while leaving the editor usable on a 13" display.
export const minDrawerRightWidth = 240;
export const maxDrawerRightWidth = 800;
export const defaultDrawerRightWidth = 360;

// Debounce between rapid mouse-drag sidebar/drawer updates and the
// backend persistence call. Reactive state updates at frame rate; the
// persisted prefs-cache write only needs to catch up on pause.
const sidebarPersistDebounceMs = 300;
const drawerWidthPersistDebounceMs = 300;

// Prefix the Go backend uses for the "clean worktree after staging" sentinel.
// Kept in sync with internal/desktop/git.go:ErrNothingToSnapshot.
const nothingToSnapshotPrefix = "downstage: nothing-to-snapshot";

function isNothingToSnapshotError(e: unknown): boolean {
  return (
    e instanceof Error &&
    typeof e.message === "string" &&
    e.message.includes(nothingToSnapshotPrefix)
  );
}

export class Workspace {
  public state: WorkspaceState;

  // hydrated guards any persistence side effects that might otherwise fire
  // during the window between constructor and init() completion. The
  // constructor seeds state with placeholder defaults; the env read
  // populates the real values in init(). Without the guard, a naive
  // toggleSidebar fired mid-init could overwrite the real value with the
  // placeholder.
  private hydrated = false;
  // ReturnType<typeof setTimeout> covers both browser (number) and Node
  // (NodeJS.Timeout) — the unit tests run in Node without a `window`.
  private dirtyReconcileTimer: ReturnType<typeof setTimeout> | null = null;
  private sidebarPersistTimer: ReturnType<typeof setTimeout> | null = null;
  private drawerWidthPersistTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(private env: DesktopCapabilities) {
    this.state = reactive<WorkspaceState>({
      libraryPath: null,
      libraryFiles: [],
      activeFile: null,
      revisions: [],
      // Placeholder. Real values come from env pref reads in init.
      sidebarCollapsed: false,
      sidebarWidth: defaultSidebarWidth,
      lastDrawerTab: "",
      drawerDock: "bottom",
      drawerRightWidth: defaultDrawerRightWidth,
      spellAllowlist: [],
      isLoadingFile: false,
      gitStatus: null,
      viewingRevisionHash: null,
      viewingRevisionContent: null,
      viewingRevisionMeta: null,
      externalFile: null,
    });
  }

  async init() {
    this.state.libraryPath = await this.env.getCurrentLibrary();
    if (this.state.libraryPath) {
      this.state.libraryFiles = await this.env.getLibraryFiles();
      this.state.spellAllowlist = await this.env.getSpellAllowlist();
    }
    this.state.sidebarCollapsed = await this.env.getSidebarCollapsed();
    const storedWidth = await this.env.getSidebarWidth();
    this.state.sidebarWidth = storedWidth > 0 ? clampSidebarWidth(storedWidth) : defaultSidebarWidth;
    this.state.lastDrawerTab = await this.env.getLastDrawerTab();
    this.state.drawerDock = await this.env.getDrawerDock();
    const storedDrawerWidth = await this.env.getDrawerRightWidth();
    this.state.drawerRightWidth = storedDrawerWidth > 0
      ? clampDrawerRightWidth(storedDrawerWidth)
      : defaultDrawerRightWidth;
    this.hydrated = true;
  }

  async changeLibraryLocation(): Promise<string | null> {
    const path = await this.env.changeLibraryLocation();
    if (!path) return null;
    this.state.libraryPath = path;
    this.state.activeFile = null;
    this.state.revisions = [];
    this.state.gitStatus = null;
    this.state.externalFile = null;
    this.cancelDirtyReconcile();
    this.clearRevisionView();
    this.state.libraryFiles = await this.env.getLibraryFiles();
    // Allowlist is library-scoped — reload after a library switch.
    this.state.spellAllowlist = await this.env.getSpellAllowlist();
    return path;
  }

  async selectFile(path: string): Promise<string> {
    // Selecting a library file always exits external-file view — the
    // editor now represents a real, editable file.
    this.state.externalFile = null;
    this.state.activeFile = path;
    this.clearRevisionView();
    this.cancelDirtyReconcile();
    this.state.isLoadingFile = true;
    try {
      const content = await this.env.readLibraryFile(path);
      // Persist the active-file pointer explicitly here, once per file
      // switch, instead of letting readLibraryFile do it on every read.
      await this.env.setActiveLibraryFile(path);
      await this.loadRevisions();
      await this.refreshGitStatus();
      return content;
    } finally {
      this.state.isLoadingFile = false;
    }
  }

  // openExternalFile implements the File → Open flow. The state
  // transitions are sequenced so intermediate states are always
  // coherent — see internal/desktop/AGENTS.md for the contract and
  // the plan-review discussion.
  async openExternalFile(absPath: string): Promise<string> {
    const result = await this.env.readExternalFile(absPath);

    if (result.insideLibrary) {
      // Picked a file that lives inside the library. Clear external
      // state first so selectFile cannot see lingering external mode,
      // then route through the normal file-open path.
      this.state.externalFile = null;
      return this.selectFile(result.relativePath);
    }

    // Genuinely external — enter read-only mode.
    this.clearRevisionView();
    this.cancelDirtyReconcile();
    this.state.activeFile = null;
    this.state.gitStatus = null;
    this.state.revisions = [];
    this.state.externalFile = { absPath, content: result.content };
    return result.content;
  }

  async addExternalFileToLibrary(destRelDir: string): Promise<string> {
    const external = this.state.externalFile;
    if (!external) {
      throw new Error("no external file to add");
    }
    const newRel = await this.env.addExternalFileToLibrary(external.absPath, destRelDir);
    this.state.libraryFiles = await this.env.getLibraryFiles();
    // Clear external state before selectFile so the editor transitions
    // cleanly from read-only external view to editable library file.
    this.state.externalFile = null;
    await this.selectFile(newRel);
    return newRel;
  }

  closeExternalFile(): void {
    this.state.externalFile = null;
  }

  async saveFile(content: string) {
    if (!this.state.activeFile) return;
    await this.env.writeLibraryFile(this.state.activeFile, content);
    // Fast path: flip dirty locally so the status bar dot appears with
    // zero IPC latency. Correctness path: schedule a debounced
    // refreshGitStatus so an undo-to-HEAD eventually clears the dot,
    // which a monotonic markDirtyLocally alone could never do.
    this.markDirtyLocally();
    this.scheduleDirtyReconcile();
  }

  async createFile(name: string, content: string): Promise<string> {
    const path = await this.env.createLibraryFile(name, content);
    this.state.libraryFiles = await this.env.getLibraryFiles();
    return path;
  }

  async snapshotFile(message: string) {
    if (!this.state.activeFile) return;
    await this.env.snapshotFile(this.state.activeFile, message);
    await this.loadRevisions();
    this.cancelDirtyReconcile();
    await this.refreshGitStatus();
  }

  // refreshGitStatus pulls the backend's view of the active file into
  // state.gitStatus. Called on file switch, snapshot, restore, and the
  // debounced reconcile cycle after a save. A no-op when there is no
  // active file.
  async refreshGitStatus() {
    const file = this.state.activeFile;
    if (!file) {
      this.state.gitStatus = null;
      return;
    }
    try {
      this.state.gitStatus = await this.env.getFileGitStatus(file);
    } catch {
      // Surface "unknown" rather than sticking with a stale cached
      // value — the UI prefers showing nothing to showing wrong info.
      this.state.gitStatus = null;
    }
  }

  // markDirtyLocally is the fast path used right after a successful
  // writeLibraryFile. It only ever flips Dirty=true and is safe to call
  // with no gitStatus cached (it materializes a minimal record in that
  // case). HeadAt / HasHead stay untouched so the "Last snapshot"
  // display doesn't regress.
  markDirtyLocally() {
    if (!this.state.activeFile) return;
    const prev = this.state.gitStatus;
    this.state.gitStatus = {
      dirty: true,
      headAt: prev?.headAt ?? "",
      hasHead: prev?.hasHead ?? false,
      untracked: prev?.untracked ?? !prev?.hasHead,
      missing: false,
    };
  }

  private scheduleDirtyReconcile() {
    if (this.dirtyReconcileTimer !== null) {
      clearTimeout(this.dirtyReconcileTimer);
    }
    this.dirtyReconcileTimer = setTimeout(() => {
      this.dirtyReconcileTimer = null;
      void this.refreshGitStatus();
    }, dirtyReconcileMs);
  }

  private cancelDirtyReconcile() {
    if (this.dirtyReconcileTimer !== null) {
      clearTimeout(this.dirtyReconcileTimer);
      this.dirtyReconcileTimer = null;
    }
  }

  async loadRevisions() {
    if (!this.state.activeFile) {
      this.state.revisions = [];
      return;
    }
    try {
      this.state.revisions = await this.env.getRevisions(
        this.state.activeFile,
      );
    } catch {
      this.state.revisions = [];
    }
  }

  // viewRevision loads a historical snapshot into read-only view mode.
  // The editor shows viewingRevisionContent (with a banner) until the user
  // either restores or exits the view. `activeFile` stays put so the
  // revisions list and file context remain correct.
  async viewRevision(hash: string): Promise<void> {
    if (!this.state.activeFile) return;
    const meta =
      this.state.revisions.find((r) => r.hash === hash) ?? null;
    const content = await this.env.readFileAtRevision(
      this.state.activeFile,
      hash,
    );
    this.state.viewingRevisionHash = hash;
    this.state.viewingRevisionContent = content;
    this.state.viewingRevisionMeta = meta;
  }

  clearRevisionView(): void {
    this.state.viewingRevisionHash = null;
    this.state.viewingRevisionContent = null;
    this.state.viewingRevisionMeta = null;
  }

  // restoreRevision takes the live editor content, first attempts a
  // pre-restore snapshot so the user can reverse the action, then overwrites
  // the working copy with the revision's content and snapshots that too.
  // Returns the new editor content so the caller can swap its reactive
  // buffer. Leaves revision-view mode cleared on success.
  //
  // Snapshot failures on the pre-restore step are tolerated when they are
  // the "nothing to snapshot" sentinel — nothing to back up means the
  // previous snapshot already represents the current on-disk state. Any
  // other error aborts before we overwrite.
  async restoreRevision(
    hash: string,
    liveContent: string,
  ): Promise<string> {
    const file = this.state.activeFile;
    if (!file) {
      throw new Error("no active file to restore");
    }

    // 1. Persist the in-memory live content so step 2 can capture it.
    await this.env.writeLibraryFile(file, liveContent);

    // 2. Snapshot the pre-restore state. If nothing to commit, HEAD is
    //    already a faithful backup — swallow the sentinel and continue.
    try {
      await this.env.snapshotFile(file, "Auto-save before restore");
    } catch (e) {
      if (!isNothingToSnapshotError(e)) throw e;
    }

    // 3. Read the revision content and write it into the working copy.
    const revisionContent = await this.env.readFileAtRevision(file, hash);
    await this.env.writeLibraryFile(file, revisionContent);

    // 4. Commit the restore so it shows up in the revisions list. The
    //    pre-restore step guarantees we have something to diff against, but
    //    defensively tolerate the sentinel here too.
    const short = hash.substring(0, 7);
    try {
      await this.env.snapshotFile(file, `Restore version ${short}`);
    } catch (e) {
      if (!isNothingToSnapshotError(e)) throw e;
    }

    await this.loadRevisions();
    this.cancelDirtyReconcile();
    await this.refreshGitStatus();
    this.clearRevisionView();
    return revisionContent;
  }

  async addAllowlistWord(word: string): Promise<boolean> {
    const added = await this.env.addSpellAllowlistWord(word);
    if (added) {
      this.state.spellAllowlist = await this.env.getSpellAllowlist();
    }
    return added;
  }

  async removeAllowlistWord(word: string): Promise<boolean> {
    const removed = await this.env.removeSpellAllowlistWord(word);
    if (removed) {
      this.state.spellAllowlist = await this.env.getSpellAllowlist();
    }
    return removed;
  }

  toggleSidebar() {
    this.state.sidebarCollapsed = !this.state.sidebarCollapsed;
    // Persist via env. Guard on hydrated so an accidental pre-init toggle
    // doesn't clobber the real stored value with the placeholder default.
    if (!this.hydrated) return;
    void this.env.setSidebarCollapsed(this.state.sidebarCollapsed);
  }

  // setSidebarWidth updates the reactive state synchronously (so the
  // drag handle redraws at frame rate) and debounces the backend write.
  // Clamps into [minSidebarWidth, maxSidebarWidth]. Gated on hydrated
  // so a pre-init write can't clobber the real stored value with a
  // placeholder.
  setSidebarWidth(px: number) {
    const next = clampSidebarWidth(px);
    this.state.sidebarWidth = next;
    if (!this.hydrated) return;
    if (this.sidebarPersistTimer !== null) {
      clearTimeout(this.sidebarPersistTimer);
    }
    this.sidebarPersistTimer = setTimeout(() => {
      this.sidebarPersistTimer = null;
      void this.env.setSidebarWidth(next);
    }, sidebarPersistDebounceMs);
  }

  // setLastDrawerTab mirrors the tab selection into state + persists
  // through the prefs cache. No debounce — tab switches are
  // user-driven and infrequent.
  setLastDrawerTab(id: string) {
    this.state.lastDrawerTab = id;
    if (!this.hydrated) return;
    void this.env.setLastDrawerTab(id);
  }

  // setDrawerDock flips the workbench drawer between bottom (default)
  // and right (vertical side-dock). Persists through the prefs cache.
  setDrawerDock(dock: DrawerDock) {
    this.state.drawerDock = dock;
    if (!this.hydrated) return;
    void this.env.setDrawerDock(dock);
  }

  // setDrawerRightWidth updates the reactive width (so the resize
  // handle redraws at frame rate) and debounces the backend write.
  setDrawerRightWidth(px: number) {
    const next = clampDrawerRightWidth(px);
    this.state.drawerRightWidth = next;
    if (!this.hydrated) return;
    if (this.drawerWidthPersistTimer !== null) {
      clearTimeout(this.drawerWidthPersistTimer);
    }
    this.drawerWidthPersistTimer = setTimeout(() => {
      this.drawerWidthPersistTimer = null;
      void this.env.setDrawerRightWidth(next);
    }, drawerWidthPersistDebounceMs);
  }
}

function clampSidebarWidth(px: number): number {
  if (!Number.isFinite(px)) return defaultSidebarWidth;
  return Math.max(minSidebarWidth, Math.min(maxSidebarWidth, Math.round(px)));
}

function clampDrawerRightWidth(px: number): number {
  if (!Number.isFinite(px)) return defaultDrawerRightWidth;
  return Math.max(minDrawerRightWidth, Math.min(maxDrawerRightWidth, Math.round(px)));
}
