import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { computeBase } from "../../vite-base";

const here = dirname(fileURLToPath(import.meta.url));
const webRoot = resolve(here, "../../..");
const repoRoot = resolve(webRoot, "..");

describe("vite config base matrix", () => {
  it("desktop dev serves at the Vite root", () => {
    expect(computeBase(true, "serve", "/")).toBe("/");
  });

  it("desktop build emits relative asset refs", () => {
    expect(computeBase(true, "build", "/")).toBe("./");
  });

  it("web build publishes to /editor/", () => {
    expect(computeBase(false, "build", "/")).toBe("/editor/");
  });

  it("web build respects a custom SITE_BASE_PATH", () => {
    expect(computeBase(false, "build", "/downstage/")).toBe("/downstage/editor/");
  });

  it("web dev command follows the same branch as web build", () => {
    // Symmetry check — the dev server mounts at the same subpath so
    // local URLs match what Pages serves.
    expect(computeBase(false, "serve", "/")).toBe("/editor/");
  });
});

describe("wails dev wiring", () => {
  // Regression guard for the "wails dev runs the web entry" bug. If
  // these drift, dev mode silently switches back to serving index.html
  // and the fix documented in the commit that added this test is gone.
  const wailsJson = JSON.parse(
    readFileSync(resolve(repoRoot, "cmd/downstage-write/wails.json"), "utf8"),
  );

  it("points the webview at the vite root", () => {
    // serverUrl is the root — the vite middleware in desktop mode
    // rewrites `/` to `/desktop.html`. Keeping it at root lets Wails
    // proxy and inject its runtime as usual.
    expect(wailsJson["frontend:dev:serverUrl"]).toMatch(/^http:\/\/localhost:\d+$/);
  });

  it("drives the dev watcher via npm run dev:desktop", () => {
    expect(wailsJson["frontend:dev:watcher"]).toContain("dev:desktop");
  });
});

describe("web package.json dev:desktop script", () => {
  const pkg = JSON.parse(
    readFileSync(resolve(webRoot, "package.json"), "utf8"),
  );

  it("defines dev:desktop and sets DESKTOP_BUILD=true", () => {
    const script = pkg.scripts["dev:desktop"];
    expect(script).toBeDefined();
    expect(script).toContain("DESKTOP_BUILD=true");
  });
});
