import {
  test as base,
  expect,
  type Page,
  type APIRequestContext,
} from "../../frontend/node_modules/@playwright/test/index.js";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
export type Runtime = {
  apiURL: string;
  controlURL: string;
  token: string;
  database: string;
  assetSha: string;
  knowledgeId: string;
  pathId: string;
};
export const test = base.extend<{
  runtime: Runtime;
  scene: (name: string) => Promise<void>;
  safeDiagnostics: void;
}>({
  safeDiagnostics: [async ({ page }, use, info) => {
    await use();
    if (info.status !== info.expectedStatus && !page.isClosed()) {
      await page.screenshot({ path: info.outputPath("masked-failure.png"), fullPage: true, mask: [page.locator("input, textarea")] }).catch(() => {});
    }
  }, { auto: true }],
  runtime: async ({}, use) => {
    let state: Runtime;
    try { state = JSON.parse(await readFile(resolve(__dirname, "runtime.local.json"), "utf8")); }
    catch { throw new Error("Test runtime unavailable"); }
    await use(state);
  },
  scene: async ({ runtime, request }, use) => {
    const change = async (name: string) => {
      await changeScene(request, runtime, name);
    };
    await change("draft");
    await use(change);
  },
});
export { expect };
export async function fitsViewport(page: Page) {
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth + 1,
      ),
    )
    .toBe(true);
}

export const TEST_PASSWORD = "Test-only 中文数学密码 with spaces";
export const NEW_PASSWORD = "New test-only 中文数学密码 with spaces";
export async function signIn(page: Page, username: string, password: string) {
  await page.goto("/login");
  await page.getByLabel("Username", { exact: true }).fill(username);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page).toHaveURL(/\/account$/);
}
export async function safeScreenshot(page: Page, info: { outputPath: (name: string) => string }, name: string) {
  if (!process.env.CI) await page.screenshot({ path: info.outputPath(name + ".png"), fullPage: true, mask: [page.locator("input, textarea")] });
}

export async function changeScene(request: Pick<APIRequestContext, "post">, runtime: Pick<Runtime, "controlURL" | "token">, name: string) {
  try {
    const response = await request.post(runtime.controlURL + "/scene/" + name, { headers: { Authorization: "Bearer " + runtime.token }, timeout: 5000 });
    if (response.status() !== 204) throw new Error("Test scene change failed");
  } catch { throw new Error("Test scene change failed"); }
}
