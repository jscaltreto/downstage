import { describe, expect, it, vi } from "vitest";
import {
  defaultEditorPreferences,
  parseEditorPreferencesBlob,
} from "../core/editor-prefs";

describe("parseEditorPreferencesBlob", () => {
  it("returns defaults for null/empty input", () => {
    expect(parseEditorPreferencesBlob(null)).toEqual(defaultEditorPreferences);
    expect(parseEditorPreferencesBlob("")).toEqual(defaultEditorPreferences);
    expect(parseEditorPreferencesBlob(undefined)).toEqual(defaultEditorPreferences);
  });

  it("round-trips a valid blob", () => {
    const blob = JSON.stringify({
      theme: "dark",
      previewHidden: true,
      spellcheckDisabled: true,
    });
    expect(parseEditorPreferencesBlob(blob)).toEqual({
      theme: "dark",
      previewHidden: true,
      spellcheckDisabled: true,
    });
  });

  it("falls back to defaults for malformed JSON (does not throw)", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(parseEditorPreferencesBlob("not-json")).toEqual(defaultEditorPreferences);
    expect(parseEditorPreferencesBlob("{unterminated")).toEqual(defaultEditorPreferences);
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it("rejects unknown theme strings", () => {
    const blob = JSON.stringify({ theme: "cyberpunk" });
    expect(parseEditorPreferencesBlob(blob).theme).toBe("system");
  });

  it("rejects non-boolean values for the boolean fields", () => {
    const blob = JSON.stringify({ previewHidden: "true", spellcheckDisabled: 1 });
    const parsed = parseEditorPreferencesBlob(blob);
    expect(parsed.previewHidden).toBe(false);
    expect(parsed.spellcheckDisabled).toBe(false);
  });

  it("fills in missing fields with defaults", () => {
    const blob = JSON.stringify({ theme: "light" });
    expect(parseEditorPreferencesBlob(blob)).toEqual({
      theme: "light",
      previewHidden: false,
      spellcheckDisabled: false,
    });
  });
});
