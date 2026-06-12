// Format the menu-catalog accelerator strings emitted by
// `internal/desktop/commands.go` (e.g. `cmdorctrl+shift+/`,
// `cmdorctrl+optionoralt+f`) into the user-facing keystroke string used
// in toolbars, palettes, and the help drawer (`⌘⇧/` on macOS,
// `Ctrl+Shift+/` elsewhere).
//
// Leaf module: must not import from `platform.ts`. `platform.ts` imports
// this module to derive its own `shortcuts` map, so any reverse
// dependency would create a cycle.

export interface FormatAcceleratorOptions {
  isMac?: boolean;
}

const MAC_SYMBOL: Record<string, string> = {
  cmdorctrl: "⌘",
  shift: "⇧",
  optionoralt: "⌥",
  option: "⌥",
  alt: "⌥",
  ctrl: "⌃",
  control: "⌃",
  enter: "⏎",
  return: "⏎",
  escape: "Esc",
  esc: "Esc",
  tab: "⇥",
  space: "Space",
  arrowup: "↑",
  arrowdown: "↓",
  arrowleft: "←",
  arrowright: "→",
  backspace: "⌫",
  delete: "⌦",
};

const NON_MAC_LABEL: Record<string, string> = {
  cmdorctrl: "Ctrl",
  shift: "Shift",
  optionoralt: "Alt",
  option: "Alt",
  alt: "Alt",
  ctrl: "Ctrl",
  control: "Ctrl",
  enter: "Enter",
  return: "Enter",
  escape: "Esc",
  esc: "Esc",
  tab: "Tab",
  space: "Space",
  arrowup: "Up",
  arrowdown: "Down",
  arrowleft: "Left",
  arrowright: "Right",
  backspace: "Backspace",
  delete: "Delete",
};

// Guarded so Vitest / SSR / any Node import path doesn't blow up on
// missing `navigator`. Matches the existing pattern in
// `core/platform.ts`.
const defaultIsMac =
  typeof navigator !== "undefined" &&
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent);

export function formatAccelerator(
  raw: string,
  opts: FormatAcceleratorOptions = {},
): string {
  if (!raw) return "";
  const isMac = opts.isMac ?? defaultIsMac;
  const parts = raw
    .split("+")
    .map((p) => p.trim())
    .filter((p) => p.length > 0);
  if (parts.length === 0) return "";

  const rendered = parts.map((part) => renderPart(part, isMac));
  return isMac ? rendered.join("") : rendered.join("+");
}

function renderPart(part: string, isMac: boolean): string {
  const lookup = part.toLowerCase();
  const table = isMac ? MAC_SYMBOL : NON_MAC_LABEL;
  if (lookup in table) {
    return table[lookup];
  }
  // Single-character keys (letters, digits, punctuation): uppercase
  // on both platforms so `b` and `/` render as `B` and `/`.
  if (part.length === 1) {
    return part.toUpperCase();
  }
  // Multi-char passthrough (function keys F1-F12, named keys we
  // haven't mapped). Capitalize first letter for readability.
  return part.charAt(0).toUpperCase() + part.slice(1).toLowerCase();
}
