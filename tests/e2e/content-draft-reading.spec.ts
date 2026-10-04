import { test, expect, fitsViewport, safeScreenshot } from "./fixtures";
import { actor, createDraft, wireContent } from "./content-helpers";
import type { DraftView } from "../../frontend/src/lib/content/types";

// Only the harness's random test database is used; no real review or publication.
test("savedDraftReaderSearchSwitchingPrivateIllustrationAndSiteFeedback", async ({ page, scene }, info) => {
    await scene("content");
    const { id, input } = await createDraft(page);
    await page.getByRole("link", { name: "Read saved draft", exact: true }).click();
    await expect(page).toHaveURL("/editor/drafts/" + id + "/preview");
    await expect(page.getByRole("heading", { name: "Read knowledge draft" })).toBeVisible();
    await expect(page.getByText("Saved revision 2 · Status: editing")).toBeVisible();
    const navigation = page.getByRole("navigation", { name: "Knowledge points" });
    const search = page.getByRole("searchbox", { name: "Search knowledge points" });
    await expect(navigation.getByRole("button")).toHaveCount(2);
    await expect(page.locator(".katex").first()).toBeVisible();
    const image = page.getByRole("img", { name: "Equal halves", exact: true });
    await expect(image).toHaveAttribute("src", `/api/v1/content/drafts/${id}/assets/${input.package.assets[0].sha256}`);
    await page.getByText("Original fixture illustration.", { exact: true }).scrollIntoViewIfNeeded();
    await expect.poll(() => image.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
    await expect(page.getByRole("link", { name: "Technical external source" })).toHaveAttribute("href", "https://example.org/original-math");
    await fitsViewport(page);
    await search.fill("整体分割");
    await expect(navigation.getByRole("button")).toHaveCount(1);
    await expect(page.getByRole("heading", { name: "e2e-parts-unit · Version 1" })).toBeVisible();
    await expect(image).toHaveCount(0);
    await search.fill(" RATIONAL ");
    await expect(navigation.getByRole("button")).toHaveCount(1);
    await expect(page.getByRole("heading", { name: "e2e-fractions-unit · Version 1" })).toBeVisible();
    await search.fill("no-such-knowledge");
    await expect(page.getByText(/No matching knowledge points/)).toBeVisible();
    await expect(page.getByRole("heading", { name: "Statement" })).toHaveCount(0);
    await search.fill("e2e-");
    await expect(navigation.getByRole("button")).toHaveCount(2);
    const next = page.getByRole("button", { name: "Next knowledge point", exact: true });
    await next.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("heading", { name: "e2e-parts-unit · Version 1" })).toBeVisible();
    await expect(next).toBeDisabled();
    await expect(page.getByLabel("Preview reference")).toHaveValue(`Draft: ${id}\nSaved revision: 2\nPackage: e2e-content-math v1\nKnowledge: e2e-parts-of-a-whole v1`);
    await fitsViewport(page);
    await safeScreenshot(page, info, "draft-reader");
    await page.getByRole("link", { name: "Give site feedback", exact: true }).click();
    await expect(page).toHaveURL("/feedback/new?kind=site&area=other");
    await expect(page.getByText(/Source is not available|Invalid feedback source/)).toHaveCount(0);
});

test("readingEntryProtectsUnsavedEditsAndTheReaderUsesOnlySavedData", async ({ page, scene }) => {
    await scene("content");
    const { id } = await createDraft(page);
    const saved = await wireContent<DraftView>(page, "/api/v1/content/drafts/" + id);
    await page.getByLabel("Knowledge 1 statement").fill("Keep this unsaved statement in the editor.");
    await expect(page.getByRole("button", { name: "Read saved draft", exact: true })).toBeDisabled();
    await expect(page.getByRole("link", { name: "Read saved draft", exact: true })).toHaveCount(0);
    const reader = await page.context().newPage();
    const writes: string[] = [];
    reader.on("request", request => {
        if (["POST", "PUT", "PATCH", "DELETE"].includes(request.method()) && /\/api\/v1\/(content|learning)\//.test(request.url())) writes.push(request.url());
    });
    try {
        await reader.goto("/editor/drafts/" + id + "/preview");
        await expect(reader.getByText("Keep this unsaved statement in the editor.")).toHaveCount(0);
        await expect(reader.getByRole("heading", { name: "e2e-fractions-unit · Version 1" })).toBeVisible();
        await reader.getByRole("button", { name: "Next knowledge point", exact: true }).click();
        expect(writes).toEqual([]);
        const current = await wireContent<DraftView>(reader, "/api/v1/content/drafts/" + id);
        expect(current.data.revision).toBe(saved.data.revision);
        expect(current.data.package).toEqual(saved.data.package);
    } finally {
        await reader.close();
    }
    await expect(page.getByLabel("Knowledge 1 statement")).toHaveValue("Keep this unsaved statement in the editor.");
    await page.getByRole("button", { name: "Save draft", exact: true }).click();
    await expect(page.getByRole("status")).toHaveText(/Draft saved/);
    await page.getByRole("link", { name: "Read saved draft", exact: true }).click();
    await expect(page).toHaveURL("/editor/drafts/" + id + "/preview");
    await expect(page.locator("#draft-knowledge-content").getByText("Keep this unsaved statement in the editor.")).toBeVisible();
    await expect(page.getByText("Saved revision 3 · Status: editing")).toBeVisible();
});

test("draftReaderKeepsOwnerAdminAndAnonymousBoundariesAndRecoversServiceFaults", async ({ page, scene }) => {
    await scene("content");
    const { id } = await createDraft(page);
    const url = "/editor/drafts/" + id + "/preview";
    await actor(page, "content_editor_two");
    await page.goto(url);
    await expect(page.getByRole("heading", { name: "Content is not available." })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Safe preview" })).toHaveCount(0);
    await actor(page, "auth_learner");
    await page.goto(url);
    await expect(page.getByRole("heading", { name: "You do not have permission." })).toBeVisible();
    await page.context().clearCookies();
    await page.goto(url);
    await expect(page.getByRole("heading", { name: "Sign in to view your account" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Safe preview" })).toHaveCount(0);
    await actor(page, "content_admin");
    await page.goto(url);
    await expect(page.getByRole("heading", { name: "Read knowledge draft" })).toBeVisible();
    await expect(page.getByRole("button", { name: /Save draft|Submit for review|Approve/ })).toHaveCount(0);
    await page.goto("/editor/drafts/not-a-uuid/preview");
    await expect(page.getByRole("heading", { name: "Content is not available." })).toBeVisible();
    await page.goto("/editor/drafts/99999999-9999-4999-8999-999999999999/preview");
    await expect(page.getByRole("heading", { name: "Content is not available." })).toBeVisible();
    await page.goto(url);
    await scene("content-unavailable");
    await page.reload();
    await expect(page.getByRole("heading", { name: "Content management is temporarily unavailable." })).toBeVisible();
    await scene("content-recover");
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Read knowledge draft" })).toBeVisible();
    await fitsViewport(page);
});

test("emptySavedDraftHasNoInventedKnowledgeAndCanBeOpenedFromTheList", async ({ page, scene }) => {
    await scene("content");
    await actor(page, "content_editor");
    await page.goto("/editor");
    await page.getByLabel("New package ID").fill("empty-reading-draft");
    await page.getByLabel("New catalogue version").fill("1");
    await page.getByRole("button", { name: "Create draft", exact: true }).click();
    await expect(page).toHaveURL(/\/editor\/drafts\/[0-9a-f-]+$/);
    await page.goto("/editor");
    await page.getByRole("link", { name: "Read saved draft", exact: true }).click();
    await expect(page.getByText("No knowledge points in this saved draft.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Next knowledge point", exact: true })).toBeDisabled();
    await expect(page.getByRole("heading", { name: "Statement" })).toHaveCount(0);
    await fitsViewport(page);
});
