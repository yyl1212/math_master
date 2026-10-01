import {
  test as base,
  expect,
  type Page,
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
}>({
  runtime: async ({}, use) => {
    const state: Runtime = JSON.parse(
      await readFile(resolve(__dirname, "runtime.local.json"), "utf8"),
    );
    await use(state);
  },
  scene: async ({ runtime, request }, use) => {
    const change = async (name: string) => {
      const r = await request.post(runtime.controlURL + "/scene/" + name, {
        headers: { Authorization: "Bearer " + runtime.token },
      });
      // Do not include the control address, authorization or body in failure diagnostics.
      if (r.status() !== 204) throw new Error("Test scene change failed");
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
