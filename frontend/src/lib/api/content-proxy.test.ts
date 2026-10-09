import { createHash } from "node:crypto";
import { it, expect } from "vitest";
import { createContentProxy } from "./content-proxy";
import { draftInput, draftView, publicationView, fixtureID, otherID, fixtureSVG, contentJSON, requestID, token } from "../content/test-fixtures";
const origin = "http://127.0.0.1:8080", publicOrigin = "http://127.0.0.1:3000", options = { publicOrigin, production: false };
const headers = { "Origin": publicOrigin, "Content-Type": "application/json", "X-CSRF-Token": token, "Idempotency-Key": fixtureID };
const req = (path: string, method = "GET", body?: string, extra: HeadersInit = {}) => new Request(publicOrigin + "/api/v1/content/" + path, { method, headers: { ...headers, ...extra }, ...(body === undefined ? {} : { body }) });
it("TestContentProxyManagedRetirementBoundary", async () => {
    const payload = { error: { code: "KNOWLEDGE_WORKFLOW_RETIRED", message: "Use current knowledge management.", requestId: requestID } };
    const proxy = createContentProxy(origin, options, async () => contentJSON(payload, 410));
    const response = await proxy(req("drafts/" + fixtureID), ["drafts", fixtureID]);
    expect(response.status).toBe(410);
    expect(await response.json()).toEqual(payload);
    expect(response.headers.get("Cache-Control")).toBe("private, no-store");
    for (const invalid of [
        { ...payload, privateBody: "must-not-leak" },
        { error: { ...payload.error, privateBody: "must-not-leak" } },
        { error: { ...payload.error, message: "private reason must-not-leak" } },
        { error: { ...payload.error, requestId: otherID } },
        { error: { ...payload.error, code: "NOT_FOUND" } },
    ]) {
        const bad = await createContentProxy(origin, options, async () => contentJSON(invalid, 410))(req("drafts/" + fixtureID), ["drafts", fixtureID]);
        expect(bad.status).toBe(503);
        expect(await bad.text()).not.toContain("must-not-leak");
    }
    const invalidHeaders: HeadersInit[] = [{ "Set-Cookie": "secret=must-not-leak" }, { "Retry-After": "1" }, { "Content-Type": "text/html" }];
    for (const extra of invalidHeaders) {
        const bad = await createContentProxy(origin, options, async () => contentJSON(payload, 410, extra))(req("drafts/" + fixtureID), ["drafts", fixtureID]);
        expect(bad.status).toBe(503);
        expect(bad.headers.getSetCookie()).toHaveLength(0);
    }
});
it("TestContentProxyRejectedRetirementCancelsUpstream", async () => {
    const invalidHeaders: HeadersInit[] = [{ "Content-Type": "text/html" }, { "Retry-After": "1" }, { "Set-Cookie": "private=hidden" }, { "X-Request-ID": "bad" }];
    for (const extra of invalidHeaders) {
        let cancelled = 0;
        const stream = new ReadableStream<Uint8Array>({ cancel() { cancelled++; } });
        const response = new Response(stream, { status: 410, headers: { "Content-Type": "application/json", "X-Request-ID": requestID, ...Object.fromEntries(new Headers(extra)) } });
        const result = await createContentProxy(origin, options, async () => response)(req("drafts/" + fixtureID), ["drafts", fixtureID]);
        expect(result.status).toBe(503);
        expect(cancelled).toBe(1);
    }
});
it("TestContentProxyRawRequestBoundary", async () => {
    let calls = 0;
    const proxy = createContentProxy(origin, options, async () => { calls++; return contentJSON(draftView(), 201); });
    const raw = JSON.stringify(draftInput());
    for (const input of [raw.slice(0, -1) + ',"catalogueVersion":1}', raw.slice(0, -1) + ',"CatalogueVersion":1}', raw.replace('workflow-ready', '\\ud800'), raw.replace('workflow-ready', '\\u0000')])
        expect((await proxy(req("drafts", "POST", input), ["drafts"])).status).toBe(400);
    expect(calls).toBe(0);
    expect((await proxy(req("drafts", "POST", raw + " ".repeat((8 << 20) + 1 - raw.length)), ["drafts"])).status).toBe(413);
    expect(calls).toBe(0);
    const prepare = createContentProxy(origin, options, async () => contentJSON(publicationView(), 201));
    expect((await prepare(req("publications/prepare", "POST", JSON.stringify({ submissionIds: [fixtureID], reason: "A technical reason." })), ["publications", "prepare"])).status).toBe(400);
    expect((await prepare(req("publications/prepare", "POST", JSON.stringify({ submissionIds: [fixtureID], expectedHead: null, reason: "A technical reason." })), ["publications", "prepare"])).status).toBe(201);
    for (const [path, segments, status] of [["other", ["other"], 404], ["drafts?unknown=1", ["drafts"], 400], ["drafts?limit=1&limit=2", ["drafts"], 400], ["drafts/", ["drafts", ""], 404], ["drafts/%31" + fixtureID.slice(1), ["drafts", fixtureID], 400]] as const) {
        expect((await proxy(req(path), [...segments])).status).toBe(status);
    }
    expect((await proxy(req("drafts", "HEAD"), ["drafts"])).status).toBe(405);
});
it("TestContentProxyResponseBoundary", async () => {
    const good = () => contentJSON({ items: [], total: 0, limit: 20, offset: 0 });
    const bad = [() => contentJSON({ ...draftView(), credential: "must-not-leak" }), () => new Response('x'.repeat((4 << 20) + 1), { headers: { "Content-Type": "application/json", "X-Request-ID": requestID } }), () => new Response('private body', { headers: { "Content-Type": "text/html" } }), () => contentJSON(draftView(), 200, { "Set-Cookie": "foreign=private" }), () => contentJSON(draftView(), 200, { "X-Request-ID": "bad-id" }), () => contentJSON({ error: { code: "RATE_LIMITED", message: "Too many requests. Try again later.", requestId: requestID } }, 429, { "Retry-After": "-1" }), () => contentJSON({ error: { code: "NOT_FOUND", message: "secret internal detail", requestId: requestID } }, 404), () => new Response(null, { status: 302, headers: { Location: "https://untrusted.example" } })];
    for (const factory of bad) {
        const response = await createContentProxy(origin, options, async () => factory())(req("drafts/" + fixtureID), ["drafts", fixtureID]);
        expect(response.status).toBe(503);
        expect(response.headers.getSetCookie()).toHaveLength(0);
        expect(await response.text()).not.toContain("must-not-leak");
    }
    expect((await createContentProxy(origin, options, async () => good())(req("drafts"), ["drafts"])).status).toBe(200);
    const error = await createContentProxy(origin, options, async () => contentJSON({ error: { code: "FORBIDDEN", message: "You do not have permission.", requestId: requestID } }, 403))(req("drafts"), ["drafts"]);
    expect(error.status).toBe(403);
});
it("TestContentSVGScopeAndBoundary", async () => {
    const sha = createHash("sha256").update(fixtureSVG).digest("hex");
    let sent: RequestInit | undefined, url = "";
    const svg = (bytes = fixtureSVG, extra: HeadersInit = {}) => new Response(bytes, { headers: { "Content-Type": "image/svg+xml", "X-Request-ID": requestID, "Content-Security-Policy": "sandbox; default-src 'none'", "X-Content-Type-Options": "nosniff", "Cache-Control": "private, no-store", ...extra } });
    const proxy = createContentProxy(origin, options, async (input, init) => { url = String(input); sent = init; return svg(); });
    for (const scope of ["drafts", "submissions"]) {
        const path = scope + "/" + fixtureID + "/assets/" + sha;
        const response = await proxy(req(path, "GET", undefined, { Cookie: `foreign=private; mm_session_dev=${token}; mm_preauth_dev=${token}`, Authorization: "private" }), [scope, fixtureID, "assets", sha]);
        expect(response.status).toBe(200);
        expect(new Uint8Array(await response.arrayBuffer())).toEqual(fixtureSVG);
        expect(url).toBe(origin + "/api/v1/content/" + path);
        expect(new Headers(sent?.headers).get("Cookie")).toBe(`mm_session_dev=${token}`);
        expect(new Headers(sent?.headers).has("Authorization")).toBe(false);
        expect(sent).toMatchObject({ cache: "no-store", redirect: "error" });
    }
    for (const response of [svg(new Uint8Array((1 << 20) + 1)), svg(new TextEncoder().encode('bad SVG')), svg(fixtureSVG, { "Set-Cookie": "foreign=private" }), new Response(null, { status: 302 })])
        expect((await createContentProxy(origin, options, async () => response)(req("drafts/" + fixtureID + "/assets/" + sha), ["drafts", fixtureID, "assets", sha])).status).toBe(503);
    const unsafe = new TextEncoder().encode('<svg xmlns="http://www.w3.org/2000/svg"><script>unsafe()</script></svg>');
    const unsafeSHA = createHash("sha256").update(unsafe).digest("hex");
    expect((await createContentProxy(origin, options, async () => svg(unsafe))(req("drafts/" + otherID + "/assets/" + unsafeSHA), ["drafts", otherID, "assets", unsafeSHA])).status).toBe(503);
});
it("TestContentProxyLimitClassification", async () => {
    let calls = 0;
    const proxy = createContentProxy(origin, options, async () => { calls++; return contentJSON(draftView(), 201); });
    const raw = JSON.stringify(draftInput());
    expect((await proxy(req("drafts", "POST", raw, { "Content-Length": "wrong" }), ["drafts"])).status).toBe(400);
    const input = draftInput();
    input.package.knowledge[0].statement = "<".repeat(360000);
    const response = await proxy(req("drafts", "POST", JSON.stringify(input)), ["drafts"]);
    expect(response.status).toBe(422);
    expect((await response.json()).error.code).toBe("CONTENT_LIMIT_EXCEEDED");
    expect(calls).toBe(0);
});
