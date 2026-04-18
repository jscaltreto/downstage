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
