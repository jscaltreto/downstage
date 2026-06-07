// Registry for the Help drawer's section navigator. Lives next to
// HelpTab.vue as a sibling TS module (mirrors workbench-tabs.ts) so
// pure-TS consumers can import the types without going through the
// Vue SFC shim.

import { defineAsyncComponent, type Component } from "vue";
import type { LucideIcon } from "lucide-vue-next";
import {
  Sparkles, Type, History, FolderTree, Download, Keyboard, Settings as SettingsIcon,
} from "lucide-vue-next";

export type HelpHost = "desktop" | "web";

export type HelpSectionId =
  | "getting-started"
  | "writing"
  | "versions"
  | "library"
  | "export"
  | "shortcuts"
  | "settings";

export interface HelpSection {
  id: HelpSectionId;
  label: string;
  icon: LucideIcon;
  component: Component;
  // Hosts that render this section. Web hides sections whose only
  // content is "this is a desktop feature" — they don't earn their
  // tab slot on a build that can't use the feature.
  hosts: ReadonlyArray<HelpHost>;
}

export interface ShortcutEntry {
  id: string;
  label: string;
  keys: string;
  group: string;
}

// Order matches the native menu (`internal/desktop/commands.go`) — file,
// edit, view, navigate, insert, format, help. Library/Help-section
// entries with unknown categories sort last (Number.MAX_SAFE_INTEGER
// fallback in sortShortcuts).
export const shortcutGroupOrder: ReadonlyArray<string> = [
  "file",
  "edit",
  "view",
  "navigate",
  "insert",
  "format",
  "help",
];

// User-facing labels for the group headers. Same keys as
// shortcutGroupOrder; missing keys fall through to a title-cased
// default in the ShortcutsSection renderer.
export const shortcutGroupLabel: Readonly<Record<string, string>> = {
  file: "File",
  edit: "Edit",
  view: "View",
  navigate: "Navigate",
  insert: "Insert",
  format: "Format",
  help: "Help",
};

// Stable sort: first by group order, then preserve incoming order
// within a group (the catalog already orders things the way the menu
// presents them). Returns a new array; does not mutate input.
export function sortShortcuts(entries: ShortcutEntry[]): ShortcutEntry[] {
  const groupIndex = new Map<string, number>();
  shortcutGroupOrder.forEach((g, i) => groupIndex.set(g, i));
  const unknownRank = Number.MAX_SAFE_INTEGER;
  return entries
    .map((entry, originalIndex) => ({ entry, originalIndex }))
    .sort((a, b) => {
      const ga = groupIndex.get(a.entry.group) ?? unknownRank;
      const gb = groupIndex.get(b.entry.group) ?? unknownRank;
      if (ga !== gb) return ga - gb;
      return a.originalIndex - b.originalIndex;
    })
    .map(({ entry }) => entry);
}

// Async-imported so the desktop-only-feature sections don't bloat the
// web bundle on first paint. Vite resolves these paths at build time
// against the help/ folder.
export const helpSections: ReadonlyArray<HelpSection> = [
  {
    id: "getting-started",
    label: "Getting Started",
    icon: Sparkles,
    component: defineAsyncComponent(() => import("./help/GettingStartedSection.vue")),
    hosts: ["desktop", "web"],
  },
  {
    id: "writing",
    label: "Writing",
    icon: Type,
    component: defineAsyncComponent(() => import("./help/WritingSection.vue")),
    hosts: ["desktop", "web"],
  },
  {
    id: "versions",
    label: "Versions",
    icon: History,
    component: defineAsyncComponent(() => import("./help/VersionsSection.vue")),
    hosts: ["desktop"],
  },
  {
    id: "library",
    label: "Library",
    icon: FolderTree,
    component: defineAsyncComponent(() => import("./help/LibrarySection.vue")),
    hosts: ["desktop"],
  },
  {
    id: "export",
    label: "Export",
    icon: Download,
    component: defineAsyncComponent(() => import("./help/ExportSection.vue")),
    hosts: ["desktop", "web"],
  },
  {
    id: "shortcuts",
    label: "Shortcuts",
    icon: Keyboard,
    component: defineAsyncComponent(() => import("./help/ShortcutsSection.vue")),
    hosts: ["desktop", "web"],
  },
  {
    id: "settings",
    label: "Settings",
    icon: SettingsIcon,
    component: defineAsyncComponent(() => import("./help/SettingsSection.vue")),
    hosts: ["desktop"],
  },
];

export function sectionsForHost(host: HelpHost): HelpSection[] {
  return helpSections.filter((s) => s.hosts.includes(host));
}

export const defaultHelpSection: HelpSectionId = "getting-started";
