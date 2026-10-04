import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.M3_E2E_BASE_URL ?? "https://app.localhost:3443";
if (new URL(baseURL).protocol !== "https:")
  throw new Error("M3 acceptance requires an HTTPS browser origin.");

export default defineConfig({
  testDir: "./tests/m3-e2e",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: Boolean(process.env.CI),
  reporter: "html",
  use: {
    baseURL,
    ignoreHTTPSErrors:
      process.env.PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS === "true",
    // Retain action diagnostics, not network bodies/headers, DOM, or secrets.
    trace: {
      mode: "retain-on-failure",
      snapshots: false,
      screenshots: false,
      sources: false,
    },
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
