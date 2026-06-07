import { describe, expect, it } from "vitest";
import { displayFileName, displayFilePath, normalizeFileRename } from "../../desktop/naming";

describe("naming helpers", () => {
  describe("displayFileName", () => {
    it("strips a single trailing .ds", () => {
      expect(displayFileName("Act One.ds")).toBe("Act One");
    });

    it("is case-insensitive on the extension", () => {
      expect(displayFileName("Act One.DS")).toBe("Act One");
    });

    it("leaves non-.ds names alone", () => {
      expect(displayFileName("notes.txt")).toBe("notes.txt");
      expect(displayFileName("Act One")).toBe("Act One");
    });

    it("does not strip an embedded .ds in the middle", () => {
      expect(displayFileName("things.ds.backup")).toBe("things.ds.backup");
    });
  });

  describe("displayFilePath", () => {
    it("strips .ds from the basename only", () => {
      expect(displayFilePath("folder/Act One.ds")).toBe("folder/Act One");
    });

    it("handles root paths", () => {
      expect(displayFilePath("Act One.ds")).toBe("Act One");
    });

    it("handles nested paths", () => {
      expect(displayFilePath("a/b/c/Scene.ds")).toBe("a/b/c/Scene");
    });
  });

  describe("normalizeFileRename", () => {
    it("appends .ds to a bare name", () => {
      expect(normalizeFileRename("Act One")).toBe("Act One.ds");
    });

    it("re-applies .ds when the user explicitly typed it", () => {
      expect(normalizeFileRename("Act One.ds")).toBe("Act One.ds");
    });

    it("does not strip a non-.ds extension the user typed", () => {
      // If the user genuinely typed ".txt", trust them. Resulting file
      // is "Act One.txt.ds" — ugly, but better than the alternative of
      // silently dropping characters from legitimate names like "v1.2".
      expect(normalizeFileRename("Act One.txt")).toBe("Act One.txt.ds");
    });

    it("trims whitespace", () => {
      expect(normalizeFileRename("  Act One  ")).toBe("Act One.ds");
    });

    it("returns empty string for empty input", () => {
      expect(normalizeFileRename("")).toBe("");
      expect(normalizeFileRename("   ")).toBe("");
    });

    it("preserves dots earlier in the name", () => {
      expect(normalizeFileRename("Act 2.1")).toBe("Act 2.1.ds");
    });
  });
});
