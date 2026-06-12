// Single source of truth for the external URLs the help surface points
// at. Sites that opened these as inline string literals
// (AppWeb.vue, WelcomeModal.vue, HelpTab.vue, desktop/commands.ts) now
// read from here so a URL change lands in one diff, not five.

export const helpLinks = {
  syntax: "https://www.getdownstage.com/syntax/",
  docs: "https://getdownstage.com/docs",
  github: "https://github.com/jscaltreto/downstage",
} as const;
