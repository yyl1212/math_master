import { readFileSync } from "node:fs";
import { it, expect, vi } from "vitest";
import { createQuestionProxy } from "./question-proxy";
import { questionInput, draftView, questionJSON, fixtureID, token, publication } from "../question/test-fixtures";
const origin = "http://127.0.0.1:8080", publicOrigin = "http://127.0.0.1:3000", options = { publicOrigin, production: false };
const headers = { Origin: publicOrigin, "Content-Type": "application/json", "X-CSRF-Token": token, "Idempotency-Key": fixtureID };
const req = (path: string, method = "GET", body?: BodyInit, extra: HeadersInit = {}) => new Request(publicOrigin + path, { method, headers: { ...headers, ...extra }, ...(body === undefined ? {} : { body }) });
it("TestQuestionProxyBoundary", async () => {
    let calls = 0, sent: RequestInit | undefined;
    const proxy = createQuestionProxy(origin, options, async (_, init) => { calls++; sent = init; return questionJSON(draftView(), 201); });
    const cases = JSON.parse(readFileSync("../api/question-boundary-cases.json", "utf8")) as {
        name: string;
        path: string;
        method: string;
        body?: string;
        bodyBase64?: string;
        status: number;
        code: string;
    }[];
    for (const c of cases) {
        const body = c.method === "GET" ? undefined : c.bodyBase64 ? Buffer.from(c.bodyBase64, "base64") : c.body;
        const response = await proxy(req(c.path, c.method, body), c.path.split("?")[0].slice("/api/v1/question-bank/".length).split("/"));
        expect(response.status, c.name).toBe(c.status);
        expect((await response.json()).error.code, c.name).toBe(c.code);
    }
    expect(calls).toBe(0);
    const raw = " \n" + JSON.stringify(questionInput(), null, 2) + " \t";
    expect((await proxy(req("/api/v1/question-bank/drafts", "POST", raw), ["drafts"])).status).toBe(201);
    expect(new TextDecoder().decode(sent?.body as ArrayBuffer)).toBe(raw);
    for (const raw of [JSON.stringify(questionInput()).replace('"version":1', '"version":1e0'), JSON.stringify(questionInput()).replace('"version":1', '"version":1.0')])
        expect((await proxy(req("/api/v1/question-bank/drafts", "POST", raw), ["drafts"])).status).toBe(400);
    expect((await proxy(req("/api/v1/question-bank/drafts", "POST", JSON.stringify(questionInput()) + " ".repeat(4194305)), ["drafts"])).status).toBe(413);
    expect((await proxy(req("/api/v1/question-bank/drafts", "HEAD"), ["drafts"])).headers.get("Allow")).toBe("GET, POST");
});
it("TestQuestionResponseIntegrity", async () => {
    const good = { items: [publication()], total: 10, limit: 1, offset: 0, head: null };
    expect((await createQuestionProxy(origin, options, async () => questionJSON(good))(req("/api/v1/question-bank/publications"), ["publications"])).status).toBe(200);
    for (const bad of [() => questionJSON({ ...good, head: undefined }), () => questionJSON({ ...good, limit: 0 }), () => questionJSON({ ...good, items: [publication(), publication()] }), () => questionJSON({ ...draftView(), credential: "must-not-leak" }), () => questionJSON(draftView(), 200, { "Set-Cookie": "foreign=private" }), () => questionJSON(good, 200, { "Cache-Control": "public" }), () => new Response("x".repeat(4194305), { headers: { "Content-Type": "application/json", "X-Request-ID": "a".repeat(32), "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" } }), () => new Response(null, { status: 302, headers: { Location: "https://untrusted.invalid" } })]) {
        const r = await createQuestionProxy(origin, options, async () => bad())(req("/api/v1/question-bank/publications"), ["publications"]);
        expect(r.status).toBe(503);
        expect(r.headers.getSetCookie()).toHaveLength(0);
        expect(await r.text()).not.toContain("must-not-leak");
    }
});
it("TestQuestionProxyCancellation", async () => {
    let sent: RequestInit | undefined, url = "";
    const proxy = createQuestionProxy(origin, options, async (input, init) => { sent = init; url = String(input); return questionJSON(draftView()); });
    expect((await proxy(req("/api/v1/question-bank/drafts/" + fixtureID, "GET", undefined, { Cookie: `foreign=secret; mm_session_dev=${token}; mm_preauth_dev=${token}`, Authorization: "secret" }), ["drafts", fixtureID])).status).toBe(200);
    expect(url).toBe(origin + "/api/v1/question-bank/drafts/" + fixtureID);
    expect(new Headers(sent?.headers).get("Cookie")).toBe(`mm_session_dev=${token}`);
    expect(new Headers(sent?.headers).has("Authorization")).toBe(false);
    expect(sent).toMatchObject({ cache: "no-store", redirect: "error" });
    vi.useFakeTimers();
    for (const stage of ["body", "fetch", "response"]) {
        let cancelled = false;
        const stream = new ReadableStream<Uint8Array>({ cancel() { cancelled = true; } });
        const fetching = async (_: RequestInfo | URL, init?: RequestInit) => stage === "fetch" ? new Promise<Response>((_, reject) => init?.signal?.addEventListener("abort", () => reject(new Error("secret")))) : new Response(stream, { headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "X-Request-ID": "a".repeat(32) } });
        const input = stage === "body" ? new Request(publicOrigin + "/api/v1/question-bank/drafts", { method: "POST", headers, body: stream, duplex: "half" } as RequestInit) : req("/api/v1/question-bank/drafts/" + fixtureID);
        const pending = createQuestionProxy(origin, options, fetching)(input, stage === "body" ? ["drafts"] : ["drafts", fixtureID]);
        await vi.advanceTimersByTimeAsync(10001);
        expect((await pending).status, stage).toBe(503);
        if (stage !== "fetch")
            expect(cancelled, stage).toBe(true);
    }
    const controller = new AbortController();
    controller.abort();
    expect((await proxy(new Request(publicOrigin + "/api/v1/question-bank/drafts/" + fixtureID, { signal: controller.signal }), ["drafts", fixtureID])).status).toBe(503);
});
it("QuestionProxyRejectsFoldedDuplicateProofHeaders", async () => {
    const proxy = createQuestionProxy(origin, options, async () => questionJSON(draftView()));
    expect((await proxy(req("/api/v1/question-bank/drafts/" + fixtureID, "GET", undefined, { "X-CSRF-Token": token + ", " + token }), ["drafts", fixtureID])).status).toBe(403);
    expect((await proxy(req("/api/v1/question-bank/drafts/" + fixtureID, "GET", undefined, { "Idempotency-Key": fixtureID + ", " + fixtureID }), ["drafts", fixtureID])).status).toBe(400);
});
