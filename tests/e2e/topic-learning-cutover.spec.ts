import { test, expect, fitsViewport } from "./fixtures";
import { actor, wireContent } from "./content-helpers";
import { report } from "./feedback-helpers";

test("real legacy facts migrate then topics learning, notes, review, timeline and feedback work", async ({ page, request, runtime, scene }) => {
  await scene("topic-cutover");
  await actor(page, "auth_learner");
  await page.goto("/learn");
  await expect(page.locator(".metrics p").filter({ hasText: "Completed" }).locator("strong")).toHaveText("1");
  await page.goto("/topics/msc-00a01");
  await page.getByRole("link", { name: /Connected addition/ }).click();
  await expect(page.getByRole("button", { name: "Complete learning", exact: true })).toBeEnabled();
  await page.getByLabel("Personal note", { exact: true }).fill("Original note after real legacy migration.");
  await page.getByRole("button", { name: "Save note", exact: true }).click();
  await expect(page.getByText("Note saved.", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Complete learning", exact: true }).click();
  await expect(page.getByRole("button", { name: "Start review", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Start review", exact: true }).click();
  await expect(page.getByRole("button", { name: "Finish review", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Finish review", exact: true }).click();
  await expect(page.getByRole("button", { name: "Start review", exact: true })).toBeEnabled();
  await page.getByRole("link", { name: "Learning history", exact: true }).last().click();
  await page.goto("/learning-history");
  await expect(page.getByText("Imported study fact", { exact: true }).first()).toBeVisible();
  await report(page, "/feedback/new?kind=knowledge&id=learning-root");
  const upgrade = await request.post(runtime.controlURL + "/study/publication/upgrade", {
    headers: { Authorization: "Bearer " + runtime.token },
  });
  expect(upgrade.status()).toBe(204);
  await page.goto("/knowledge/learning-root");
  await expect(page.getByText("Completed", { exact: true })).toBeVisible();
  await expect(page.getByText("The material has changed. Review the current version.", { exact: true })).toBeVisible();
  const detail = await page.request.get("/api/v2/study/knowledge/learning-root");
  expect(detail.status()).toBe(200);
  const current = await detail.json();
  const retired = await wireContent(page, "/api/v1/learning/knowledge/learning-root/start", "POST", {
    knowledge: { id: current.currentKnowledge.id, version: current.currentKnowledge.version, sha256: current.currentKnowledge.sha256 },
    expectedKnowledgeHead: current.pair.knowledgeHead,
  });
  expect(retired.status).toBe(410);
  await fitsViewport(page);
});
