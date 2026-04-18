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
    return this.page.getByRole("button", { name: /Export PDF/ });
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

  async downloadPdf(): Promise<Download> {
    const downloadPromise = this.page.waitForEvent("download");
    await this.exportPdfButton.click();
    return downloadPromise;
  }
}
