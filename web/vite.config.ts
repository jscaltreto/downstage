/// <reference types="vitest" />
import { execSync } from "node:child_process";
import { renameSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, type Plugin } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

function withTrailingSlash(path: string): string {
  return path.endsWith("/") ? path : `${path}/`;
}

import { computeBase } from "./src/vite-base";

const siteBasePath = withTrailingSlash(process.env.SITE_BASE_PATH || "/");
const isDesktop = process.env.DESKTOP_BUILD === "true";

function resolveAppVersion(): string {
  if (process.env.RELEASE_VERSION) {
    return process.env.RELEASE_VERSION;
  }

  try {
    return execSync("git describe --tags --always --dirty", {
      encoding: "utf8",
    }).trim();
  } catch {
    return "dev";
  }
}

// Wails expects its webview's root (`/`) to serve the desktop entry,
// and Wails' dev proxy injects the runtime into whatever comes back.
// Vite dev serves `index.html` at `/` by default, so in desktop mode
// we rewrite `/` → `/desktop.html` in the dev middleware. Keeps the
// web build untouched.
function serveDesktopAtRoot(): Plugin {
  return {
    name: "downstage-serve-desktop-at-root",
    apply: "serve",
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        if (req.url === "/" || req.url === "/index.html") {
          req.url = "/desktop.html";
        }
        next();
      });
    },
  };
}

// Wails expects the desktop frontend to be served from `index.html` in the
// configured frontend dir. Our source HTML lives at `desktop.html` so the
// web build can keep owning `index.html`. This plugin renames the HTML
// asset at bundle generation so we don't need a post-build `mv` hack.
// Wails expects index.html in the frontend dir, but our source HTML is
// desktop.html (so the web build can keep owning index.html). Rename on
// disk after the bundle is written — Vite's HTML assets don't appear in
// generateBundle's `bundle` object, so we can't intercept earlier.
function renameHtmlOnDisk(from: string, to: string): Plugin {
  let outDir = "";
  return {
    name: "downstage-rename-html",
    apply: "build",
    configResolved(config) {
      outDir = config.build.outDir;
    },
    closeBundle() {
      if (!outDir) return;
      try {
        renameSync(resolve(outDir, from), resolve(outDir, to));
      } catch {
        // If desktop.html wasn't produced (e.g. a non-desktop build) this
        // is a no-op.
      }
    },
  };
}

export default defineConfig(({ command }) => ({
  base: computeBase(isDesktop, command, siteBasePath),
  plugins: [
    vue(),
    tailwindcss(),
    ...(isDesktop ? [serveDesktopAtRoot(), renameHtmlOnDisk("desktop.html", "index.html")] : []),
  ],
  // Tailwind v4 runs via its Vite plugin — we don't need PostCSS. But
  // Vite's PostCSS loader walks up looking for `postcss.config.js`, and
  // the repo root has one for the Eleventy site that pulls Tailwind v3.
  // On machines where the root `node_modules` isn't populated (e.g. any
  // Windows contributor who only ran `npm install` in `web/`), that
  // config fails to resolve `tailwindcss` and the build dies. An inline
  // empty PostCSS config tells Vite to stop searching.
  css: { postcss: {} },
  define: {
    __APP_VERSION__: JSON.stringify(resolveAppVersion()),
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    emptyOutDir: true,
    rollupOptions: {
      input: (isDesktop
        ? { desktop: resolve(__dirname, "desktop.html") }
        : { main: resolve(__dirname, "index.html") }) as Record<string, string>,
    },
  },
  // Keep Vitest from picking up Playwright specs under `e2e/` — those run
  // against a real browser via `playwright test`, not happy-dom.
  test: {
    exclude: ["node_modules", "dist", "e2e/**"],
  },
}));
