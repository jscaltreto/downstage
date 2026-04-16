export const isMac =
  typeof navigator !== "undefined" &&
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent);

const mod = isMac ? "\u2318" : "Ctrl";
const shift = isMac ? "\u21E7" : "Shift";
const alt = isMac ? "\u2325" : "Alt";

export interface Shortcut {
  label: string;
  keys: string;
  tooltip: string;
}

function formatKeys(parts: string[]): string {
  return isMac ? parts.join("") : parts.join("+");
}

export const shortcuts: Record<string, Shortcut> = {
  bold:           { label: "Bold",                keys: formatKeys([mod, "B"]),            tooltip: `Bold (${formatKeys([mod, "B"])})` },
  italic:        { label: "Italic",              keys: formatKeys([mod, "I"]),            tooltip: `Italic (${formatKeys([mod, "I"])})` },
  underline:     { label: "Underline",           keys: formatKeys([mod, "U"]),            tooltip: `Underline (${formatKeys([mod, "U"])})` },
  find:          { label: "Find",                keys: formatKeys([mod, "F"]),            tooltip: `Find (${formatKeys([mod, "F"])})` },
  findReplace:   { label: "Find & Replace",      keys: formatKeys([mod, "H"]),            tooltip: `Find & Replace (${formatKeys([mod, "H"])})` },
  findReplaceAlt:{ label: "Find & Replace",      keys: formatKeys([mod, alt, "F"]),       tooltip: `Find & Replace (${formatKeys([mod, alt, "F"])})` },
  preview:       { label: "Show / Hide Preview", keys: formatKeys([mod, shift, "P"]),     tooltip: `Toggle Preview (${formatKeys([mod, shift, "P"])})` },
  help:          { label: "Help",                keys: formatKeys([mod, shift, "/"]),     tooltip: `Help (${formatKeys([mod, shift, "/"])})` },
};
