// Playwright E2E (docs/05 §2): the journeys in e2e/ against a running app. MRs and main run Chromium,
// the release tag run on staging all three browsers. Test tags @ci-only and @smoke: docs/05 §2.
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "e2e",
  // One journey at a time: staging shares the host with production (docs/05 §2).
  workers: 1,
  // No automatic retries: a flaky test is a bug (docs/05 §1).
  retries: 0,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    // App origin: local stack (deploy/docker-compose.local.yml), CI stack (http://caddy) or staging.
    baseURL: process.env.MSP_E2E_BASE_URL ?? "http://localhost:8000",
    trace: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      // MSP_E2E_CHROMIUM_PATH: a Chromium already on the machine instead of the one Playwright downloads.
      use: { ...devices["Desktop Chrome"], launchOptions: { executablePath: process.env.MSP_E2E_CHROMIUM_PATH } },
    },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
