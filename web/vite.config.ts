import { execSync } from "node:child_process";
import { defineConfig } from "vite";

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
  define: {
    __APP_VERSION__: JSON.stringify(resolveAppVersion()),
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    emptyOutDir: true,
  },
});
