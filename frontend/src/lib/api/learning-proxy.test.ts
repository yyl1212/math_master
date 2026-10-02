import { it, expect, vi } from "vitest";
import { createLearningProxy } from "./learning-proxy";
import { learningJSON, practice, fixtureID, token, overview, identity } from "../learning/test-fixtures";
import { fixtureSVG } from "../content/test-fixtures";
import { createHash } from "node:crypto";
const publicOrigin = "http://127.0.0.1:3000", origin = "http://127.0.0.1:8080", config = { publicOrigin, production: false };
const headers = { Origin: publicOrigin, "Content-Type": "application/json", "X-CSRF-Token": token, "Idempotency-Key": fixtureID };
const req = (path: string, method = "GET", body?: BodyInit, extra: HeadersInit = {}) => new Request(publicOrigin + "/api/v1/learning/" + path, { method, headers: { ...headers, ...extra }, ...(body === undefined ? {} : { body }) });
it("strict proxy preserves original bytes, proof and per-user cookie", async () => { let sent: RequestInit | undefined, url = ""; const p = practice(); const proxy = createLearningProxy(origin, config, async (u, i) => { url = String(u); sent = i; return learningJSON(p, 201); }); const input = { knowledge: p.summary.knowledge, expectedKnowledgeHead: fixtureID, expectedQuestionHead: fixtureID }; const raw = " \n" + JSON.stringify(input, null, 2) + " \t"; expect((await proxy(req("practice", "POST", raw, { Cookie: `other=private; mm_session_dev=${token}; mm_preauth_dev=${token}` }))).status).toBe(201); expect(new TextDecoder().decode(sent?.body as ArrayBuffer)).toBe(raw); expect(url).toBe(origin + "/api/v1/learning/practice"); expect(sent).toMatchObject({ cache: "no-store", redirect: "error" }); expect(new Headers(sent?.headers).get("Cookie")).toBe(`mm_session_dev=${token}`); expect(new Headers(sent?.headers).get("Idempotency-Key")).toBe(fixtureID); });
it("invalid routes, queries, headers and raw JSON never reach Go", async () => {
    const fetcher = vi.fn(async () => learningJSON(practice()));
    const proxy = createLearningProxy(origin, config, fetcher);
    for (const r of [req("overview?owner=x"), req("knowledge/math-root?version=1.0"), req("knowledge/math-root?version=%31"), req("history?limit=1&limit=1"), req("paths/no-uuid"), req("practice", "GET"), req("practice", "POST", '{"knowledge":null}'), req("practice/" + fixtureID + "/reveal", "POST", "{} ".repeat(2731)), req("practice/" + fixtureID + "/reveal", "POST", "{}", { "X-CSRF-Token": token + ", " + token }), req("practice/" + fixtureID + "/reveal", "POST", "{}", { "Idempotency-Key": fixtureID + ", " + fixtureID })])
        expect((await proxy(r)).status).toBeGreaterThanOrEqual(400);
    expect(fetcher).not.toHaveBeenCalled();
});
it("malformed and overlimit private JSON is sanitized", async () => {
    for (const response of [learningJSON({ ...practice(), correctNumeric: "PRIVATE" }), learningJSON(practice(), 200, { "Set-Cookie": "foreign=secret" }), new Response("x".repeat(4194305), { headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "X-Request-ID": "a".repeat(32) } })]) {
        const r = await createLearningProxy(origin, config, async () => response)(req("practice/" + fixtureID));
        expect(r.status).toBe(503);
        expect(await r.text()).not.toContain("PRIVATE");
        expect(r.headers.getSetCookie()).toHaveLength(0);
    }
});
it("owned SVG retains byte bound, digest and security headers", async () => { const sha = createHash("sha256").update(fixtureSVG).digest("hex"); const good = () => new Response(fixtureSVG.slice().buffer as ArrayBuffer, { headers: { "Content-Type": "image/svg+xml", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "X-Request-ID": "a".repeat(32), "Content-Security-Policy": "sandbox; default-src 'none'" } }); const r = await createLearningProxy(origin, config, async () => good())(req("assets/" + fixtureID + "/" + sha)); expect(r.status).toBe(200); expect(r.headers.get("Content-Security-Policy")).toBe("sandbox; default-src 'none'"); expect(new Uint8Array(await r.arrayBuffer())).toEqual(fixtureSVG); const bad = await createLearningProxy(origin, config, async () => good())(req("assets/" + fixtureID + "/" + "b".repeat(64))); expect(bad.status).toBe(503); });
it("ten seconds includes delayed response reading and cancellation", async () => { vi.useFakeTimers(); let cancelled = false; const stream = new ReadableStream<Uint8Array>({ cancel() { cancelled = true; } }); const pending = createLearningProxy(origin, config, async () => new Response(stream, { headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "X-Request-ID": "a".repeat(32) } }))(req("practice/" + fixtureID)); await vi.advanceTimersByTimeAsync(10001); expect((await pending).status).toBe(503); expect(cancelled).toBe(true); });
it("complete JSON and original command boundaries include exact bytes", async () => {
    for (const n of [8191, 8192, 8193]) {
        const abandoned = practice();
        abandoned.summary = { ...abandoned.summary, kind: "practice", mode: null, state: "abandoned" };
        const r = await createLearningProxy(origin, config, async () => learningJSON(abandoned))(req("practice/" + fixtureID + "/abandon", "POST", "{}" + " ".repeat(n - 2)));
        expect(r.status).toBe(n <= 8192 ? 200 : 413);
    }
    ;
    for (const n of [4194303, 4194304, 4194305]) {
        const v = overview();
        v.availablePaths = [{ path: identity("route"), title: "", titleZh: "", totalNodes: 3 }];
        const base = JSON.stringify(v).length;
        v.availablePaths[0].title = "x".repeat(n - base);
        const response = new Response(JSON.stringify(v), { headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "X-Request-ID": "a".repeat(32) } });
        expect(new TextEncoder().encode(JSON.stringify(v)).byteLength).toBe(n);
        const r = await createLearningProxy(origin, config, async () => response)(req("overview"));
        expect(r.status).toBe(n <= 4194304 ? 200 : 503);
        expect((await r.arrayBuffer()).byteLength).toBeLessThanOrEqual(4194304);
    }
    ;
    const fetcher = vi.fn(async () => learningJSON(practice()));
    const r = await createLearningProxy(origin, config, fetcher)(req("practice/" + fixtureID + "/abandon", "POST", "{}", { "Content-Length": "3" }));
    expect(r.status).toBe(400);
    expect(fetcher).not.toHaveBeenCalled();
});
it("SVG limit uses actual valid bytes at 1048576 plus or minus one", async () => { const prefix = '<svg xmlns="http://www.w3.org/2000/svg"><title>', suffix = '</title></svg>'; for (const n of [1048575, 1048576, 1048577]) {
    const bytes = new TextEncoder().encode(prefix + "x".repeat(n - prefix.length - suffix.length) + suffix), sha = createHash("sha256").update(bytes).digest("hex");
    const upstream = new Response(bytes.buffer as ArrayBuffer, { headers: { "Content-Type": "image/svg+xml", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "X-Request-ID": "a".repeat(32), "Content-Security-Policy": "sandbox; default-src 'none'" } });
    const r = await createLearningProxy(origin, config, async () => upstream)(req("assets/" + fixtureID + "/" + sha));
    expect(r.status).toBe(n <= 1048576 ? 200 : 503);
} });
