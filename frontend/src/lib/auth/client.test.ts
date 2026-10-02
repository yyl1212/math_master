import { it, expect, vi } from "vitest";
import { getAuthContext, authRequest, notifyAuthChanged } from "./client";
const context = { data: { user: null, csrfToken: "A".repeat(43) } };
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
it("TestContextSingleFlight", async () => { let complete: (response: Response) => void = () => { }; let calls = 0; vi.spyOn(globalThis, "fetch").mockImplementation(async () => { calls++; return new Promise<Response>(resolve => { complete = resolve; }); }); const one = getAuthContext(), two = getAuthContext(); await Promise.resolve(); expect(calls).toBe(1); complete(json(context)); const [a, b] = await Promise.all([one, two]); expect(a.ok && b.ok).toBe(true); expect(a).toEqual(b); const next = getAuthContext(); await Promise.resolve(); expect(calls).toBe(2); complete(json(context)); await next; vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("a private fault")); expect((await getAuthContext()).ok).toBe(false); vi.spyOn(globalThis, "fetch").mockResolvedValue(json(context)); expect((await getAuthContext()).ok).toBe(true); });
it("TestPrivateClientErrorSemantics", async () => {
    for (const [status, code, message] of [[401, "INVALID_CREDENTIALS", "Invalid username or password."], [401, "AUTHENTICATION_REQUIRED", "Please sign in to continue."], [429, "RATE_LIMITED", "Too many requests. Try again later."]] as const) {
        let writes = 0;
        vi.spyOn(globalThis, "fetch").mockImplementation(async (_input, init) => {
            if (init?.method === "POST") {
                writes++;
                const r = json({ error: { code, message, requestId: "a".repeat(32) } }, status);
                if (status === 429)
                    r.headers.set("Retry-After", "25");
                return r;
            }
            return json(context);
        });
        const result = await authRequest<void>({ kind: "password" }, { currentPassword: "a public test password", newPassword: "another public password" });
        expect(result).toMatchObject({ ok: false, status, code, message });
        expect(writes).toBe(1);
        if (status === 429)
            expect(result).toMatchObject({ retryAfter: 25 });
    }
    vi.spyOn(globalThis, "fetch").mockResolvedValue(json({ data: { user: { password: "private-test-field" } } }));
    const r = await getAuthContext();
    expect(r).toMatchObject({ ok: false, status: 503, code: "SERVICE_UNAVAILABLE" });
    expect(JSON.stringify(r)).not.toContain("private-test-field");
});

it("TestFreshAuthContextDoesNotReuseAnOutstandingAccountProof", async () => {
 const user = (id: string) => ({ id, username: "learner", roles: ["learner"], mustChangePassword: false });
 let finishOlder: (r: Response) => void = () => {};
 let calls = 0;
 vi.spyOn(globalThis, "fetch").mockImplementation(async () => {
  if (++calls === 1) return new Promise<Response>(resolve => { finishOlder = resolve; });
  return json({ data: { user: user("22222222-2222-4222-8222-222222222222"), csrfToken: "A".repeat(43) } });
 });
 const older = getAuthContext(); const fresh = getAuthContext(true);
 finishOlder(json({ data: { user: user("11111111-1111-4111-8111-111111111111"), csrfToken: "A".repeat(43) } }));
 const [a, b] = await Promise.all([older, fresh]);
 expect(a.ok && a.data.user?.id).toBe("11111111-1111-4111-8111-111111111111");
 expect(b.ok && b.data.user?.id).toBe("22222222-2222-4222-8222-222222222222");
 expect(calls).toBe(2);
});
it("TestAuthChangeBroadcastIsEphemeralAndContainsNoIdentity", () => {
 const post = vi.fn(), close = vi.fn(), channels: string[] = [];
 vi.stubGlobal("BroadcastChannel", class { constructor(name: string) { channels.push(name); } postMessage = post; close = close; });
 const event = vi.spyOn(window, "dispatchEvent");
 try {
  notifyAuthChanged();
  expect(event).toHaveBeenCalledWith(expect.objectContaining({ type: "math-master:auth-change" }));
  expect(channels).toEqual(["math-master-auth"]);
  expect(post).toHaveBeenCalledExactlyOnceWith("changed");
  expect(close).toHaveBeenCalledOnce();
 } finally { vi.unstubAllGlobals(); }
});
