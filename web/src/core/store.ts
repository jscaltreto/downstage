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
    if (this.state.isDark) {
        this.state.theme = "light";
    } else {
        this.state.theme = "dark";
    }
  }

  setTheme(theme: Theme) {
    this.state.theme = theme;
  }

  activeDraft() {
    return this.state.drafts.find((draft) => draft.id === this.state.activeDraftId) || null;
  }

  async addSpellAllowlistWord(word: string) {
    const draft = this.activeDraft();
    if (!draft) return false;

    const trimmed = word.trim();
    if (!trimmed) return false;

    const key = trimmed.toLocaleLowerCase();
    if (draft.spellAllowlist.some((existing) => existing.toLocaleLowerCase() === key)) {
      return false;
    }

    draft.spellAllowlist.push(trimmed);
    draft.spellAllowlist.sort((a, b) => a.localeCompare(b));
    draft.updatedAt = new Date().toISOString();
    await this.env.saveDrafts(this.state.drafts);
    return true;
  }

  async removeSpellAllowlistWord(word: string) {
    const draft = this.activeDraft();
    if (!draft) return false;

    const key = word.trim().toLocaleLowerCase();
    const nextWords = draft.spellAllowlist.filter(
      (existing) => existing.toLocaleLowerCase() !== key,
    );
    if (nextWords.length === draft.spellAllowlist.length) {
      return false;
    }

    draft.spellAllowlist = nextWords;
    draft.updatedAt = new Date().toISOString();
    await this.env.saveDrafts(this.state.drafts);
    return true;
  }

  async saveActiveDraft(content: string) {
    const draft = this.activeDraft();
    if (!draft) return;
    draft.content = content;
    draft.title = extractDocumentTitle(content);
    draft.updatedAt = new Date().toISOString();
    await this.env.saveDrafts(this.state.drafts);
  }
}
