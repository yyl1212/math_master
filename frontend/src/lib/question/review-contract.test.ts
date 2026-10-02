import { it, expect, vi, afterEach } from "vitest";
import { questionInputError, readQuestionResponse } from "./schemas";
import { readServerQuestion } from "./server-client";
import { createQuestionProxy } from "../api/question-proxy";
import { componentSubmission, componentDraft, fixtureID, otherID } from "@/features/question/test-fixtures";
import { questionJSON, token } from "./test-fixtures";
afterEach(() => { vi.unstubAllEnvs(); vi.restoreAllMocks(); });
it("ApprovedTemplateWithEmptyGenerationNoteRemainsReadable", async () => {
 const sub = componentSubmission(); sub.frozen.questionPackage.templates = componentDraft().questionPackage.templates;
 const input = { decision: "approve" as const, checks: { mathematics: true, explanations: true, objectives: true, sources: true, illustrations: true, generation: true }, independenceNote: "Independently reviewed every generated question.", generationNote: "", note: "All six checks and the full finite range were reviewed." };
 sub.status = "approved"; sub.review = { ...input, id: otherID, submissionId: sub.id, reviewerId: fixtureID, frozenDigest: sub.frozen.frozenDigest, createdAt: sub.createdAt };
 // Reviewer must be different from the frozen author and owner.
 expect(questionInputError("decideReview", input)).toBeNull();
 expect(await readQuestionResponse(questionJSON(sub), "readSubmission")).toMatchObject({ ok: true, data: { review: { generationNote: "" } } });
 vi.stubEnv("GO_API_INTERNAL_URL", "http://127.0.0.1:8080"); vi.stubEnv("AUTH_PUBLIC_ORIGIN", "http://127.0.0.1:3000"); vi.stubEnv("APP_ENV", "development");
 vi.spyOn(globalThis, "fetch").mockImplementation(async () => questionJSON(sub));
 expect(await readServerQuestion({ kind: "readSubmission", id: sub.id }, `mm_session_dev=${token}`)).toMatchObject({ ok: true });
 const proxy = createQuestionProxy("http://127.0.0.1:8080", { publicOrigin: "http://127.0.0.1:3000", production: false }, async () => questionJSON(sub));
 expect((await proxy(new Request(`http://127.0.0.1:3000/api/v1/question-bank/submissions/${sub.id}`), ["submissions", sub.id])).status).toBe(200);
 sub.frozen.questionPackage.templates = [];
 expect(await readQuestionResponse(questionJSON(sub), "readSubmission")).toMatchObject({ ok: false, status: 503 });
 sub.review.generationNote = "Only fixed questions are present; generation does not apply.";
 expect(await readQuestionResponse(questionJSON(sub), "readSubmission")).toMatchObject({ ok: true });
});
