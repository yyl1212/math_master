import { defineConfig } from "../../frontend/node_modules/@playwright/test/index.js";
import {readFileSync} from 'node:fs';
import { resolve } from "node:path";
const root = resolve(__dirname, "../..");
const frontendRoot=process.env.MATH_MASTER_E2E_FRONTEND?resolve(process.env.MATH_MASTER_E2E_FRONTEND):root+'/frontend';
if(frontendRoot!==root+'/frontend'){
 const allowed=resolve(root,'.superpowers/sdd/2026-10-11-site-streamline/legacy/frontend');
 if(frontendRoot!==allowed)throw new Error('Unapproved test frontend');
 const marker=JSON.parse(readFileSync(frontendRoot+'/compatibility-build.json','utf8'));
 if(marker.kind!=='legacy-test-only'||marker.baseCommit!=='06d04a2eb20cf2a56cfec57283224583659c2ffd')throw new Error('Invalid test compatibility build');
}
const selectedFiles = process.argv.filter((arg) => /^[a-z-]+\.spec\.ts$/.test(arg));
export default defineConfig({
  testDir: ".",
  testMatch: selectedFiles.length ? selectedFiles.map((file) => "**/" + file) : "*.spec.ts",
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
    screenshot: "off",
    trace: "off",
    video: "off",
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
      cwd: frontendRoot,
      url: "http://127.0.0.1:18080",
      env: {
        GO_API_INTERNAL_URL: "http://127.0.0.1:18081",
        NEXT_TELEMETRY_DISABLED: "1",
        APP_ENV: "development",
        AUTH_PUBLIC_ORIGIN: "http://127.0.0.1:18080",
      },
      reuseExistingServer: false,
      timeout: 30000,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5000 },
      stdout: "ignore",
      stderr: "pipe",
    },
  ],
});
