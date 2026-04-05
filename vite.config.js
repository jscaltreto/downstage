const path = require("path");
const { defineConfig } = require("vite");

const base = process.env.SITE_BASE_PATH || "/";

module.exports = defineConfig({
  base,
  build: {
    outDir: "dist",
    assetsDir: "assets",
    emptyOutDir: true,
    manifest: "asset-manifest.json",
    rollupOptions: {
      input: {
        styles: path.resolve(__dirname, "site/styles/tailwind.css"),
        script: path.resolve(__dirname, "site/assets/main.js"),
      },
      output: {
        entryFileNames: "assets/[name].[hash].js",
        assetFileNames: "assets/[name].[hash][extname]",
      },
    },
  },
});
