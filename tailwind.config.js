module.exports = {
  content: ["./site/**/*.{njk,md}", "./.eleventy.js", "./web/index.html"],
  theme: {
    extend: {
      colors: {
        ember: {
          950: "#0d0908",
          900: "#120d0c",
          850: "#231614",
          800: "#2f1c17",
          700: "#42271f",
          600: "#7f2718",
        },
        brass: {
          300: "#ffcf83",
          400: "#f0c67a",
          500: "#e3a857",
        },
        parchment: {
          100: "#f8ecd8",
          200: "#f4e8d5",
          300: "#d8c0a6",
        },
      },
      fontFamily: {
        sans: ["Avenir Next", "Segoe UI Variable", "Helvetica Neue", "Arial Narrow", "sans-serif"],
        serif: ["Iowan Old Style", "Palatino Linotype", "Book Antiqua", "URW Palladio L", "serif"],
        mono: ["Berkeley Mono", "SFMono-Regular", "Consolas", "Liberation Mono", "monospace"],
      },
      boxShadow: {
        stage: "0 24px 70px rgba(0, 0, 0, 0.38)",
      },
      borderRadius: {
        panel: "32px",
      },
      backgroundImage: {
        "page-glow":
          "radial-gradient(circle at top center, rgba(227, 168, 87, 0.16), transparent 26%), radial-gradient(circle at 20% 20%, rgba(127, 39, 24, 0.4), transparent 32%), linear-gradient(180deg, #231614 0%, #120d0c 55%, #0d0908 100%)",
      },
    },
  },
  plugins: [],
};
