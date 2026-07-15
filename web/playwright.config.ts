import { defineConfig, devices } from "@playwright/test";

// End-to-end suite for the core user flows (TODO.md Phase 5 cross-cutting:
// "create ticket, run workflow, approve/reject step"). The backend (Go
// conduit-server, in-memory) and the Next.js web app must both be reachable;
// `npm run e2e` (scripts/run-e2e.mjs) spins both up before launching this
// suite, so locally you only run that one command.
const API_URL = process.env.CONDUIT_API_URL || "http://localhost:8080";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
  use: {
    baseURL: "http://localhost:3000",
    trace: "on-first-retry",
    actionTimeout: 10_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  // Playwright owns the web app; the Go backend is started by run-e2e.mjs.
  webServer: {
    command: "npm run dev",
    url: "http://localhost:3000",
    timeout: 120_000,
    reuseExistingServer: false,
    env: { CONDUIT_API_URL: API_URL },
  },
});
