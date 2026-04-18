import type { EditorPreferences, Theme } from "./types";

export const defaultEditorPreferences: EditorPreferences = {
  theme: "system",
  previewHidden: false,
  spellcheckDisabled: false,
};

const validThemes: readonly Theme[] = ["light", "dark", "system"];

function isValidTheme(value: unknown): value is Theme {
  return typeof value === "string" && (validThemes as readonly string[]).includes(value);
}

// Given a raw JSON string (or null/undefined), returns a fully-populated
// EditorPreferences. Corrupt, partial, or type-mismatched JSON falls back
// to defaults without throwing. A malformed blob must not brick the editor.
export function parseEditorPreferencesBlob(raw: string | null | undefined): EditorPreferences {
  if (!raw) return { ...defaultEditorPreferences };
  try {
    const parsed = JSON.parse(raw) as Partial<EditorPreferences>;
    return {
      theme: isValidTheme(parsed?.theme) ? parsed.theme : defaultEditorPreferences.theme,
      previewHidden:
        typeof parsed?.previewHidden === "boolean"
          ? parsed.previewHidden
          : defaultEditorPreferences.previewHidden,
      spellcheckDisabled:
        typeof parsed?.spellcheckDisabled === "boolean"
          ? parsed.spellcheckDisabled
          : defaultEditorPreferences.spellcheckDisabled,
    };
  } catch (e) {
    console.warn("downstage: ignoring malformed editor prefs in localStorage:", e);
    return { ...defaultEditorPreferences };
  }
}
