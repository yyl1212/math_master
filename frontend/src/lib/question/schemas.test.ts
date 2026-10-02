import { it, expect } from "vitest";
import { questionRouteRequest, readQuestionResponse, questionInputError } from "./schemas";
import { questionJSON, publication, fixtureID, otherID, draftView, sha, gate } from "./test-fixtures";
it("TestQuestionResponseIntegrity", async () => {
    expect(await readQuestionResponse(questionJSON({ items: [], total: 0, limit: 1, offset: 0, head: null }), "listPublications")).toMatchObject({ ok: true, data: { head: null, limit: 1 } });
    expect(await readQuestionResponse(questionJSON(draftView()), "readDraft")).toMatchObject({ ok: true, data: { gate: { digest: "" } } });
    const d = draftView();
    const frozen = { catalogueVersion: 1, catalogueSha256: sha, questionPackage: d.questionPackage, sourceMap: [], authorIds: [otherID], legacyUnattributed: false, resolved: [], objectives: [], generation: [], instanceIdentities: [], coverage: [], generatorVersions: [], verifierVersions: [], frozenDigest: sha };
    const sub = { id: fixtureID, workspaceId: otherID, ownerId: otherID, status: "pending", createdAt: d.createdAt, revision: 1, frozen, gate: gate(), review: null };
    expect(await readQuestionResponse(questionJSON(sub), "readSubmission")).toMatchObject({ ok: true, data: { review: null } });
    for (const bad of [{ ...publication(), status: "draft" }, { ...publication(), baseKnowledgeHead: undefined }])
        expect(await readQuestionResponse(questionJSON(bad), "readPublication")).toMatchObject({ ok: false, code: "SERVICE_UNAVAILABLE" });
    const error = { error: { code: "RATE_LIMITED", message: "Too many requests. Try again later.", requestId: "a".repeat(32) } };
    expect(await readQuestionResponse(questionJSON(error, 429, { "Retry-After": "60" }), "readDraft")).toMatchObject({ ok: false, status: 429, retryAfter: 60 });
    expect(await readQuestionResponse(questionJSON(error, 403, { "Retry-After": "60" }), "readDraft")).toMatchObject({ ok: false, status: 503 });
});
it("QuestionRoutesAreClosedAndQueriesAreActionSpecific", () => {
    expect(questionRouteRequest({ kind: "listInstances", id: fixtureID, query: { limit: 1, offset: 2 } })).toEqual({ method: "GET", path: "/api/v1/question-bank/submissions/" + fixtureID + "/instances?limit=1&offset=2" });
    for (const route of [{ kind: "readDraft", id: fixtureID, url: "https://untrusted.invalid" }, { kind: "readCoverage", query: { scope: "all" } }, { kind: "listPublications", query: { status: "draft" } }, { kind: "listMembers", id: fixtureID, query: { status: "published" } }])
        expect(questionRouteRequest(route as never)).toBeNull();
    expect(questionInputError("prepareRelease", { submissionIds: [fixtureID], expectedKnowledgeHead: null, expectedQuestionHead: null, reason: "Prepare a fully reviewed question bank." })).toBeNull();
    expect(questionInputError("prepareRelease", { submissionIds: [fixtureID], expectedQuestionHead: null, reason: "Prepare a fully reviewed question bank." })).toBe("INVALID_REQUEST");
});
it("FailedGenerationReportsRemainReadable", async () => {
    const report = { ...gate(), completenessErrors: [{ code: "TEMPLATE_NOT_READY", path: "/templates/0", message: "Question validation failed." }], generation: [{ template: { id: "rational-addition", version: 1, sha256: "" }, rawCombinations: 10, excludedCombinations: 0, validInstances: 0, generatorVersion: 1, verifierVersion: 1, constraintCounts: [] }] };
    expect(await readQuestionResponse(questionJSON(report), "validateDraft")).toMatchObject({ ok: true, data: { readyToSubmit: false, generation: [{ validInstances: 0 }] } });
});
it("ControlRequestLimitsCountTransmittedBytes", () => {
    const input = { decision: "approve", checks: { mathematics: true, explanations: true, objectives: true, sources: true, illustrations: true, generation: true }, independenceNote: "<".repeat(1000), generationNote: "<".repeat(1000), note: "<".repeat(1000) };
    expect(questionInputError("decideReview", input)).toBeNull();
});
