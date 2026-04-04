module.exports = function (eleventyConfig) {
  eleventyConfig.addPassthroughCopy({ "site/assets": "assets" });
  eleventyConfig.addPassthroughCopy({ "downstage_logo.png": "downstage_logo.png" });
  eleventyConfig.addCollection("homeSections", (collectionApi) =>
    collectionApi.getFilteredByGlob("site/content/home/*.md").sort((a, b) => a.data.order - b.data.order),
  );
  eleventyConfig.addCollection("docsSections", (collectionApi) =>
    collectionApi.getFilteredByGlob("site/content/docs/*.md").sort((a, b) => a.data.order - b.data.order),
  );

  return {
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
