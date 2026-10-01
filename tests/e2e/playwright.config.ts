import { defineConfig } from "../../frontend/node_modules/@playwright/test/index.js";
import { resolve } from "node:path";
const root = resolve(__dirname, "../..");
export default defineConfig({
  testDir: ".",
  testMatch: "*.spec.ts",
  outputDir: "../../test-results",
  workers: 1,
  retries: 0,
  globalTimeout: 480000,
  timeout: 30000,
  expect: { timeout: 5000 },
  reporter: [
    ["list"],
    ["html", { outputFolder: "../../playwright-report", open: "never" }],
  ],
  use: {
    baseURL: "http://127.0.0.1:18080",
    browserName: "chromium",
    screenshot: "only-on-failure",
    trace: "off",
  },
  projects: [
    { name: "desktop", use: { viewport: { width: 1280, height: 900 } } },
    { name: "mobile", use: { viewport: { width: 390, height: 844 } } },
  ],
  webServer: [
    {
      command: "./backend/bin/e2e-harness",
      cwd: root,
      url: "http://127.0.0.1:18082/readyz",
      reuseExistingServer: false,
      timeout: 30000,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5000 },
      stdout: "ignore",
      stderr: "pipe",
    },
    {
      command: "npm run start -- --hostname 127.0.0.1 --port 18080",
      cwd: root + "/frontend",
      url: "http://127.0.0.1:18080",
      env: {
        GO_API_INTERNAL_URL: "http://127.0.0.1:18081",
        NEXT_TELEMETRY_DISABLED: "1",
      },
      reuseExistingServer: false,
      timeout: 30000,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5000 },
      stdout: "ignore",
      stderr: "pipe",
    },
  ],
});
