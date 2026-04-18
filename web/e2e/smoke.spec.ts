import { expect, test } from "@playwright/test";
import { EditorPage } from "./pages/EditorPage";

test.describe("smoke", () => {
  test("editor boots and WASM is ready", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();

    await expect(editor.newPlayButton).toBeVisible();
    await expect(editor.saveDsButton).toBeVisible();
    await expect(editor.editor).toBeVisible();

    await expect.poll(() =>
      page.evaluate(
        () =>
          typeof (window as unknown as { downstage?: { renderPDF?: unknown; renderHTML?: unknown } })
            .downstage?.renderPDF === "function"
          && typeof (window as unknown as { downstage?: { renderHTML?: unknown } })
            .downstage?.renderHTML === "function",
      ),
    ).toBe(true);
  });

  test("app header shows Downstage branding", async ({ page }) => {
    const editor = new EditorPage(page);
    await editor.gotoReady();

    await expect(page.getByRole("heading", { name: "Downstage", exact: true })).toBeVisible();
  });
});
