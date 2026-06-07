import { formatAccelerator } from "./accelerators";

export const isMac =
  typeof navigator !== "undefined" &&
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent);

export interface Shortcut {
  label: string;
  keys: string;
  tooltip: string;
}

function shortcut(label: string, raw: string, tooltipLabel?: string): Shortcut {
  const keys = formatAccelerator(raw, { isMac });
  return { label, keys, tooltip: `${tooltipLabel ?? label} (${keys})` };
}

export const shortcuts: Record<string, Shortcut> = {
  bold:        shortcut("Bold",                "cmdorctrl+b"),
  italic:      shortcut("Italic",              "cmdorctrl+i"),
  underline:   shortcut("Underline",           "cmdorctrl+u"),
  find:        shortcut("Find",                "cmdorctrl+f"),
  findReplace: shortcut("Find & Replace",      "cmdorctrl+optionoralt+f"),
  preview:     shortcut("Show / Hide Preview", "cmdorctrl+\\", "Toggle Preview"),
  help:        shortcut("Help",                "cmdorctrl+shift+/"),
};
