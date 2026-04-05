import { defineConfig } from "vite";

function withTrailingSlash(path: string): string {
  return path.endsWith("/") ? path : `${path}/`;
}

const siteBasePath = withTrailingSlash(process.env.SITE_BASE_PATH || "/");

export default defineConfig({
  base: `${siteBasePath}editor/`,
  build: {
    outDir: "dist",
    assetsDir: "assets",
    emptyOutDir: true,
  },
});
