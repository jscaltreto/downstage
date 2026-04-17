import { reactive } from "vue";
import type { DesktopCapabilities, ProjectFile, Revision } from "./types";

export interface WorkspaceState {
  projectPath: string | null;
  projectFiles: ProjectFile[];
  activeFile: string | null;
  revisions: Revision[];
  sidebarCollapsed: boolean;
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
        localStorage.getItem("downstage-sidebar-collapsed") === "true",
    });
  }

  async init() {
    this.state.projectPath = await this.env.getCurrentProject();
    if (this.state.projectPath) {
      this.state.projectFiles = await this.env.getProjectFiles();
    }
  }

  async openFolder(): Promise<string | null> {
    const path = await this.env.openProjectFolder();
    if (!path) return null;
    this.state.projectPath = path;
    this.state.activeFile = null;
    this.state.revisions = [];
    this.state.projectFiles = await this.env.getProjectFiles();
    return path;
  }

  async selectFile(path: string): Promise<string> {
    this.state.activeFile = path;
    const content = await this.env.readProjectFile(path);
    await this.loadRevisions();
    return content;
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

  toggleSidebar() {
    this.state.sidebarCollapsed = !this.state.sidebarCollapsed;
    localStorage.setItem(
      "downstage-sidebar-collapsed",
      String(this.state.sidebarCollapsed),
    );
  }
}
