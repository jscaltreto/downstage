// Pure helper for Vite's `base` setting. Lives in src/ (not at
// vite.config.ts) so unit tests under src/__tests__ can import it
// without violating tsconfig's `rootDir: src`.
//
// Three branches:
//   - desktop dev → "/": served at the Vite dev-server root so
//     `http://localhost:5173/desktop.html` resolves its asset URLs.
//   - desktop build → "./": Wails embeds the bundle and loads it via
//     a file-style URL, which needs relative asset refs.
//   - web (dev or build) → "${siteBasePath}editor/": GitHub Pages
//     subpath; dev server mounts at the same subpath so local URLs
//     match prod.
export function computeBase(
  isDesktop: boolean,
  command: "serve" | "build",
  siteBasePath: string,
): string {
  if (isDesktop) return command === "serve" ? "/" : "./";
  return `${siteBasePath}editor/`;
}
