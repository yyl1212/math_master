import { it, expect, vi } from "vitest";
import { contentRequest } from "./client";
import { draftInput, draftView, fixtureID, otherID, contentJSON, token } from "./test-fixtures";
const context = () => new Response(JSON.stringify({ data: { user: { id: otherID, username: "content_editor", roles: ["learner", "editor"], mustChangePassword: false }, csrfToken: token } }), { headers: { "Content-Type": "application/json" } });
it("TestContentClient", async () => {
    const calls: {
        url: string;
        init?: RequestInit;
    }[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { calls.push({ url: String(input), init }); return String(input).endsWith("/auth/context") ? context() : contentJSON(draftView(), 201); });
    const result = await contentRequest({ kind: "createDraft" }, draftInput(), fixtureID);
    expect(result.ok).toBe(true);
    expect(calls).toHaveLength(2);
    expect(calls[0].url).toBe("/api/v1/auth/context");
    expect(calls[1].init).toMatchObject({ method: "POST", credentials: "same-origin", cache: "no-store", redirect: "error" });
    const headers = new Headers(calls[1].init?.headers);
    expect(headers.get("Idempotency-Key")).toBe(fixtureID);
    expect(headers.get("X-CSRF-Token")).toBe(token);
    expect(JSON.parse(String(calls[1].init?.body))).toEqual(draftInput());
    expect(JSON.stringify(result)).not.toContain(token);
    calls.length = 0;
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { calls.push({ url: String(input), init }); return contentJSON(draftView()); });
    expect((await contentRequest({ kind: "readDraft", id: fixtureID })).ok).toBe(true);
    expect(calls).toHaveLength(1);
    expect(calls[0].url).toBe("/api/v1/content/drafts/" + fixtureID);
});
it("TestContentClientTimeoutDoesNotRetry", async () => {
    vi.useFakeTimers();
    let mutations = 0;
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { if (String(input).endsWith("/auth/context"))
        return context(); mutations++; return new Promise<Response>((_, reject) => init?.signal?.addEventListener("abort", () => reject(new Error("private timeout details")))); });
    const result = contentRequest({ kind: "createDraft" }, draftInput(), fixtureID);
    await vi.advanceTimersByTimeAsync(10001);
    expect(await result).toMatchObject({ ok: false, status: 503, code: "SERVICE_UNAVAILABLE" });
    expect(mutations).toBe(1);
});
