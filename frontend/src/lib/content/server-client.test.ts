import { it, expect, vi, afterEach } from "vitest";
import { readServerContent } from "./server-client";
import { draftView, fixtureID, requestID, token, contentJSON } from "./test-fixtures";
afterEach(() => vi.unstubAllEnvs());
it("TestContentSSR", async () => {
    vi.stubEnv("GO_API_INTERNAL_URL", "http://127.0.0.1:8080");
    vi.stubEnv("AUTH_PUBLIC_ORIGIN", "http://127.0.0.1:3000");
    vi.stubEnv("APP_ENV", "development");
    let sent: RequestInit | undefined, url = "";
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { url = String(input); sent = init; return contentJSON(draftView()); });
    const result = await readServerContent({ kind: "readDraft", id: fixtureID }, `foreign=excluded; mm_session_dev=${token}; mm_preauth_dev=${token}`);
    expect(result.ok).toBe(true);
    expect(url).toBe("http://127.0.0.1:8080/api/v1/content/drafts/" + fixtureID);
    expect(new Headers(sent?.headers).get("Cookie")).toBe(`mm_session_dev=${token}`);
    expect(sent).toMatchObject({ method: "GET", cache: "no-store", redirect: "error" });
    expect(JSON.stringify(result)).not.toContain(token);
    expect(new Headers(sent?.headers).has("X-CSRF-Token")).toBe(false);
    for (const [status, code, message] of [[401, "AUTHENTICATION_REQUIRED", "Please sign in to continue."], [403, "FORBIDDEN", "You do not have permission."], [503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable."]] as const) {
        vi.spyOn(globalThis, "fetch").mockResolvedValue(contentJSON({ error: { code, message, requestId: requestID } }, status));
        expect(await readServerContent({ kind: "readDraft", id: fixtureID }, "")).toMatchObject({ ok: false, status, code });
    }
    vi.spyOn(globalThis, "fetch").mockResolvedValue(contentJSON(draftView(), 200, { "Set-Cookie": "foreign=private" }));
    expect(await readServerContent({ kind: "readDraft", id: fixtureID }, "")).toMatchObject({ ok: false, status: 503 });
    expect(await readServerContent({ kind: "createDraft" }, "")).toMatchObject({ ok: false, status: 400 });
});
