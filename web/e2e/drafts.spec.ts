import { expect, test } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { EditorPage } from "./pages/EditorPage";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const fixturesDir = path.join(__dirname, "fixtures");

test.describe("drafts", () => {
  test("welcome modal appears on first visit and is dismissed by Start Writing", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();

    await expect(editor.welcomeModalHeading).toBeVisible();
    await editor.welcomeStartButton.click();
    await expect(editor.welcomeModalHeading).toBeHidden();

    await page.reload();
    await page.waitForFunction(
      () => typeof (window as unknown as { downstage?: { parse?: unknown } }).downstage?.parse === "function",
    );
    await expect(editor.welcomeModalHeading).toBeHidden();
  });

  test("pending placeholder promotes to a saved draft on first edit", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();

    await editor.typeIntoEditor("\n\nNew line from typing");

    await editor.myDraftsButton.click();
    const myDraftsDialog = page.locator("dialog", {
      has: page.getByRole("heading", { name: "My Drafts", exact: true }),
    });
    await expect(myDraftsDialog).toBeVisible();
    await expect(myDraftsDialog.locator("div.group").first()).toContainText("The Example Play");
  });

  test("drafts survive a reload via localStorage", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.typeIntoEditor("\n\nLine to persist across reload");

    // saveDrafts is debounced (schedulePersist fires after ~250ms), so poll
    // localStorage rather than asserting immediately.
    await expect
      .poll(() => page.evaluate(() => window.localStorage.getItem("downstage-editor-drafts")), {
        timeout: 5_000,
      })
      .toContain("Line to persist across reload");

    await page.reload();
    await page.waitForFunction(
      () => typeof (window as unknown as { downstage?: { parse?: unknown } }).downstage?.parse === "function",
    );
    await expect(editor.editor).toContainText("Line to persist across reload");
  });

  test("New Play with a saved draft shows confirmation modal", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.typeIntoEditor("\n\nCommit to draft");

    await editor.newPlayButton.click();
    const confirm = page.getByRole("heading", { name: "Start a new play?" });
    await expect(confirm).toBeVisible();

    await page.getByRole("button", { name: "Start New Play" }).click();
    await expect(confirm).toBeHidden();

    await expect(editor.editor).toContainText("Untitled Play");
  });

  test("import .ds via filechooser loads the fixture content", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.myDraftsButton.click();

    const chooserPromise = page.waitForEvent("filechooser");
    await page.getByRole("button", { name: /Import \.ds File/ }).click();
    const chooser = await chooserPromise;
    await chooser.setFiles(path.join(fixturesDir, "sample-play.ds"));

    await expect(editor.editor).toContainText("The Playwright's Test");
  });

  test("delete draft flow removes the draft from the list", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.typeIntoEditor("\n\nDoomed draft");

    await editor.myDraftsButton.click();
    const modalDialog = page.locator("dialog", { has: page.getByRole("heading", { name: "My Drafts" }) });
    const draftRow = modalDialog.locator("div.group").first();
    await draftRow.hover();
    await draftRow.getByRole("button", { name: "Delete Draft" }).click();

    await page.getByRole("button", { name: "Delete", exact: true }).click();

    // After deletion there are no saved drafts, so the empty state (only the
    // Import button) should be what's left inside the modal.
    await expect(modalDialog.locator("div.group")).toHaveCount(0);
  });
});
