import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { EditorPage } from "./pages/EditorPage";

const body = [
  "# Export Target",
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
  "A line of dialogue.",
  "",
].join("\n");

test.describe("export", () => {
  test("Copy writes the document to the clipboard", async ({ page, context }) => {
    // Clipboard permissions are also granted at the context level, but confirm
    // they're in effect for this origin.
    await context.grantPermissions(["clipboard-read", "clipboard-write"], {
      origin: "http://127.0.0.1:4173",
    });

    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent(body);

    await editor.copyButton.click();

    const clipboard = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboard).toContain("# Export Target");
    expect(clipboard).toContain("ALICE");
  });

  test("Save .ds produces a download with the expected filename and content", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent(body);

    const downloadPromise = page.waitForEvent("download");
    await editor.saveDsButton.click();
    const download = await downloadPromise;

    expect(download.suggestedFilename()).toBe("export-target.ds");
    const savedPath = await download.path();
    const saved = await readFile(savedPath!, "utf8");
    expect(saved).toContain("# Export Target");
    expect(saved).toContain("ALICE");
  });

  test("Export PDF yields a downloadable %PDF- file", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent(body);

    // PDF render uses diagnostics + render; wait for them to settle so the
    // V1-document guard doesn't fire falsely.
    await expect(editor.exportPdfButton).toBeEnabled();

    const download = await editor.downloadPdf();
    expect(download.suggestedFilename()).toMatch(/export-target-.*\.pdf$/);

    const pdfPath = await download.path();
    const buf = await readFile(pdfPath!);
    expect(buf.slice(0, 5).toString("utf8")).toBe("%PDF-");
    expect(buf.byteLength).toBeGreaterThan(1000);
  });

  test("Export PDF dialog respects A4 selection and persists it", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();
    await editor.welcomeStartButton.click();
    await editor.setEditorContent(body);
    await expect(editor.exportPdfButton).toBeEnabled();

    const download = await editor.downloadPdf("a4");
    const pdfPath = await download.path();
    const buf = await readFile(pdfPath!);
    expect(buf.slice(0, 5).toString("utf8")).toBe("%PDF-");
    expect(buf.byteLength).toBeGreaterThan(1000);

    const stored = await page.evaluate(() =>
      window.localStorage.getItem("downstage-editor-export-page-size"),
    );
    expect(stored).toBe("a4");

    // Reopen the dialog to confirm the prior selection is preselected.
    await editor.exportPdfButton.click();
    await expect(editor.exportDialog).toBeVisible();
    await expect(editor.pageSizeOption("a4")).toHaveAttribute("aria-checked", "true");
  });
});
