function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

const fs = require("fs");
const path = require("path");

const downstageGrammar = require("./editors/vscode/syntaxes/downstage.tmLanguage.json");
const pathPrefix = process.env.SITE_BASE_PATH || "/";

module.exports = function (eleventyConfig) {
  let highlighterPromise;
  async function getHighlighter() {
    if (!highlighterPromise) {
      highlighterPromise = import("shiki").then(({ createHighlighter }) =>
        createHighlighter({
          themes: ["github-dark"],
          langs: [
            "bash",
            "json",
            "markdown",
            "yaml",
            "text",
            "plaintext",
            {
              ...downstageGrammar,
              name: "downstage",
              displayName: "Downstage",
              aliases: ["ds"],
            },
          ],
        }),
      );
    }
    return highlighterPromise;
  }

  eleventyConfig.addNunjucksAsyncShortcode("renderCodeBlock", async (code, language = "text", label = null) => {
    const highlighter = await getHighlighter();
    const rendered = highlighter.codeToHtml(code || "", {
      lang: language,
      theme: "github-dark",
    });
    const displayLabel = label || language || "text";

    const tryButton = language === "downstage"
      ? `<button type="button" class="try-in-editor-button" aria-label="Open in web editor">Try it</button>`
      : "";

    return `
      <div class="code-block-shell group my-4 overflow-hidden rounded-2xl border border-white/10 bg-[#0d1117] shadow-stage">
        <div class="code-block-header">
          <span class="code-block-label">${escapeHtml(displayLabel)}</span>
          ${tryButton}
          <button type="button" class="copy-code-button" aria-label="Copy code to clipboard">
            Copy
          </button>
          <span class="copy-code-status sr-only" aria-live="polite"></span>
        </div>
        <div class="shiki-frame overflow-x-auto p-0">${rendered}</div>
        <template class="copy-source">${escapeHtml(code || "")}</template>
      </div>
    `;
  });

  eleventyConfig.addPassthroughCopy({ "downstage_logo.png": "downstage_logo.png" });
  eleventyConfig.addPassthroughCopy({ "editors/vscode/images/icon.png": "favicon.png" });
  eleventyConfig.addPassthroughCopy({ "web/dist": "editor" });
  eleventyConfig.addGlobalData("assetManifest", () => {
    const manifestPath = path.join(__dirname, "dist", "asset-manifest.json");
    if (!fs.existsSync(manifestPath)) {
      return {};
    }

    const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
    const findEntry = (sourcePath) =>
      Object.values(manifest).find((entry) => entry.src === sourcePath || entry.file === sourcePath);
    const stylesEntry = findEntry("site/styles/tailwind.css");
    const scriptEntry = findEntry("site/assets/main.js");

    return {
      styles: stylesEntry?.file ? `/${stylesEntry.file}` : undefined,
      script: scriptEntry?.file ? `/${scriptEntry.file}` : undefined,
    };
  });
  eleventyConfig.addCollection("homeSections", (collectionApi) =>
    collectionApi.getFilteredByGlob("site/content/home/*.md").sort((a, b) => a.data.order - b.data.order),
  );
  eleventyConfig.addCollection("docsSections", (collectionApi) =>
    collectionApi.getFilteredByGlob("site/content/docs/*.md").sort((a, b) => a.data.order - b.data.order),
  );

  return {
    pathPrefix,
    dir: {
      input: "site",
      includes: "_includes",
      data: "_data",
      output: "dist",
    },
    markdownTemplateEngine: "njk",
    htmlTemplateEngine: "njk",
  };
};
