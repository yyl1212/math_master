import { it, expect, vi } from "vitest";
import { requestLearning, bindLearningInput } from "./client";
import { learningJSON, practice, fixtureID, token } from "./test-fixtures";
vi.mock("../auth/client", () => ({ getAuthContext: vi.fn() }));
import { getAuthContext } from "../auth/client";
it("manual retry preserves exact bytes and key with no automatic writes", async () => {
    vi.mocked(getAuthContext).mockResolvedValue({ ok: true, data: { user: { id: fixtureID, username: "learner", roles: ["learner"], mustChangePassword: false }, csrfToken: token } });
    const p = practice();
    const seen: {
        body: unknown;
        key: string | null;
    }[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (_, init) => {
        seen.push({ body: init?.body, key: new Headers(init?.headers).get("Idempotency-Key") });
        if (seen.length === 1)
            throw new Error("lost");
        return learningJSON({ ...p, summary: { ...p.summary, state: "answered", submittedAt: "2026-10-02T00:00:00Z" }, result: { outcome: "answered", item: { question: p.question, answer: { kind: "numeric", raw: " 2 / 3 " }, correct: false, correctChoiceId: null, correctNumeric: { numerator: "2", denominator: "1" }, explanation: "Original answer.", validity: "effective", reasons: [] } } });
    });
    const input = Object.freeze({ kind: "numeric", raw: " 2 / 3 " });
    const route = { kind: "answerPractice", id: fixtureID } as const;
    expect((await requestLearning(route, input, fixtureID)).ok).toBe(false);
    expect(seen).toHaveLength(1);
    expect((await requestLearning(route, input, fixtureID)).ok).toBe(true);
    expect(seen[0]).toEqual(seen[1]);
    expect(seen[0].body).toContain('" 2 / 3 "');
});
it("read has no auth context request and wrong keys are not sent", async () => { const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(learningJSON(practice())); expect((await requestLearning({ kind: "readPractice", id: fixtureID })).ok).toBe(true); expect(spy).toHaveBeenCalledTimes(1); expect((await requestLearning({ kind: "answerPractice", id: fixtureID }, { kind: "numeric", raw: "2" }, "bad")).ok).toBe(false); expect(spy).toHaveBeenCalledTimes(1); });
it("a frozen pending input cannot be sent from a different account", async () => { const input = Object.freeze({ kind: "numeric", raw: "2" }); bindLearningInput(input, "22222222-2222-4222-8222-222222222222"); vi.mocked(getAuthContext).mockResolvedValue({ ok: true, data: { user: { id: fixtureID, username: "learner", roles: ["learner"], mustChangePassword: false }, csrfToken: token } }); const spy = vi.spyOn(globalThis, "fetch"); expect(await requestLearning({ kind: "answerPractice", id: fixtureID }, input, fixtureID)).toMatchObject({ ok: false, code: "FORBIDDEN" }); expect(spy).not.toHaveBeenCalled(); });
it("the total client deadline also covers an unresolved auth context", async () => { vi.useFakeTimers(); vi.mocked(getAuthContext).mockReturnValue(new Promise(() => { })); const fetcher = vi.spyOn(globalThis, "fetch"); const pending = requestLearning({ kind: "answerPractice", id: fixtureID }, { kind: "numeric", raw: "2" }, fixtureID); await vi.advanceTimersByTimeAsync(10001); expect(await pending).toMatchObject({ ok: false, code: "SERVICE_UNAVAILABLE" }); expect(fetcher).not.toHaveBeenCalled(); });
