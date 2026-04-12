import { reactive, watch } from "vue";
import type { EditorEnv, SavedDraft } from "./types";

export type Theme = "light" | "dark" | "system";

export interface State {
  drafts: SavedDraft[];
  activeDraftId: string | null;
  theme: Theme;
  isDark: boolean;
  appVersion: string;
}

function extractDocumentTitle(content: string) {
  return content.match(/^#\s+(.+)$/m)?.[1]?.trim() || "Untitled Play";
}

export class Store {
  public state: State;

  constructor(private env: EditorEnv) {
    this.state = reactive({
      drafts: [],
      activeDraftId: null,
      theme: (localStorage.getItem("downstage-theme") as Theme) || "system",
      isDark: false,
      appVersion: env.getAppVersion(),
    });

    this.initTheme();
  }

  async init() {
    this.state.drafts = await this.env.loadDrafts();
    this.state.activeDraftId = await this.env.loadActiveDraftId();
  }

  private initTheme() {
    const updateDark = () => {
      if (this.state.theme === "system") {
        this.state.isDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      } else {
        this.state.isDark = this.state.theme === "dark";
      }
      document.documentElement.classList.toggle("dark", this.state.isDark);
    };

    window.matchMedia("(prefers-color-scheme: dark)").onchange = updateDark;
    
    watch(() => this.state.theme, (newTheme) => {
      localStorage.setItem("downstage-theme", newTheme);
      updateDark();
    }, { immediate: true });
  }

  toggleTheme() {
    // If system or dark, switch to light. If light, switch to dark.
    if (this.state.isDark) {
        this.state.theme = "light";
    } else {
        this.state.theme = "dark";
    }
  }

  setTheme(theme: Theme) {
    this.state.theme = theme;
  }

  async saveActiveDraft(content: string) {
    const draft = this.state.drafts.find(d => d.id === this.state.activeDraftId);
    if (!draft) return;
    draft.content = content;
    draft.title = extractDocumentTitle(content);
    draft.updatedAt = new Date().toISOString();
    await this.env.saveDrafts(this.state.drafts);
  }
}
