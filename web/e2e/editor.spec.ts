import { expect, test } from "@playwright/test";
import { EditorPage } from "./pages/EditorPage";

test.describe("editor", () => {
  test("typing updates editor content", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();

    await editor.typeIntoEditor("\n\nA fresh line from the test");
    await expect(editor.editor).toContainText("A fresh line from the test");
  });

  test("bold toolbar button wraps selection in markdown bold", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();

    await editor.setEditorContent("ALICE\nHello world\n");

    // Select the word "Hello"
    await editor.editor.click();
    await page.keyboard.press("Control+Home");
    await page.keyboard.press("ArrowDown");
    for (let i = 0; i < 5; i++) await page.keyboard.press("Shift+ArrowRight");

    await page.getByRole("button", { name: "Bold (Ctrl+B)" }).click();
    await expect(editor.editor).toContainText("**Hello**");
  });

  test("issues badge button appears for a document with warnings", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();

    // Document with an unknown character — produces a warning diagnostic.
    await editor.setEditorContent(
      [
        "# Troubled Play",
        "",
        "## Dramatis Personae",
        "",
        "ALICE - The known protagonist",
        "",
        "## ACT I",
        "",
        "### SCENE 1",
        "",
        "UNKNOWN_STRANGER",
        "I was never introduced.",
        "",
      ].join("\n"),
    );

    // The floating issues button is visible only when issues exist AND the
    // user is not actively typing; wait past the typing indicator window.
    const issuesButton = page.getByRole("button", { name: /script issue/ });
    await expect(issuesButton).toBeVisible({ timeout: 15_000 });
  });

  test("spellcheck modal adds and removes a custom dictionary word", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    // Promote to a saved draft so the allowlist can persist.
    await editor.typeIntoEditor("\n\nA fresh line");

    await page.getByRole("button", { name: "Spell Check" }).click();
    const spellDialog = page.locator("dialog", {
      has: page.getByRole("heading", { name: "Spell Check", exact: true }),
    });
    await expect(spellDialog).toBeVisible();

    await spellDialog.getByPlaceholder("Add a custom word").fill("FLORIZEL");
    await spellDialog.getByRole("button", { name: "Add", exact: true }).click();

    await expect(spellDialog.getByText("FLORIZEL", { exact: true })).toBeVisible();

    await spellDialog
      .getByRole("button", { name: "Remove FLORIZEL from this script dictionary" })
      .click();

    await expect(spellDialog.getByText("No custom words yet.")).toBeVisible();
  });
});
