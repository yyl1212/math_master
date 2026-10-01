import { it, expect, vi } from "vitest";
import { requestQuestion } from "./client";
import { questionInput, draftView, questionJSON, fixtureID, otherID, token } from "./test-fixtures";
const context = () => new Response(JSON.stringify({ data: { user: { id: otherID, username: "question_editor", roles: ["learner", "editor"], mustChangePassword: false }, csrfToken: token } }), { headers: { "Content-Type": "application/json" } });
it("TestQuestionClientNoAutomaticRetry", async () => {
    vi.useFakeTimers();
    const keys: string[] = [];
    const bodies: string[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { if (String(input).endsWith("/auth/context"))
        return context(); keys.push(new Headers(init?.headers).get("Idempotency-Key")!); bodies.push(String(init?.body)); return new Promise<Response>((_, reject) => init?.signal?.addEventListener("abort", () => reject(new Error("secret timeout")))); });
    const first = requestQuestion({ kind: "createDraft" }, questionInput(), fixtureID);
    await vi.advanceTimersByTimeAsync(10001);
    expect(await first).toMatchObject({ ok: false, status: 503, code: "SERVICE_UNAVAILABLE" });
    expect(keys).toEqual([fixtureID]);
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { if (String(input).endsWith("/auth/context"))
        return context(); keys.push(new Headers(init?.headers).get("Idempotency-Key")!); bodies.push(String(init?.body)); return questionJSON(draftView(), 201); });
    expect((await requestQuestion({ kind: "createDraft" }, questionInput(), fixtureID)).ok).toBe(true);
    expect(keys).toEqual([fixtureID, fixtureID]);
    expect(bodies[0]).toBe(bodies[1]);
});
it("TestQuestionClientTotalDeadlineAndAbort", async () => {
    vi.useFakeTimers();
    const calls: string[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { calls.push(String(input)); return new Promise<Response>((_, reject) => init?.signal?.addEventListener("abort", () => reject(new Error("timeout")))); });
    const controller = new AbortController();
    const pending = requestQuestion({ kind: "createDraft" }, questionInput(), fixtureID, controller.signal);
    await vi.advanceTimersByTimeAsync(1000);
    controller.abort();
    await vi.advanceTimersByTimeAsync(1);
    expect(await pending).toMatchObject({ ok: false, code: "SERVICE_UNAVAILABLE" });
    expect(calls).toEqual(["/api/v1/auth/context"]);
    await vi.advanceTimersByTimeAsync(5000);
});
