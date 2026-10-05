import { expect, signIn, TEST_PASSWORD, type Runtime } from "./fixtures";
import type { Page } from "../../frontend/node_modules/@playwright/test/index.js";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { actor, wireContent } from "./content-helpers";
import type { DraftInput, DraftView, SubmissionView, ValidationReport, PublicationSummary, PublicationPage } from "../../frontend/src/lib/question/types";
export { actor, wireContent as wireQuestion };
export const questionKnowledge = "e2e-question-fractions", questionPackage = "e2e-original-question-bank";
export async function originalQuestionInput(): Promise<DraftInput> {
    const p = JSON.parse(await readFile(resolve(__dirname, "../../content/questions/elementary-rationals.v1.json"), "utf8"));
    p.id = questionPackage;
    for (const t of p.templates) {
        t.knowledge = { id: questionKnowledge, version: 1 };
        for (const c of t.coverage)
            c.knowledge = t.knowledge;
        t.promptTemplate = t.engine.family === "missing_operand" ? "Find the missing operand: $x + {{known}} = {{result}}$." : t.engine.family === "rational_comparison" ? "Compare ${{left}}$ and ${{right}}$." : "Calculate ${{left}} + {{right}}$.";
    }
    for (const b of p.blueprints)
        b.knowledge = { id: questionKnowledge, version: 1 };
    return { catalogueVersion: 1, questionPackage: p, sourceMap: [{ knowledge: { id: questionKnowledge, version: 1 }, batchSha256: "a".repeat(64), relativePath: "original/question-source.json", sha256: "b".repeat(64), legacyId: "technical-only", note: "Original test source, not a production lesson." }] };
}
export async function createQuestion(page: Page, input?: DraftInput) { await actor(page, "content_editor"); const value = input ?? await originalQuestionInput(); await page.goto("/editor/questions"); await page.getByLabel("New question package ID").fill(value.questionPackage.id); await page.getByRole("button", { name: "Create draft", exact: true }).click(); await expect(page).toHaveURL(/\/editor\/questions\/drafts\/[0-9a-f-]+$/); const id = page.url().split("/").pop()!; await page.getByText("Editable JSON and source mapping", { exact: true }).click(); await page.getByLabel("Import question JSON file").setInputFiles({ name: "original-question.json", mimeType: "application/json", buffer: Buffer.from(JSON.stringify(value)) }); await expect(page.getByRole("status")).toHaveText(/imported/i); await page.getByRole("button", { name: "Save draft", exact: true }).click(); await expect(page.getByRole("status")).toHaveText(/Draft saved/); return { id, input: value }; }
export async function submitQuestion(page: Page) {
    const d = await createQuestion(page);
    await page.getByRole("button", { name: "Validate saved revision" }).click();
    await expect(page.getByText("Machine checks passed. Reviewer approval is required.")).toBeVisible();
    await page.getByRole("button", { name: "Submit for review" }).click();
    await expect(page.getByRole("status")).toHaveText(/frozen for independent review/);
    const list = await wireContent<{
        items: SubmissionView[];
    }>(page, "/api/v1/question-bank/submissions?scope=mine");
    expect(list.status).toBe(200);
    return { ...d, submissionID: list.data.items[0].id };
}
export async function approveQuestion(page: Page, id: string, decision = "approve") {
    await actor(page, "content_reviewer");
    await page.goto("/review/questions/" + id);
    for (const label of ["Mathematics", "Explanations", "Objectives", "Sources", "Illustrations", "Generation"])
        await page.getByLabel(label, { exact: true }).check();
    await page.getByLabel("Independence statement").fill("Different real fixture authors and reviewer accounts.");
    await page.getByLabel("Generation review statement").fill("All finite questions and independent verifier checked, no skipped combinations.");
    await page.getByLabel("Review note").fill("All six requirements checked on original technical fixture sources.");
    await page.getByRole("button", { name: decision === "approve" ? "Approve submission" : "Return for changes" }).click();
    await expect(page.getByRole("heading", { name: "Final review decision" })).toBeVisible();
}
export async function prepareQuestion(page: Page, id: string) { await actor(page, "content_admin"); await page.goto("/admin/question-publications"); await page.getByRole("checkbox").check(); await page.getByLabel("Publication reason", { exact: true }).fill("Publish the original independently reviewed question fixture."); await page.getByRole("button", { name: "Prepare snapshot" }).click(); await expect(page.getByRole("status")).toHaveText(/Snapshot prepared/); const r = await wireContent<PublicationPage>(page, "/api/v1/question-bank/publications"); expect(r.status).toBe(200); return r.data.items[0]; }
export async function activateQuestion(page: Page) { await page.getByRole("button", { name: "Activate snapshot" }).click(); await expect(page.getByRole("dialog")).toBeVisible(); await page.getByLabel("Your password", { exact: true }).fill(TEST_PASSWORD); await page.getByRole("button", { name: "Verify password" }).click(); await expect(page.getByRole("dialog")).toHaveCount(0); await page.getByRole("button", { name: "Retry previous request" }).click(); await expect(page.getByRole("status")).toHaveText(/Snapshot activated/); }
import type { APIRequestContext } from "../../frontend/node_modules/@playwright/test/index.js";
export type QuestionDatabaseState = {
    workspaces: number;
    maxRevision: number;
    submissions: number;
    reviews: number;
    published: number;
    head: string | null;
    manifestSha: string | null;
    members: number;
    withdrawals: number;
    events: Record<string, number>;
};
export async function questionState(request: APIRequestContext, runtime: Runtime): Promise<QuestionDatabaseState> { const response = await request.get(runtime.controlURL + "/question/state", { headers: { Authorization: "Bearer " + runtime.token } }); expect(response.status()).toBe(200); return response.json(); }
export async function wireReauth(page: Page) { const r = await wireContent(page, "/api/v1/auth/reauth", "POST", { password: TEST_PASSWORD }); expect(r.status).toBe(200); }
export async function wireCreateQuestion(page: Page, input: DraftInput) { const r = await wireContent<DraftView>(page, "/api/v1/question-bank/drafts", "POST", input); expect(r.status).toBe(201); return r.data; }
export async function wireSubmitQuestion(page: Page, id: string, revision: number) { const g = await wireContent<ValidationReport>(page, `/api/v1/question-bank/drafts/${id}/validate`, "POST", { expectedRevision: revision }); expect(g.status).toBe(200); expect(g.data.readyToSubmit).toBe(true); const sub = await wireContent<SubmissionView>(page, `/api/v1/question-bank/drafts/${id}/submit`, "POST", { expectedRevision: revision, expectedDigest: g.data.digest }); expect(sub.status).toBe(201); return sub.data; }
export async function wirePrepareQuestion(page: Page, id: string) { const [q, k] = await Promise.all([wireContent<PublicationPage>(page, "/api/v1/question-bank/publications?limit=1"), wireContent<{
        head: string | null;
    }>(page, "/api/v1/content/publications?limit=1")]); expect(q.status).toBe(200); expect(k.status).toBe(200); const r = await wireContent<PublicationSummary>(page, "/api/v1/question-bank/publications/prepare", "POST", { submissionIds: [id], expectedKnowledgeHead: k.data.head, expectedQuestionHead: q.data.head, reason: "Prepare an exact real independently approved technical revision." }); expect(r.status).toBe(201); return r.data; }
export async function wireActivateQuestion(page: Page, p: PublicationSummary) { return wireContent<PublicationSummary>(page, `/api/v1/question-bank/publications/${p.id}/activate`, "POST", { expectedKnowledgeHead: p.baseKnowledgeHead, expectedQuestionHead: p.baseQuestionHead, expectedManifestSha: p.manifestSha, reason: "Activate an exact original test manifest after real password verification." }); }
