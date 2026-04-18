import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { EditorPage } from "./pages/EditorPage";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

function encodeShared(raw: string): string {
  return btoa(unescape(encodeURIComponent(raw)));
}

test.describe("v1 migration", () => {
  test("V1 document disables Export PDF and upgrading re-enables it", async ({ page }) => {
    const editor = new EditorPage(page);
    const v1Source = await readFile(path.join(__dirname, "fixtures", "v1-doc.ds"), "utf8");

    await page.addInitScript(() => {
      try {
        if (!window.sessionStorage.getItem("__e2e_cleared")) {
          window.localStorage.clear();
          // Pre-dismiss the welcome modal so it cannot stack above the V1
          // migration modal and block clicks.
          window.localStorage.setItem("downstage-editor-welcome-dismissed", "true");
          window.sessionStorage.setItem("__e2e_cleared", "1");
        }
      } catch {}
    });
    // Seed the editor with a V1 document via the share-link path so we land
    // on V1 content immediately, without first running Example Play through
    // the pending-placeholder flow.
    await page.goto(`/editor/?content=${encodeURIComponent(encodeShared(v1Source))}`);
    await page.waitForFunction(
      () => typeof (window as unknown as { downstage?: { parse?: unknown } }).downstage?.parse === "function",
    );
    await expect(editor.newPlayButton).toBeVisible();

    // The V1 migration modal auto-opens when V1 content is detected. Dismiss
    // it via "Keep Raw Editing" so the toolbar/preview are unobstructed for
    // the guard assertions.
    const migrationModal = page.locator("dialog", {
      has: page.getByRole("heading", { name: /This looks like a V1 Downstage document/ }),
    });
    await expect(migrationModal).toBeVisible({ timeout: 15_000 });
    await migrationModal.getByRole("button", { name: "Keep Raw Editing" }).click();
    await expect(migrationModal).toBeHidden();

    // Guard 1: Export PDF button is disabled (toolbar-level `:disabled`).
    await expect(editor.exportPdfButton).toBeDisabled({ timeout: 15_000 });

    // Guard 2: clicking Export PDF after force-enabling it in the DOM still
    // fires the handleExport toast-block path, proving the second guard
    // rejects V1 exports.
    await editor.exportPdfButton.evaluate((el) => el.removeAttribute("disabled"));
    await editor.exportPdfButton.click();
    await expect(page.getByText(/Upgrade this V1 document/i)).toBeVisible();

    // Run the upgrade via the preview-pane banner. The modal is dismissed,
    // so there is only one "Update Script to V2" button left on the page.
    await page.getByRole("button", { name: /Update Script to V2/ }).click();

    // Post-upgrade: the toolbar re-enables, and exporting yields a PDF. Give
    // the render debounce time to settle before clicking.
    await expect(editor.exportPdfButton).toBeEnabled({ timeout: 15_000 });
    await expect(editor.exportPdfButton).toHaveAttribute(
      "title",
      /Export to PDF/,
    );

    const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
    await editor.exportPdfButton.click();
    const download = await downloadPromise;
    const pdf = await readFile((await download.path())!);
    expect(pdf.slice(0, 5).toString("utf8")).toBe("%PDF-");
  });
});
