import { computed, reactive, type ComputedRef } from "vue";
import type { DesktopCapabilities, FileGitStatus, LibraryFile, LibraryNode, Revision } from "./types";

export type DrawerDock = 'bottom' | 'right';

export interface WorkspaceState {
  libraryPath: string | null;
  libraryTree: LibraryNode[];
  expandedFolders: Set<string>;
  activeFile: string | null;
  revisions: Revision[];
  sidebarCollapsed: boolean;
  sidebarWidth: number;
  lastDrawerTab: string;
  drawerDock: DrawerDock;
  drawerRightWidth: number;
  spellAllowlist: string[];
  isLoadingFile: boolean;
  gitStatus: FileGitStatus | null;
  viewingRevisionHash: string | null;
  viewingRevisionContent: string | null;
  viewingRevisionMeta: Revision | null;
  revisionViewMode: 'preview' | 'compare';
  compareSecondHash: string | null;
  compareSecondContent: string | null;
  compareSecondMeta: Revision | null;
  pickingSecondForCompare: boolean;
  hiddenRevisionHashes: Set<string>;
  showHidden: boolean;
  externalFile: { absPath: string; content: string } | null;
}

const dirtyReconcileMs = 2000;

export const minSidebarWidth = 180;
export const maxSidebarWidth = 600;
export const defaultSidebarWidth = 256;

export const minDrawerRightWidth = 240;
export const maxDrawerRightWidth = 800;
export const defaultDrawerRightWidth = 360;

const sidebarPersistDebounceMs = 300;
const drawerWidthPersistDebounceMs = 300;

const nothingToSnapshotPrefix = "downstage: nothing-to-snapshot";

function isNothingToSnapshotError(e: unknown): boolean {
  const message = String((e as { message?: unknown } | null)?.message ?? e ?? "");
  return message.includes(nothingToSnapshotPrefix);
}

export class Workspace {
  public state: WorkspaceState;
  public libraryFiles: ComputedRef<LibraryFile[]>;
  public visibleRevisions!: ComputedRef<Revision[]>;

  private hydrated = false;
  private dirtyReconcileTimer: ReturnType<typeof setTimeout> | null = null;
  private sidebarPersistTimer: ReturnType<typeof setTimeout> | null = null;
  private drawerWidthPersistTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(private env: DesktopCapabilities) {
    this.state = reactive<WorkspaceState>({
      libraryPath: null,
      libraryTree: [],
      expandedFolders: new Set<string>(),
      activeFile: null,
      revisions: [],
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
      revisionViewMode: 'preview',
      compareSecondHash: null,
      compareSecondContent: null,
      compareSecondMeta: null,
      pickingSecondForCompare: false,
      hiddenRevisionHashes: new Set<string>(),
      showHidden: false,
      externalFile: null,
    });

    this.libraryFiles = computed(() => flattenLibraryTree(this.state.libraryTree));
    this.visibleRevisions = computed(() => {
      if (this.state.showHidden) return this.state.revisions;
      if (this.state.hiddenRevisionHashes.size === 0) {
        return this.state.revisions;
      }
      return this.state.revisions.filter(
        (r) => !this.state.hiddenRevisionHashes.has(r.hash),
      );
    });
  }

  async init() {
    this.state.libraryPath = await this.env.getCurrentLibrary();
    if (this.state.libraryPath) {
      this.state.libraryTree = await this.env.getLibraryTree();
      this.state.spellAllowlist = await this.env.getSpellAllowlist();
      await this.loadHiddenRevisions();
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
    this.state.showHidden = false;
    this.cancelDirtyReconcile();
    this.clearRevisionView();
    this.state.libraryTree = await this.env.getLibraryTree();
    this.state.spellAllowlist = await this.env.getSpellAllowlist();
    await this.loadHiddenRevisions();
    return path;
  }

  async selectFile(path: string): Promise<string> {
    this.state.isLoadingFile = true;
    try {
      // Throwing operations come first: a failure leaves activeFile,
      // externalFile, viewing-revision state, and the dirty-reconcile
      // timer untouched, so the UI still represents the old selection
      // and the user can recover. loadRevisions/refreshGitStatus run
      // AFTER the commit because they swallow errors internally and
      // can degrade gracefully.
      const content = await this.env.readLibraryFile(path);
      await this.env.setActiveLibraryFile(path);
      this.state.externalFile = null;
      this.state.activeFile = path;
      this.clearRevisionView();
      this.cancelDirtyReconcile();
      await this.loadRevisions();
      await this.refreshGitStatus();
      return content;
    } finally {
      this.state.isLoadingFile = false;
    }
  }

  async openExternalFile(absPath: string): Promise<string> {
    const result = await this.env.readExternalFile(absPath);

    if (result.insideLibrary) {
      this.state.externalFile = null;
      return this.selectFile(result.relativePath);
    }

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
    this.state.libraryTree = await this.env.getLibraryTree();
    this.state.externalFile = null;
    await this.selectFile(newRel);
    return newRel;
  }

  async createFolder(relPath: string): Promise<void> {
    await this.env.createLibraryFolder(relPath);
    this.state.libraryTree = await this.env.getLibraryTree();
    let prefix = "";
    for (const segment of relPath.split("/")) {
      prefix = prefix ? `${prefix}/${segment}` : segment;
      this.state.expandedFolders.add(prefix);
    }
  }

  async moveEntry(src: string, dst: string): Promise<string> {
    const newPath = await this.env.moveLibraryEntry(src, dst);
    this.state.libraryTree = await this.env.getLibraryTree();
    this.retargetActiveFile(src, newPath);
    return newPath;
  }

  async renameEntry(src: string, newName: string): Promise<string> {
    const newPath = await this.env.renameLibraryEntry(src, newName);
    this.state.libraryTree = await this.env.getLibraryTree();
    this.retargetActiveFile(src, newPath);
    return newPath;
  }

  private retargetActiveFile(src: string, dst: string): void {
    const active = this.state.activeFile;
    if (active === null) return;
    let nextActive: string | null = null;
    if (active === src) {
      nextActive = dst;
    } else if (active.startsWith(src + "/")) {
      nextActive = dst + active.slice(src.length);
    }
    if (nextActive === null) return;
    this.state.activeFile = nextActive;
    this.clearRevisionView();
    void this.env.setActiveLibraryFile(nextActive);
    void this.refreshGitStatus();
  }

  toggleFolderExpansion(relPath: string): void {
    if (this.state.expandedFolders.has(relPath)) {
      this.state.expandedFolders.delete(relPath);
    } else {
      this.state.expandedFolders.add(relPath);
    }
  }

  closeExternalFile(): void {
    this.state.externalFile = null;
  }

  async saveFile(content: string) {
    if (!this.state.activeFile) return;
    await this.env.writeLibraryFile(this.state.activeFile, content);
    this.markDirtyLocally();
    this.scheduleDirtyReconcile();
  }

  async createFile(name: string, content: string): Promise<string> {
    const path = await this.env.createLibraryFile(name, content);
    this.state.libraryTree = await this.env.getLibraryTree();
    return path;
  }

  async snapshotFile(message: string) {
    if (!this.state.activeFile) return;
    await this.env.snapshotFile(this.state.activeFile, message);
    await this.loadRevisions();
    this.cancelDirtyReconcile();
    await this.refreshGitStatus();
  }

  async refreshGitStatus() {
    const file = this.state.activeFile;
    if (!file) {
      this.state.gitStatus = null;
      return;
    }
    try {
      this.state.gitStatus = await this.env.getFileGitStatus(file);
    } catch {
      this.state.gitStatus = null;
    }
  }

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

  async viewRevision(hash: string): Promise<void> {
    if (!this.state.activeFile) return;
    const meta =
      this.state.revisions.find((r) => r.hash === hash) ?? null;
    const lookupPath = meta?.path || this.state.activeFile;
    const content = await this.env.readFileAtRevision(lookupPath, hash);
    this.state.viewingRevisionHash = hash;
    this.state.viewingRevisionContent = content;
    this.state.viewingRevisionMeta = meta;
  }

  clearRevisionView(): void {
    this.state.viewingRevisionHash = null;
    this.state.viewingRevisionContent = null;
    this.state.viewingRevisionMeta = null;
    this.state.revisionViewMode = 'preview';
    this.state.compareSecondHash = null;
    this.state.compareSecondContent = null;
    this.state.compareSecondMeta = null;
    this.state.pickingSecondForCompare = false;
  }

  toggleRevisionCompare(): void {
    if (this.state.viewingRevisionHash === null) return;
    this.state.revisionViewMode =
      this.state.revisionViewMode === 'compare' ? 'preview' : 'compare';
  }

  async startPickSecond(hash: string): Promise<void> {
    if (this.state.viewingRevisionHash !== hash) {
      await this.viewRevision(hash);
    }
    this.state.pickingSecondForCompare = true;
  }

  async resolvePickSecond(hash: string): Promise<void> {
    if (!this.state.pickingSecondForCompare) return;
    if (hash === this.state.viewingRevisionHash) {
      return;
    }
    const meta = this.state.revisions.find((r) => r.hash === hash) ?? null;
    const lookupPath = meta?.path || this.state.activeFile;
    if (!lookupPath) return;
    const content = await this.env.readFileAtRevision(lookupPath, hash);
    this.state.compareSecondHash = hash;
    this.state.compareSecondContent = content;
    this.state.compareSecondMeta = meta;
    this.state.revisionViewMode = 'compare';
    this.state.pickingSecondForCompare = false;
  }

  cancelPickSecond(): void {
    this.state.pickingSecondForCompare = false;
  }

  stopCompareTwo(): void {
    this.state.compareSecondHash = null;
    this.state.compareSecondContent = null;
    this.state.compareSecondMeta = null;
  }

  private async loadHiddenRevisions(): Promise<void> {
    try {
      const hashes = await this.env.getHiddenRevisions();
      this.state.hiddenRevisionHashes = new Set(hashes);
    } catch {
      this.state.hiddenRevisionHashes = new Set();
    }
  }

  async hideRevision(hash: string): Promise<void> {
    await this.env.hideRevision(hash);
    this.state.hiddenRevisionHashes = new Set(this.state.hiddenRevisionHashes).add(hash);
    if (
      this.state.viewingRevisionHash === hash ||
      this.state.compareSecondHash === hash
    ) {
      this.clearRevisionView();
    }
  }

  async unhideRevision(hash: string): Promise<void> {
    await this.env.unhideRevision(hash);
    const next = new Set(this.state.hiddenRevisionHashes);
    next.delete(hash);
    this.state.hiddenRevisionHashes = next;
  }

  toggleShowHidden(): void {
    this.state.showHidden = !this.state.showHidden;
  }

  async restoreRevision(
    hash: string,
    liveContent: string,
  ): Promise<string> {
    const file = this.state.activeFile;
    if (!file) {
      throw new Error("no active file to restore");
    }

    await this.env.writeLibraryFile(file, liveContent);

    try {
      await this.env.snapshotFile(file, "Auto-save before restore");
    } catch (e) {
      if (!isNothingToSnapshotError(e)) throw e;
    }

    const meta = this.state.revisions.find((r) => r.hash === hash) ?? null;
    const lookupPath = meta?.path || file;
    const revisionContent = await this.env.readFileAtRevision(lookupPath, hash);
    await this.env.writeLibraryFile(file, revisionContent);

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
    if (!this.hydrated) return;
    void this.env.setSidebarCollapsed(this.state.sidebarCollapsed);
  }

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

  setLastDrawerTab(id: string) {
    this.state.lastDrawerTab = id;
    if (!this.hydrated) return;
    void this.env.setLastDrawerTab(id);
  }

  setDrawerDock(dock: DrawerDock) {
    this.state.drawerDock = dock;
    if (!this.hydrated) return;
    void this.env.setDrawerDock(dock);
  }

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

export function flattenLibraryTree(nodes: readonly LibraryNode[]): LibraryFile[] {
  const out: LibraryFile[] = [];
  const walk = (ns: readonly LibraryNode[]) => {
    for (const n of ns) {
      if (n.kind === "folder") {
        if (n.children) walk(n.children);
        continue;
      }
      out.push({
        path: n.path,
        name: n.name,
        updatedAt: n.updatedAt ?? "",
      });
    }
  };
  walk(nodes);
  return out;
}
