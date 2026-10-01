import { expect, signIn, TEST_PASSWORD, type Runtime } from "./fixtures";
import type { Page } from "../../frontend/node_modules/@playwright/test/index.js";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import type { DraftInput, DraftView, SubmissionView, PublicationPage } from "../../frontend/src/lib/content/types";
export const contentKnowledge = "e2e-rational-fractions", contentPath = "e2e-fractions-route", contentPackage = "e2e-content-math";
export async function originalInput(): Promise<DraftInput> {
 const root=resolve(__dirname,"../../backend/internal/content/testdata");
 const p=JSON.parse(await readFile(root+"/workflow-ready.json","utf8"));
 p.id=contentPackage;p.knowledge[0].id=contentKnowledge;p.units[0].id="e2e-fractions-unit";p.units[0].knowledge.id=contentKnowledge;p.assets[0].knowledge.id=contentKnowledge;p.assets[0].id="e2e-halves";p.units[0].assetIds=["e2e-halves"];
 p.units[0].angles=p.units[0].angles.map((a:{body:string;kind:string})=>({...a,body:a.body.replaceAll("asset:halves","asset:e2e-halves")}));
 const foundation=structuredClone(p.knowledge[0]);foundation.id="e2e-parts-of-a-whole";foundation.title="Parts of a whole";foundation.titleZh="整体分割";foundation.statement="Original technical fixture for equal parts of a whole.";foundation.relations=[];
 const unit=structuredClone(p.units[0]);unit.id="e2e-parts-unit";unit.knowledge.id=foundation.id;unit.assetIds=[];unit.angles=unit.angles.map((a:{body:string;kind:string})=>({...a,body:"Equal parts compose a whole."}));
 p.knowledge[0].relations=[{kind:"prerequisite",target:{id:foundation.id,version:1}}];
 p.knowledge[0].sources.push({kind:"external",author:"Technical fixture author",title:"Technical external source",url:"https://example.org/original-math",accessedAt:"2026-10-01",license:"CC0-1.0",attribution:"Original technical source reference, not a production claim."});
 p.knowledge.push(foundation);p.units.push(unit);
 p.paths=[{id:contentPath,version:1,domainIds:["elementary-mathematics"],title:"Original rational fractions route",titleZh:"分数路线",nodes:[{id:foundation.id,version:1},{id:contentKnowledge,version:1}]}];
 return {catalogueVersion:1,package:p,assetBytes:[{id:p.assets[0].id,base64:(await readFile(root+"/workflow-ready.svg")).toString("base64")}],sourceMap:[{knowledge:{id:contentKnowledge,version:1},batchSha256:"a".repeat(64),relativePath:"original/fractions.json",sha256:"b".repeat(64),legacyId:"technical-only",note:"Original technical browser fixture."}]};
}
export async function actor(page: Page, name: string) { await page.context().clearCookies(); await signIn(page, name, TEST_PASSWORD); }
export async function wireContent<T>(page: Page, path: string, method = "GET", input?: unknown, key?: string): Promise<{
    status: number;
    data: T;
}> { return page.evaluate(async ({ path, method, input, key }) => { try {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (method !== "GET") {
        const c = await fetch("/api/v1/auth/context", { headers: { "X-Requested-With": "MathMaster" } }).then(r => r.json());
        headers["Content-Type"] = "application/json";
        headers["X-CSRF-Token"] = c.data.csrfToken;
        headers["Idempotency-Key"] = key ?? crypto.randomUUID();
    }
    const response = await fetch(path, { method, headers, body: method === "GET" ? undefined : JSON.stringify(input), cache: "no-store", redirect: "error" });
    return { status: response.status, data: await response.json() };
}
catch {
    return { status: 503, data: null };
} }, { path, method, input, key }) as Promise<{
    status: number;
    data: T;
}>; }
export async function createDraft(page: Page) { await actor(page, "content_editor"); await page.goto("/editor"); await page.getByLabel("New package ID").fill(contentPackage); await page.getByLabel("New package version").fill("1"); await page.getByLabel("New catalogue version").fill("1"); await page.getByRole("button", { name: "Create draft", exact: true }).click(); await expect(page).toHaveURL(/\/editor\/drafts\/[0-9a-f-]+$/); const id = page.url().split("/").pop()!; const input = await originalInput(); await page.getByLabel("Import DraftInput JSON").setInputFiles({ name: "original-draft.json", mimeType: "application/json", buffer: Buffer.from(JSON.stringify(input)) }); await expect(page.getByRole("status")).toHaveText(/JSON imported/); await page.getByRole("button", { name: "Save draft", exact: true }).click(); await expect(page.getByRole("status")).toHaveText(/Draft saved/); return { id, input }; }
export async function submitDraft(page: Page) { const draft = await createDraft(page); await page.getByRole("button", { name: "Submit for review" }).click(); await expect(page).toHaveURL(/\/review\/[0-9a-f-]+$/); return { ...draft, submissionID: page.url().split("/").pop()! }; }
export async function approve(page: Page, id: string) { await actor(page, "content_reviewer"); await page.goto("/review/" + id); for (const label of ["Mathematics", "Explanations", "Relationships", "Sources", "Illustrations"])
    await page.getByLabel(label, { exact: true }).check(); await page.getByLabel("Independence statement").fill("Independently checked this isolated technical fixture."); await page.getByLabel("Review note").fill("All five technical review checks have been completed."); await page.getByRole("button", { name: "Approve submission" }).click(); await expect(page.getByRole("heading", { name: "Final review decision" })).toBeVisible(); }
export async function prepare(page: Page, id: string) { await actor(page, "content_admin"); await page.goto("/admin/publications"); await page.getByRole("checkbox").check(); await page.getByLabel("Publication reason").fill("Publish this isolated independently reviewed technical batch."); await page.getByRole("button", { name: "Prepare snapshot" }).click(); await expect(page.getByRole("status")).toHaveText(/Snapshot prepared/); const pageResult = await wireContent<PublicationPage>(page, "/api/v1/content/publications"); expect(pageResult.status).toBe(200); return pageResult.data.items[0]; }
export async function verifyPassword(page: Page) { await expect(page.getByRole("dialog")).toBeVisible(); await page.getByLabel("Your password", { exact: true }).fill(TEST_PASSWORD); await page.getByRole("button", { name: "Verify password" }).click(); await expect(page.getByRole("dialog")).toHaveCount(0); await expect(page.getByRole("status")).toHaveText(/Password verified/); }
export async function activate(page: Page) { await page.getByRole("button", { name: "Activate snapshot" }).click(); await verifyPassword(page); await page.getByRole("button", { name: "Activate snapshot" }).click(); await expect(page.getByRole("status")).toHaveText(/Snapshot activated/); }
