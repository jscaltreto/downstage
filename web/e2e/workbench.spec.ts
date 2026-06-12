import { expect, test } from "@playwright/test";
import { EditorPage } from "./pages/EditorPage";

const samplePlay = [
  "# Workbench Smoke",
  "Author: E2E Suite",
  "",
  "## Dramatis Personae",
  "",
  "ALICE - The protagonist",
  "",
  "## ACT I",
  "",
  "### SCENE 1",
  "",
  "ALICE",
  "Hello world, this is a line of dialogue.",
  "",
].join("\n");

test.describe("workbench", () => {
  test("Outline tab lists headings from the document", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent(samplePlay);

    await editor.openDrawer("outline");

    const drawer = editor.drawer;
    await expect(drawer).toContainText("Workbench Smoke");
    // Outline renders symbol display names in title case ("Act I", "Scene 1")
    // regardless of the source heading's case in the .ds file.
    await expect(drawer).toContainText(/Act I/i);
    await expect(drawer).toContainText(/Scene 1/i);
  });

  test("Stats tab shows manuscript counts", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent(samplePlay);

    await editor.openDrawer("stats");

    const drawer = editor.drawer;
    await expect(drawer.getByText(/Est\. Runtime/)).toBeVisible();
    await expect(drawer.getByText(/^Words$/)).toBeVisible();
    await expect(drawer.getByText("1 act")).toBeVisible();
  });

  test("Issues tab surfaces diagnostics from the document", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    // ALICE is declared in Dramatis Personae but never speaks — produces an
    // info-severity `dp-character-no-dialogue` diagnostic.
    await editor.setEditorContent(
      [
        "# Troubled Play",
        "",
        "## Dramatis Personae",
        "",
        "ALICE - Declared but silent",
        "",
        "## ACT I",
        "",
        "### SCENE 1",
        "",
        "> A stage direction with no dialogue.",
        "",
      ].join("\n"),
    );

    // The floating issues button opens the Issues tab when diagnostics arrive.
    const issuesFab = page.getByRole("button", { name: /script issue/ });
    await expect(issuesFab).toBeVisible({ timeout: 15_000 });
    await issuesFab.click();

    await expect(editor.drawer).toHaveAttribute("aria-hidden", "false");
    await expect(editor.drawerTab("issues")).toHaveAttribute("aria-selected", "true");
    await expect(editor.drawer).toContainText(/ALICE.*never speaks|dp-character-no-dialogue/i, {
      timeout: 15_000,
    });
  });

  test("Find & Replace finds and replaces text", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent("ALICE\nBanana banana banana.\n");

    await editor.openDrawer("find");
    const drawer = editor.drawer;

    await drawer.getByLabel("Find", { exact: true }).fill("banana");
    await drawer.getByLabel("Replace with", { exact: true }).fill("apple");
    await drawer.getByLabel("Replace all matches").click();

    await expect(editor.editor).toContainText("apple apple apple");
  });

  test("Help tab renders the sectioned help nav", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();

    await editor.openDrawer("help");

    const drawer = editor.drawer;
    // Section nav is a tablist; the inner buttons carry role="tab".
    // Web host filters out desktop-only sections (Versions, Library,
    // Settings). The four cross-host sections must render.
    const nav = drawer.getByRole("tablist", { name: "Help sections" });
    await expect(nav.getByRole("tab", { name: "Getting Started" })).toBeVisible();
    await expect(nav.getByRole("tab", { name: "Writing" })).toBeVisible();
    await expect(nav.getByRole("tab", { name: "Export" })).toBeVisible();
    await expect(nav.getByRole("tab", { name: "Shortcuts" })).toBeVisible();

    await nav.getByRole("tab", { name: "Shortcuts" }).click();
    await expect(drawer).toContainText(/Bold|Italic|Find/);
  });

  test("drawer close button collapses the drawer", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();

    await editor.openDrawer("outline");
    await expect(editor.drawer).toBeVisible();

    await editor.drawer.getByRole("button", { name: "Close workbench" }).click();
    await expect(editor.drawer).toHaveAttribute("aria-hidden", "true");
  });
});
