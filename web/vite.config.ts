/// <reference types="vitest" />
import { execSync } from "node:child_process";
import { resolve } from "node:path";
import { defineConfig, type Plugin } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

function withTrailingSlash(path: string): string {
  return path.endsWith("/") ? path : `${path}/`;
}

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

// Wails expects the desktop frontend to be served from `index.html` in the
// configured frontend dir. Our source HTML lives at `desktop.html` so the
// web build can keep owning `index.html`. This plugin renames the HTML
// asset at bundle generation so we don't need a post-build `mv` hack.
function renameHtml(from: string, to: string): Plugin {
  return {
    name: "downstage-rename-html",
    generateBundle(_opts, bundle) {
      const asset = bundle[from];
      if (!asset) return;
      asset.fileName = to;
      bundle[to] = asset;
      delete bundle[from];
    },
  };
}

export default defineConfig({
  base: isDesktop ? "./" : `${siteBasePath}editor/`,
  plugins: [
    vue(),
    tailwindcss(),
    ...(isDesktop ? [renameHtml("desktop.html", "index.html")] : []),
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
      input: isDesktop
        ? { desktop: resolve(__dirname, "desktop.html") }
        : { main: resolve(__dirname, "index.html") },
    },
  },
  // Keep Vitest from picking up Playwright specs under `e2e/` — those run
  // against a real browser via `playwright test`, not happy-dom.
  test: {
    exclude: ["node_modules", "dist", "e2e/**"],
  },
});
