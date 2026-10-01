import { it, expect, vi } from "vitest";
import { GET as authRouteGET } from "../../app/api/v1/auth/[...segments]/route";
import { createPrivateProxy } from "./private-proxy";
const origin = "http://127.0.0.1:8080", publicOrigin = "http://127.0.0.1:3000", options = { publicOrigin, production: false };
const token = "A".repeat(43), user = { id: "10000000-0000-4000-8000-000000000001", username: "test_user", roles: ["learner"], mustChangePassword: false };
const setSession = `mm_session_dev=${token}; Path=/; Max-Age=604800; HttpOnly; SameSite=Lax`, clearPreauth = "mm_preauth_dev=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax";
const json = (body: unknown, status = 200, cookies: string[] = []) => {
    const headers = new Headers({ "Content-Type": "application/json" });
    for (const c of cookies)
        headers.append("Set-Cookie", c);
    return new Response(JSON.stringify(body), { status, headers });
};
const req = (method = "GET", path = "auth/context", body?: string, headers?: HeadersInit) => new Request(publicOrigin + "/api/v1/" + path, { method, ...(body === undefined ? {} : { body }), headers });
it("TestPrivateProxyRequestBoundary", async () => {
    let calls = 0, received: RequestInit | undefined;
    const proxy = createPrivateProxy(origin, options, async (_input, init) => { calls++; received = init; return json({ data: { user, csrfToken: token } }); });
    for (const [method, path, segments, want] of [["GET", "auth/unknown", ["auth", "unknown"], 404], ["HEAD", "auth/context", ["auth", "context"], 405], ["GET", "auth/context?unknown=1", ["auth", "context"], 400], ["GET", "admin/users?limit=0", ["admin", "users"], 400]] as const) {
        expect((await proxy(req(method, path), [...segments])).status).toBe(want);
    }
    expect(calls).toBe(0);
    const cookie = `foreign=excluded; mm_session_dev=${token}; mm_session_dev=${token}; mm_preauth_dev=${token}`;
    const r = await proxy(req("GET", "auth/context", undefined, { Cookie: cookie, Authorization: "secret-test-auth", "X-Forwarded-For": "untrusted", "X-Request-ID": "user-supplied", "X-Requested-With": "MathMaster" }), ["auth", "context"]);
    expect(r.status).toBe(200);
    const h = new Headers(received?.headers);
    expect(h.get("Cookie")).toBe(`mm_session_dev=${token}; mm_session_dev=${token}; mm_preauth_dev=${token}`);
    expect(h.has("Authorization") || h.has("X-Forwarded-For") || h.has("X-Request-ID")).toBe(false);
    expect(received).toMatchObject({ redirect: "error", cache: "no-store" });
    const before = calls;
    expect((await proxy(req("POST", "auth/login", "x".repeat(8193), { Origin: publicOrigin, "X-CSRF-Token": token, "Content-Type": "application/json" }), ["auth", "login"])).status).toBe(400);
    expect(calls).toBe(before);
});
it("TestPrivateProxyResponseBoundary", async () => {
    for (const cookies of [[setSession, clearPreauth], [setSession, clearPreauth + "; Expires=Thu, 01 Jan 1970 00:00:01 GMT"]]) {
        const r = await createPrivateProxy(origin, options, async () => json({ data: { user } }, 200, cookies))(req("POST", "auth/login", "{}", { Origin: publicOrigin, "Content-Type": "application/json", "X-CSRF-Token": token }), ["auth", "login"]);
        expect(r.status).toBe(200);
        expect(r.headers.getSetCookie()).toHaveLength(2);
    }
    const badCookies = ["foreign=value; Path=/; HttpOnly; SameSite=Lax", setSession + "; Domain=example.com", setSession.replace("HttpOnly; ", ""), setSession.replace("SameSite=Lax", "SameSite=None"), setSession + ", " + clearPreauth];
    const badResponses = badCookies.map(cookie => () => json({ data: { user } }, 200, [cookie]));
    badResponses.push(() => json({ data: { user } }, 200, [setSession, setSession]), () => json({ data: { user: { ...user, password_phc: "private-field" } } }, 200, [setSession]), () => json({ error: { code: "INVALID_REQUEST", message: "Invalid request.", requestId: "a".repeat(32) } }, 404, [clearPreauth]), () => json({ error: { code: "NOT_FOUND", message: "private internal error", requestId: "a".repeat(32) } }, 404, [clearPreauth]), () => new Response(null, { status: 302, headers: { "Set-Cookie": setSession, Location: "https://untrusted.invalid" } }));
    for (const factory of badResponses) {
        const r = await createPrivateProxy(origin, options, async () => factory())(req("POST", "auth/login", "{}", { Origin: publicOrigin, "Content-Type": "application/json", "X-CSRF-Token": token }), ["auth", "login"]);
        expect(r.status).toBe(503);
        expect(r.headers.getSetCookie()).toHaveLength(0);
    }
    let cancelled = false;
    const stream = new ReadableStream<Uint8Array>({ start(c) { c.enqueue(new Uint8Array(1024 * 1024 + 1)); }, cancel() { cancelled = true; } });
    const tooBig = new Response(stream, { headers: { "Content-Type": "application/json", "Set-Cookie": setSession } });
    const r = await createPrivateProxy(origin, options, async () => tooBig)(req(), ["auth", "context"]);
    expect(r.status).toBe(503);
    expect(cancelled).toBe(true);
    expect(r.headers.getSetCookie()).toHaveLength(0);
    const invalid204 = new Response(null, { status: 204, headers: { "Content-Length": "1", "Set-Cookie": setSession } });
    expect((await createPrivateProxy(origin, options, async () => invalid204)(req("POST", "auth/logout", "{}", { Origin: publicOrigin, "Content-Type": "application/json", "X-CSRF-Token": token }), ["auth", "logout"])).status).toBe(503);
});
it("TestPrivateRouteBeforeConfiguration", async () => {
    vi.stubEnv("AUTH_PUBLIC_ORIGIN", "");
    try {
        const response = await authRouteGET(req("GET", "auth/unknown"), { params: Promise.resolve({ segments: ["unknown"] }) });
        expect(response.status).toBe(404);
        expect(response.headers.get("Cache-Control")).toBe("private, no-store");
    }
    finally {
        vi.unstubAllEnvs();
    }
});
it("TestPrivatePrefixRoots", async () => {
    for (const route of [await import("../../app/api/v1/auth/route"), await import("../../app/api/v1/admin/route")]) {
        const response = route.GET();
        expect(response.status).toBe(404);
        expect(response.headers.get("Cache-Control")).toBe("private, no-store");
    }
});
it("TestPrivateProxyContextIdentityTransition", async () => { const clearSession = "mm_session_dev=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax"; const response = await createPrivateProxy(origin, options, async () => json({ data: { user, csrfToken: token } }, 200, [clearSession]))(req(), ["auth", "context"]); expect(response.status).toBe(503); expect(response.headers.getSetCookie()).toHaveLength(0); });
it("TestPrivateRouteReadOnlyMode", async () => {
    vi.stubEnv("AUTH_PUBLIC_ORIGIN", "");
    try {
        const response = await authRouteGET(req("GET", "auth/session"), { params: Promise.resolve({ segments: ["session"] }) });
        expect(response.status).toBe(503);
        expect(await response.json()).toMatchObject({ error: { code: "AUTH_NOT_CONFIGURED" } });
    }
    finally {
        vi.unstubAllEnvs();
    }
});
it("TestPrivateProxyPathMethodPrecedence", async () => { const proxy = createPrivateProxy(origin, options, async () => { throw new Error("Unexpected upstream request."); }); for (const [method, path, segments, status] of [["HEAD", "auth/context?extra=1", ["auth", "context"], 405], ["HEAD", "admin/users?limit=0", ["admin", "users"], 405], ["GET", "auth/unknown?q=%FF", ["auth", "unknown"], 404]] as const)
    expect((await proxy(req(method, path), [...segments])).status).toBe(status); });
