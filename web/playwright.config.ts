import { defineConfig, devices } from "@playwright/test";

const isCI = !!process.env.CI;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  workers: isCI ? 2 : undefined,
  reporter: isCI ? [["list"], ["html", { open: "never" }]] : "list",
  timeout: 60_000,
  expect: { timeout: 10_000 },

  use: {
    baseURL: "http://127.0.0.1:4173/editor/",
    trace: "on-first-retry",
    video: "retain-on-failure",
    screenshot: "only-on-failure",
    permissions: ["clipboard-read", "clipboard-write"],
    testIdAttribute: "data-testid",
  },

  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        colorScheme: "light",
        reducedMotion: "reduce",
      },
    },
  ],

  webServer: {
    command: "npm run preview:strict",
    url: "http://127.0.0.1:4173/editor/",
    reuseExistingServer: !isCI,
    timeout: 60_000,
    stdout: "pipe",
    stderr: "pipe",
  },
});
