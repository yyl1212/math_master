import { test, expect } from "./fixtures";
import { actor, wireContent } from "./content-helpers";

test("passed diagnostic has no invented completion and original protected results remain", async ({ page, scene }) => {
  await scene("topic-cutover");
  await actor(page, "learning_other");
  await page.goto("/learn");
  const overview = await page.request.get("/api/v2/study/overview");
  expect(overview.status()).toBe(200);
  expect((await overview.json()).completed).toBe(0);
  const history = await wireContent<{ items: Array<{ attemptId: string | null; kind: string; state: string }> }>(page, "/api/v1/learning/history?limit=20");
  expect(history.status).toBe(200);
  const attempt = history.data.items.find(e => e.kind === "assessment" && e.state === "submitted")?.attemptId;
  expect(attempt).toBeTruthy();
  await page.goto("/learning-history?archive=legacy&module=assessment&attempt=" + attempt);
  await expect(page.getByText("5 / 5", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Submit five answers", exact: true })).toHaveCount(0);
  const records = await page.request.get("/api/v2/study/history?kind=completed");
  expect(records.status()).toBe(200);
  expect((await records.json()).items).toHaveLength(0);
});
