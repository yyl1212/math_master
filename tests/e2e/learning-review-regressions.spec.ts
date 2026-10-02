import { test, expect, fitsViewport } from "./fixtures";
import { actor, beginPractice, arithmeticAnswer, knowledgeDetail, learningState, rootKnowledge, wireLearning } from "./learning-helpers";
import type { AttemptView } from "../../frontend/src/lib/learning/types";

test("a published single-choice practice renders and submits its choice through the strict contract", async ({ page, scene, request, runtime }) => {
 await scene("learning-choice"); await actor(page, "auth_learner");
 const p = await beginPractice(page, "learning-practice-only");
 expect(p.question.type).toBe("single_choice");
 expect(JSON.stringify(p)).not.toMatch(/correctChoiceId|correctNumeric|explanation/);
 await page.getByRole("radio", { name: arithmeticAnswer(p.question.prompt), exact: true }).check();
 await page.getByRole("button", { name: "Submit answer", exact: true }).click();
 await expect(page.getByText("Correct", { exact: true })).toBeVisible();
 await expect(page.getByText("Practice answered.", { exact: true })).toBeVisible();
 await page.reload();
 await expect(page.getByText("Correct", { exact: true })).toBeVisible();
 const facts = await learningState(request, runtime);
 expect(facts.practiceAttempts).toBe(1); expect(facts.qualifications).toBe(0); expect(facts.learningEvents).toBe(0);
 await fitsViewport(page);
});

test("an unread unlocked lesson can pass a mixed normal check before explicit reading grants qualification", async ({ page, scene, request, runtime }) => {
 await scene("learning-choice"); await actor(page, "auth_learner"); await page.goto("/knowledge/" + rootKnowledge);
 await expect(page.getByLabel("Assessment mode")).toHaveValue("node");
 await page.getByRole("button", { name: "Start five-question assessment" }).click();
 await expect(page).toHaveURL(/\/assessments\/[0-9a-f-]+$/);
 const response = await wireLearning<AttemptView>(page, "/api/v1/learning/assessments/" + page.url().split("/").pop());
 expect(response.status).toBe(200); const a = response.data;
 expect(a.questions.filter(q => q.type === "single_choice")).toHaveLength(1);
 expect(a.questions.filter(q => q.type === "numeric")).toHaveLength(4);
 expect(JSON.stringify(a)).not.toMatch(/correctChoiceId|correctNumeric|explanation/);
 for (const q of a.questions) {
  if (q.type === "single_choice") await page.getByRole("group", { name: "Answer for question " + q.position }).getByRole("radio", { name: arithmeticAnswer(q.prompt), exact: true }).check();
  else await page.getByLabel("Answer for question " + q.position).fill(arithmeticAnswer(q.prompt));
 }
 await page.getByRole("button", { name: "Submit five answers" }).click();
 await expect(page).toHaveURL(/\/result$/); await expect(page.getByText("5 / 5", { exact: true })).toBeVisible();
 await page.reload(); await expect(page.getByText("5 / 5", { exact: true })).toBeVisible();
 let facts = await learningState(request, runtime);
 expect(facts.answers).toBe(5); expect(facts.results).toBe(1); expect(facts.qualifications).toBe(0); expect(facts.learningEvents).toBe(0);
 expect((await knowledgeDetail(page, "learning-middle")).state.canEnter).toBe(false);
 await page.goto("/knowledge/" + rootKnowledge);
 await page.getByRole("button", { name: "Start learning", exact: true }).click();
 await page.getByRole("button", { name: "Mark as learned" }).click();
 await expect(page.getByText("Reading completion recorded.")).toBeVisible();
 facts = await learningState(request, runtime);
 expect(facts.qualifications).toBe(1); expect(facts.learningEvents).toBe(2);
 expect((await knowledgeDetail(page, "learning-middle")).state.canEnter).toBe(true);
 expect((await knowledgeDetail(page, "learning-target")).state.canEnter).toBe(false);
 await fitsViewport(page);
});

test("another tab changing the shared account clears the original tab's unsent private draft", async ({ page, scene, request, runtime }) => {
 await scene("learning-basic"); await actor(page, "auth_learner");
 const p = await beginPractice(page);
 await page.getByLabel("Answer for question 1").fill("private unsent spelling");
 let oldTabWrites = 0;
 page.on("request", r => { if (r.method() === "POST" && new URL(r.url()).pathname.startsWith("/api/v1/learning/")) oldTabWrites++; });
 const other = await page.context().newPage();
 try {
  await actor(other, "learning_other");
  await expect(page.getByLabel("Answer for question 1")).toHaveCount(0);
  await expect(page.getByText("Resource not found.", { exact: true })).toBeVisible();
  expect((await wireLearning(other, "/api/v1/learning/practice/" + p.summary.id)).status).toBe(404);
  const facts = await learningState(request, runtime);
  expect(facts.practiceAttempts).toBe(1); expect(facts.results).toBe(0); expect(facts.learningEvents).toBe(0); expect(oldTabWrites).toBe(0);
 } finally { await other.close(); }
 await fitsViewport(page);
});
