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
    this.state.projectFiles = await this.env.getProjectFiles();
    // Allowlist is project-scoped — reload after a project switch.
    this.state.spellAllowlist = await this.env.getSpellAllowlist();
    return path;
  }

  async selectFile(path: string): Promise<string> {
    this.state.activeFile = path;
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
