import { expect, test } from "@playwright/test";
import { EditorPage } from "./pages/EditorPage";

// Matches the parent AppWeb.vue decoder:
//   decodeURIComponent(escape(atob(encoded)))
// which is the inverse of:
//   btoa(unescape(encodeURIComponent(raw)))
function encodeShared(raw: string): string {
  return btoa(unescape(encodeURIComponent(raw)));
}

const sharedBody = "# Shared From URL\n\nALICE\nHello traveller.\n";

test.describe("share links", () => {
  test("?content=<b64> promotes to a saved draft and clears the URL", async ({ page }) => {
    const editor = new EditorPage(page);
    const encoded = encodeShared(sharedBody);

    await page.addInitScript(() => {
      try {
        if (!window.sessionStorage.getItem("__e2e_cleared")) {
          window.localStorage.clear();
          window.sessionStorage.setItem("__e2e_cleared", "1");
        }
      } catch {}
    });
    await page.goto(`/editor/?content=${encodeURIComponent(encoded)}`);
    await page.waitForFunction(
      () => typeof (window as unknown as { downstage?: { parse?: unknown } }).downstage?.parse === "function",
    );

    await expect(editor.editor).toContainText("Shared From URL");

    // The URL is stripped of the ?content= param on successful import.
    await expect.poll(() => page.url()).not.toContain("content=");

    // The shared content landed as a real saved draft (persisted).
    await expect
      .poll(() => page.evaluate(() => window.localStorage.getItem("downstage-editor-drafts")))
      .toContain("Shared From URL");
  });

  test("?try=<b64> shows a placeholder that is not persisted until edited", async ({ page }) => {
    const editor = new EditorPage(page);
    const encoded = encodeShared("# Snippet Preview\n\nALICE\nTry it out.\n");

    await page.addInitScript(() => {
      try {
        if (!window.sessionStorage.getItem("__e2e_cleared")) {
          window.localStorage.clear();
          window.sessionStorage.setItem("__e2e_cleared", "1");
        }
      } catch {}
    });
    await page.goto(`/editor/?try=${encodeURIComponent(encoded)}`);
    await page.waitForFunction(
      () => typeof (window as unknown as { downstage?: { parse?: unknown } }).downstage?.parse === "function",
    );

    await expect(editor.editor).toContainText("Snippet Preview");

    // Nothing is persisted until the user edits.
    const drafts = await page.evaluate(() => window.localStorage.getItem("downstage-editor-drafts"));
    expect(drafts === null || drafts === "[]").toBe(true);

    // The ?try= param is intentionally preserved on initial load so reloads
    // re-hydrate the snippet.
    expect(page.url()).toContain("try=");
  });
});
