// Canonical list of workbench drawer tabs. Kept in a plain TS module
// (rather than exported from WorkbenchDrawer.vue) so pure-TS consumers
// like `desktop/commands.ts` can import it without relying on the Vue
// SFC shim, which only declares a default export.

export type WorkbenchTab = 'issues' | 'find' | 'outline' | 'stats' | 'help';
