import type { Download, Locator, Page } from "@playwright/test";
import { expect } from "@playwright/test";

export type DrawerTab = "outline" | "stats" | "issues" | "find" | "help";

export class EditorPage {
  readonly page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  // --- Header toolbar ---

  get newPlayButton(): Locator {
    return this.page.getByRole("button", { name: /New Play/ });
  }

  get myDraftsButton(): Locator {
    return this.page.getByRole("button", { name: /My Drafts/ });
  }

  get copyButton(): Locator {
    return this.page.getByRole("button", { name: /^Copy$/ });
  }

  get saveDsButton(): Locator {
    return this.page.getByRole("button", { name: /Save \.ds/ });
  }

  get exportPdfButton(): Locator {
    // The toolbar toggle and the confirm button inside the export dialog both
    // read "Export PDF". Scope this accessor to the header so tests can still
    // target the toolbar trigger unambiguously.
    return this.page.locator("header").getByRole("button", { name: /Export PDF/ });
  }

  get exportDialog(): Locator {
    return this.page.locator("dialog", {
      has: this.page.getByRole("heading", { name: "Export PDF" }),
    });
  }

  get exportConfirmButton(): Locator {
    return this.exportDialog.locator('[data-testid="export-confirm"]');
  }

  pageSizeOption(value: "letter" | "a4"): Locator {
    return this.exportDialog.locator(`button[data-page-size="${value}"]`);
  }

  exportStyleOption(value: "standard" | "condensed"): Locator {
    return this.exportDialog.locator(`button[data-export-style="${value}"]`);
  }

  layoutOption(value: "single" | "2up" | "booklet"): Locator {
    return this.exportDialog.locator(`button[data-pdf-layout="${value}"]`);
  }

  get layoutGroup(): Locator {
    return this.exportDialog.locator('[data-testid="layout-group"]');
  }

  get gutterRow(): Locator {
    return this.exportDialog.locator('[data-testid="gutter-row"]');
  }

  get gutterValueInput(): Locator {
    return this.exportDialog.locator('[data-testid="gutter-value"]');
  }

  get gutterUnitSelect(): Locator {
    return this.exportDialog.locator('[data-testid="gutter-unit"]');
  }

  // --- Workbench drawer ---

  get drawer(): Locator {
    return this.page.locator('section[aria-label="Workbench"]');
  }

  drawerTab(tab: DrawerTab): Locator {
    const name = {
      outline: "Outline",
      stats: "Stats",
      issues: /^Issues/,
      find: /Find & Replace/,
      help: "Help",
    }[tab];
    return this.page.getByRole("tab", { name: name as string });
  }

  // --- Editor toolbar buttons (inside Editor.vue) ---

  get outlineToolbarButton(): Locator {
    return this.page.getByRole("button", { name: "Outline", exact: true });
  }

  get statsToolbarButton(): Locator {
    return this.page.getByRole("button", { name: "Stats", exact: true });
  }

  get findToolbarButton(): Locator {
    return this.page.getByRole("button", { name: /Find .*\(/ });
  }

  get helpToolbarButton(): Locator {
    return this.page.getByRole("button", { name: /Help .*\(/ });
  }

  // --- CodeMirror editor ---

  get editor(): Locator {
    return this.page.locator(".cm-editor .cm-content");
  }

  // --- Modals ---

  get welcomeModalHeading(): Locator {
    return this.page.getByRole("heading", { name: "Welcome to Downstage" });
  }

  get welcomeStartButton(): Locator {
    return this.page.getByRole("button", { name: "Start Writing" });
  }

  async gotoReady(options: { clearStorage?: boolean } = {}): Promise<void> {
    const { clearStorage = true } = options;
    if (clearStorage) {
      // Clear localStorage once per tab so intra-test reloads still see the
      // state the user just set (e.g. welcome-dismissed flag).
      await this.page.addInitScript(() => {
        try {
          if (!window.sessionStorage.getItem("__e2e_cleared")) {
            window.localStorage.clear();
            window.sessionStorage.setItem("__e2e_cleared", "1");
          }
        } catch {}
      });
    }
    await this.page.goto("/editor/");
    await this.page.waitForFunction(
      () => typeof (window as unknown as { downstage?: { parse?: unknown } }).downstage?.parse === "function",
      undefined,
      { timeout: 30_000 },
    );
    await expect(this.newPlayButton).toBeVisible();
  }

  async typeIntoEditor(text: string): Promise<void> {
    await this.editor.click();
    await this.page.keyboard.type(text);
  }

  async setEditorContent(text: string): Promise<void> {
    await this.editor.click();
    await this.page.keyboard.press("ControlOrMeta+a");
    await this.page.keyboard.press("Delete");
    if (text.length > 0) {
      await this.page.keyboard.type(text);
    }
  }

  async openDrawer(tab: DrawerTab): Promise<void> {
    const toolbarToggle: Record<Exclude<DrawerTab, "issues">, Locator> = {
      outline: this.outlineToolbarButton,
      stats: this.statsToolbarButton,
      find: this.findToolbarButton,
      help: this.helpToolbarButton,
    };

    if (tab === "issues") {
      // Issues tab has no dedicated toolbar toggle — open Outline then switch.
      await toolbarToggle.outline.click();
      await expect(this.drawer).toHaveAttribute("aria-hidden", "false");
      await this.drawerTab("issues").click();
      await expect(this.drawerTab("issues")).toHaveAttribute("aria-selected", "true");
      return;
    }

    await toolbarToggle[tab].click();
    await expect(this.drawer).toHaveAttribute("aria-hidden", "false");
    await expect(this.drawerTab(tab)).toHaveAttribute("aria-selected", "true");
  }

  async downloadPdf(options?: {
    pageSize?: "letter" | "a4";
    style?: "standard" | "condensed";
    layout?: "single" | "2up" | "booklet";
    gutterValue?: number;
    gutterUnit?: "in" | "mm";
  }): Promise<Download> {
    await this.exportPdfButton.click();
    await expect(this.exportDialog).toBeVisible();

    if (options?.pageSize) {
      await this.pageSizeOption(options.pageSize).click();
      await expect(this.pageSizeOption(options.pageSize)).toHaveAttribute("aria-checked", "true");
    }
    if (options?.style) {
      await this.exportStyleOption(options.style).click();
      await expect(this.exportStyleOption(options.style)).toHaveAttribute("aria-checked", "true");
    }
    if (options?.layout) {
      await this.layoutOption(options.layout).click();
      await expect(this.layoutOption(options.layout)).toHaveAttribute("aria-checked", "true");
    }
    if (options?.gutterValue !== undefined) {
      await this.gutterValueInput.fill(String(options.gutterValue));
    }
    if (options?.gutterUnit) {
      await this.gutterUnitSelect.selectOption(options.gutterUnit);
    }

    const downloadPromise = this.page.waitForEvent("download");
    await this.exportConfirmButton.click();
    return downloadPromise;
  }
}
