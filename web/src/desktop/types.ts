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
  getCurrentProject(): Promise<string>;
  getLastActiveFile(): Promise<string>;
  setActiveProjectFile(rel: string): Promise<void>;
  getSpellAllowlist(): Promise<string[]>;
  addSpellAllowlistWord(word: string): Promise<boolean>;
  removeSpellAllowlistWord(word: string): Promise<boolean>;
}

export interface DesktopCapabilities extends EditorEnv, ProjectEnv {}
