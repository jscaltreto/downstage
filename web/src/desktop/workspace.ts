import { reactive } from "vue";
import type { DesktopCapabilities, ProjectFile, Revision } from "./types";

export interface WorkspaceState {
  projectPath: string | null;
  projectFiles: ProjectFile[];
  activeFile: string | null;
  revisions: Revision[];
  sidebarCollapsed: boolean;
  spellAllowlist: string[];
  isLoadingFile: boolean;
  // Revision-view mode: when a user clicks an older snapshot, these fields
  // carry the read-only preview. `activeFile` still points at the live file;
  // the banner + read-only editor read from viewingRevisionContent. Null
  // here means the editor is showing the live working copy.
  viewingRevisionHash: string | null;
  viewingRevisionContent: string | null;
  viewingRevisionMeta: Revision | null;
}

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

  constructor(private env: DesktopCapabilities) {
    this.state = reactive<WorkspaceState>({
      projectPath: null,
      projectFiles: [],
      activeFile: null,
      revisions: [],
      sidebarCollapsed:
        typeof localStorage !== "undefined" &&
        localStorage.getItem("downstage-sidebar-collapsed") === "true",
      spellAllowlist: [],
      isLoadingFile: false,
      viewingRevisionHash: null,
      viewingRevisionContent: null,
      viewingRevisionMeta: null,
    });
  }

  async init() {
    this.state.projectPath = await this.env.getCurrentProject();
    if (this.state.projectPath) {
      this.state.projectFiles = await this.env.getProjectFiles();
      this.state.spellAllowlist = await this.env.getSpellAllowlist();
    }
  }

  async openFolder(): Promise<string | null> {
    const path = await this.env.openProjectFolder();
    if (!path) return null;
    this.state.projectPath = path;
    this.state.activeFile = null;
    this.state.revisions = [];
    this.clearRevisionView();
    this.state.projectFiles = await this.env.getProjectFiles();
    // Allowlist is project-scoped — reload after a project switch.
    this.state.spellAllowlist = await this.env.getSpellAllowlist();
    return path;
  }

  async selectFile(path: string): Promise<string> {
    this.state.activeFile = path;
    this.clearRevisionView();
    this.state.isLoadingFile = true;
    try {
      const content = await this.env.readProjectFile(path);
      // Persist the active-file pointer explicitly here, once per file
      // switch, instead of letting readProjectFile do it on every read.
      await this.env.setActiveProjectFile(path);
      await this.loadRevisions();
      return content;
    } finally {
      this.state.isLoadingFile = false;
    }
  }

  async saveFile(content: string) {
    if (!this.state.activeFile) return;
    await this.env.writeProjectFile(this.state.activeFile, content);
  }

  async createFile(name: string, content: string): Promise<string> {
    const path = await this.env.createProjectFile(name, content);
    this.state.projectFiles = await this.env.getProjectFiles();
    return path;
  }

  async snapshotFile(message: string) {
    if (!this.state.activeFile) return;
    await this.env.snapshotFile(this.state.activeFile, message);
    await this.loadRevisions();
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
    await this.env.writeProjectFile(file, liveContent);

    // 2. Snapshot the pre-restore state. If nothing to commit, HEAD is
    //    already a faithful backup — swallow the sentinel and continue.
    try {
      await this.env.snapshotFile(file, "Auto-save before restore");
    } catch (e) {
      if (!isNothingToSnapshotError(e)) throw e;
    }

    // 3. Read the revision content and write it into the working copy.
    const revisionContent = await this.env.readFileAtRevision(file, hash);
    await this.env.writeProjectFile(file, revisionContent);

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
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(
        "downstage-sidebar-collapsed",
        String(this.state.sidebarCollapsed),
      );
    }
  }
}
