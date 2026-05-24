import { reactive, watch } from "vue";
import type { EditorEnv, EditorPreferences, SavedDraft, Theme } from "./types";

export type { Theme } from "./types";

export interface State {
  drafts: SavedDraft[];
  activeDraftId: string | null;
  theme: Theme;
  previewHidden: boolean;
  spellcheckDisabled: boolean;
  isDark: boolean;
  appVersion: string;
}

function extractDocumentTitle(content: string) {
  return content.match(/^#\s+(.+)$/m)?.[1]?.trim() || "Untitled Play";
}

export class Store {
  public state: State;

  // hydrated guards the persistence watcher. Before init() finishes the
  // reactive `state` holds placeholder defaults; letting the watcher fire
  // during that window would overwrite whatever the env has on disk with
  // those placeholders. Flip to true as the final step of init().
  private hydrated = false;

  constructor(private env: EditorEnv) {
    this.state = reactive<State>({
      drafts: [],
      activeDraftId: null,
      theme: "system",
      previewHidden: false,
      spellcheckDisabled: false,
      isDark: false,
      appVersion: env.getAppVersion(),
    });

    this.initTheme();
    this.initPreferencePersistence();
  }

  async init() {
    this.state.drafts = await this.env.loadDrafts();
    this.state.activeDraftId = await this.env.loadActiveDraftId();

    const prefs = await this.env.getEditorPreferences();
    this.state.theme = prefs.theme;
    this.state.previewHidden = prefs.previewHidden;
    this.state.spellcheckDisabled = prefs.spellcheckDisabled;

    // Final step — open the persistence gate only after the env read
    // settles, so nothing we just wrote into state echoes back out.
    this.hydrated = true;
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

    watch(() => this.state.theme, () => {
      updateDark();
    }, { immediate: true });
  }

  private initPreferencePersistence() {
    // Single watcher over all three editor-pref fields. When any of them
    // changes after hydration, snapshot the full EditorPreferences and
    // hand the whole thing to the env — matches the backend's full-struct
    // write contract.
    watch(
      () => ({
        theme: this.state.theme,
        previewHidden: this.state.previewHidden,
        spellcheckDisabled: this.state.spellcheckDisabled,
      }),
      (snapshot) => {
        if (!this.hydrated) return;
        void this.persistEditorPreferences(snapshot);
      },
    );
  }

  private async persistEditorPreferences(prefs: EditorPreferences) {
    try {
      await this.env.setEditorPreferences(prefs);
    } catch (e) {
      console.error("failed to persist editor preferences:", e);
    }
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
