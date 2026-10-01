import { it, expect, vi, afterEach } from "vitest";
import { parseAuthOrigin, getAuthConfig } from "./config";
import { readServerSession, readServerUsers } from "./server-client";
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
afterEach(() => vi.unstubAllEnvs());
it("TestServerSessionIsolation", async () => {
    vi.stubEnv("GO_API_INTERNAL_URL", "http://127.0.0.1:8080");
    vi.stubEnv("APP_ENV", "development");
    vi.stubEnv("AUTH_PUBLIC_ORIGIN", "http://127.0.0.1:3000");
    let url = "";
    let options: RequestInit | undefined;
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { url = String(input); options = init; const r = json({ data: { user: null } }); r.headers.append("Set-Cookie", "foreign=ignored"); return r; });
    const result = await readServerSession(`private=excluded; mm_session_dev=${"A".repeat(43)}; mm_preauth_dev=${"A".repeat(43)}`);
    expect(result).toEqual({ ok: true, data: null });
    expect(url).toBe("http://127.0.0.1:8080/api/v1/auth/session");
    expect(new Headers(options?.headers).get("Cookie")).toBe(`mm_session_dev=${"A".repeat(43)}`);
    expect(options).toMatchObject({ cache: "no-store", redirect: "error" });
    expect(JSON.stringify(result)).not.toContain("127.0.0.1");
    for (const [status, code, message] of [[403, "FORBIDDEN", "You do not have permission."], [503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable."]] as const) {
        vi.spyOn(globalThis, "fetch").mockResolvedValue(json({ error: { code, message, requestId: "a".repeat(32) } }, status));
        expect(await readServerUsers("", { q: "", limit: 20, offset: 0 })).toMatchObject({ ok: false, status, code });
    }
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("private database address"));
    expect(await readServerSession("")).toMatchObject({ ok: false, status: 503 });
});
it("TestServerAuthConfigCanonicalOrigin", () => { for (const [raw, want] of [["http://[0:0:0:0:0:0:0:1]:003000", "http://[::1]:3000"], ["http://[::ffff:127.0.0.1]:3000", "http://[::ffff:7f00:1]:3000"], ["https://MATH.EXAMPLE:00443", "https://math.example"]])
    expect(parseAuthOrigin(raw, raw.startsWith("https:"))).toBe(want); vi.stubEnv("APP_ENV", ""); vi.stubEnv("AUTH_PUBLIC_ORIGIN", "http://localhost:3000"); expect(getAuthConfig().production).toBe(false); for (const raw of ["http://remote.invalid", "https://%6dath.example", "https://math.example\\", "http://127.1"])
    expect(() => parseAuthOrigin(raw, false)).toThrow(); });
