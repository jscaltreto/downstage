import { describe, expect, it } from "vitest";
import { formatAccelerator } from "../core/accelerators";

describe("formatAccelerator", () => {
  it("returns empty string for empty input", () => {
    expect(formatAccelerator("")).toBe("");
    expect(formatAccelerator("+++")).toBe("");
  });

  describe("mac formatting", () => {
    it("renders cmdorctrl as ⌘ and joins symbols with no separator", () => {
      expect(formatAccelerator("cmdorctrl+b", { isMac: true })).toBe("⌘B");
      expect(formatAccelerator("cmdorctrl+shift+s", { isMac: true })).toBe("⌘⇧S");
    });

    it("renders optionoralt as ⌥", () => {
      expect(formatAccelerator("cmdorctrl+optionoralt+f", { isMac: true })).toBe(
        "⌘⌥F",
      );
    });

    it("handles punctuation single-char keys", () => {
      expect(formatAccelerator("cmdorctrl+shift+/", { isMac: true })).toBe("⌘⇧/");
      expect(formatAccelerator("cmdorctrl+\\", { isMac: true })).toBe("⌘\\");
      expect(formatAccelerator("cmdorctrl+,", { isMac: true })).toBe("⌘,");
    });

    it("renders named keys with mac glyphs", () => {
      expect(formatAccelerator("cmdorctrl+enter", { isMac: true })).toBe("⌘⏎");
      expect(formatAccelerator("escape", { isMac: true })).toBe("Esc");
    });
  });

  describe("non-mac formatting", () => {
    it("renders cmdorctrl as Ctrl and joins with +", () => {
      expect(formatAccelerator("cmdorctrl+b", { isMac: false })).toBe("Ctrl+B");
      expect(formatAccelerator("cmdorctrl+shift+s", { isMac: false })).toBe(
        "Ctrl+Shift+S",
      );
    });

    it("renders optionoralt as Alt", () => {
      expect(formatAccelerator("cmdorctrl+optionoralt+f", { isMac: false })).toBe(
        "Ctrl+Alt+F",
      );
    });

    it("handles punctuation single-char keys", () => {
      expect(formatAccelerator("cmdorctrl+shift+/", { isMac: false })).toBe(
        "Ctrl+Shift+/",
      );
      expect(formatAccelerator("cmdorctrl+\\", { isMac: false })).toBe("Ctrl+\\");
      expect(formatAccelerator("cmdorctrl+,", { isMac: false })).toBe("Ctrl+,");
    });

    it("renders named keys with word labels", () => {
      expect(formatAccelerator("cmdorctrl+enter", { isMac: false })).toBe(
        "Ctrl+Enter",
      );
      expect(formatAccelerator("escape", { isMac: false })).toBe("Esc");
    });
  });

  it("uppercases single-letter passthrough keys", () => {
    expect(formatAccelerator("z", { isMac: true })).toBe("Z");
    expect(formatAccelerator("cmdorctrl+shift+z", { isMac: false })).toBe(
      "Ctrl+Shift+Z",
    );
  });

  it("title-cases multi-char passthrough keys (e.g. function keys)", () => {
    expect(formatAccelerator("F1", { isMac: false })).toBe("F1");
    expect(formatAccelerator("home", { isMac: false })).toBe("Home");
  });

  // Coverage net: every accelerator the desktop catalog emits today
  // must render to a non-empty string. Update this list when commands.go
  // gains new accelerators.
  it("renders every accelerator the desktop catalog currently emits", () => {
    const desktopAccelerators = [
      "cmdorctrl+n",
      "cmdorctrl+o",
      "cmdorctrl+s",
      "cmdorctrl+shift+s",
      "cmdorctrl+e",
      "cmdorctrl+,",
      "cmdorctrl+q",
      "cmdorctrl+z",
      "cmdorctrl+shift+z",
      "cmdorctrl+x",
      "cmdorctrl+c",
      "cmdorctrl+v",
      "cmdorctrl+a",
      "cmdorctrl+f",
      "cmdorctrl+optionoralt+f",
      "cmdorctrl+k",
      "cmdorctrl+\\",
      "cmdorctrl+shift+b",
      "cmdorctrl+b",
      "cmdorctrl+i",
      "cmdorctrl+u",
      "cmdorctrl+shift+x",
      "cmdorctrl+shift+/",
    ];
    for (const accel of desktopAccelerators) {
      expect(formatAccelerator(accel, { isMac: true })).not.toBe("");
      expect(formatAccelerator(accel, { isMac: false })).not.toBe("");
    }
  });
});
