// @vitest-environment happy-dom
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { mount } from "@vue/test-utils";
import StatusBar from "../../desktop/StatusBar.vue";
import type { FileGitStatus } from "../../desktop/types";

// StatusBar rendering tests. Owned-by-host behaviors (click dispatch,
// data sourcing) are covered indirectly by AppDesktop tests; this file
// pins the pure rendering contract: dirty dot, snapshot labels,
// missing-file handling, and the always-enabled library button.

function mountWith(props: Partial<{
  libraryName: string;
  activeFile: string;
  cursor: { line: number; col: number };
  wordCount: number;
  gitStatus: FileGitStatus | null;
  hasLibrary: boolean;
  hasActiveFile: boolean;
}> = {}) {
  return mount(StatusBar, {
    props: {
      libraryName: "",
      activeFile: "",
      cursor: { line: 1, col: 1 },
      wordCount: 0,
      gitStatus: null,
      hasLibrary: false,
      hasActiveFile: false,
      ...props,
    },
  });
}

describe("StatusBar", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-18T00:00:00Z"));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("library button is always enabled and shows empty-state copy when no library", () => {
    const wrapper = mountWith({ hasLibrary: false });
    const btn = wrapper.find("button");
    expect(btn.exists()).toBe(true);
    expect((btn.element as HTMLButtonElement).disabled).toBe(false);
    expect(btn.text()).toContain("No library");
  });

  it("emits openFolder when the library button is clicked", async () => {
    const wrapper = mountWith({ hasLibrary: true, libraryName: "alpha" });
    await wrapper.find("button").trigger("click");
    expect(wrapper.emitted("openFolder")).toHaveLength(1);
  });

  it("renders cursor and word count only when an active file is present", () => {
    const withFile = mountWith({
      hasLibrary: true, libraryName: "alpha",
      hasActiveFile: true, activeFile: "play.ds",
      cursor: { line: 12, col: 7 }, wordCount: 1234,
    });
    expect(withFile.text()).toContain("Ln 12, Col 7");
    expect(withFile.text()).toContain("1,234 words");

    const noFile = mountWith({ hasLibrary: true, libraryName: "alpha" });
    expect(noFile.text()).not.toContain("Ln 1");
    expect(noFile.text()).not.toContain("words");
  });

  it("shows the dirty indicator when gitStatus.dirty and not missing", () => {
    const wrapper = mountWith({
      hasLibrary: true, hasActiveFile: true, activeFile: "play.ds",
      gitStatus: { dirty: true, headAt: "2026-04-17T23:55:00Z", hasHead: true, untracked: false, missing: false },
    });
    // amber-500 dot is a small span with rounded-full class.
    expect(wrapper.find("span.bg-amber-500").exists()).toBe(true);
  });

  it("omits the dirty dot when file is missing, even if dirty would otherwise be true", () => {
    const wrapper = mountWith({
      hasLibrary: true, hasActiveFile: true, activeFile: "gone.ds",
      gitStatus: { dirty: true, headAt: "2026-04-01T00:00:00Z", hasHead: true, untracked: false, missing: true },
    });
    expect(wrapper.find("span.bg-amber-500").exists()).toBe(false);
    expect(wrapper.text()).toContain("File moved or deleted");
  });

  it("formats relative snapshot time via a compact ladder", () => {
    const now = Date.parse("2026-04-18T00:00:00Z");
    const minus = (ms: number) => new Date(now - ms).toISOString();
    const cases: Array<[string, string]> = [
      [minus(5_000), "just now"],
      [minus(30_000), "30s ago"],
      [minus(5 * 60_000), "5m ago"],
      [minus(2 * 3_600_000), "2h ago"],
      [minus(3 * 86_400_000), "3d ago"],
      [minus(14 * 86_400_000), "2w ago"],
    ];
    for (const [iso, expected] of cases) {
      const wrapper = mountWith({
        hasLibrary: true, hasActiveFile: true, activeFile: "play.ds",
        gitStatus: { dirty: false, headAt: iso, hasHead: true, untracked: false, missing: false },
      });
      expect(wrapper.text()).toContain(`Last snapshot ${expected}`);
      wrapper.unmount();
    }
  });

  it("renders 'No snapshots' when a file has no history", () => {
    const wrapper = mountWith({
      hasLibrary: true, hasActiveFile: true, activeFile: "new.ds",
      gitStatus: { dirty: true, headAt: "", hasHead: false, untracked: true, missing: false },
    });
    expect(wrapper.text()).toContain("No snapshots");
  });
});
