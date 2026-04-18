/// <reference types="vitest" />
import { execSync } from "node:child_process";
import { resolve } from "node:path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

function withTrailingSlash(path: string): string {
  return path.endsWith("/") ? path : `${path}/`;
}

const siteBasePath = withTrailingSlash(process.env.SITE_BASE_PATH || "/");

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

export default defineConfig({
  base: `${siteBasePath}editor/`,
  plugins: [
    vue(),
    tailwindcss(),
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
      input: {
        main: resolve(__dirname, "index.html"),
      },
    },
  },
  // Keep Vitest from picking up Playwright specs under `e2e/` — those run
  // against a real browser via `playwright test`, not happy-dom.
  test: {
    exclude: ["node_modules", "dist", "e2e/**"],
  },
});
