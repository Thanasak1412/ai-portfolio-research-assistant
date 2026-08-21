import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.M2_E2E_BASE_URL ?? "https://app.localhost:3443";

export default defineConfig({
  testDir: "./tests/m2-e2e",
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: "html",
  use: {
    baseURL,
    ignoreHTTPSErrors:
      process.env.PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS === "true",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
