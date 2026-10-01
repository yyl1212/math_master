import { it, expect, vi, afterEach } from "vitest";
import { readServerQuestion } from "./server-client";
import { questionJSON, draftView, fixtureID, token } from "./test-fixtures";
afterEach(() => vi.unstubAllEnvs());
it("TestQuestionServerOnlyReads", async () => {
    vi.stubEnv("GO_API_INTERNAL_URL", "http://127.0.0.1:8080");
    vi.stubEnv("AUTH_PUBLIC_ORIGIN", "http://127.0.0.1:3000");
    vi.stubEnv("APP_ENV", "development");
    const calls: {
        url: string;
        init?: RequestInit;
    }[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => { calls.push({ url: String(input), init }); return questionJSON(draftView()); });
    expect((await readServerQuestion({ kind: "readDraft", id: fixtureID }, `foreign=secret; mm_session_dev=${token}; mm_preauth_dev=${token}`)).ok).toBe(true);
    expect(calls).toHaveLength(1);
    expect(calls[0].url).toBe("http://127.0.0.1:8080/api/v1/question-bank/drafts/" + fixtureID);
    expect(calls[0].init).toMatchObject({ method: "GET", cache: "no-store", redirect: "error" });
    expect(new Headers(calls[0].init?.headers).get("Cookie")).toBe(`mm_session_dev=${token}`);
    expect(new Headers(calls[0].init?.headers).has("X-CSRF-Token")).toBe(false);
    expect(await readServerQuestion({ kind: "createDraft" }, "")).toMatchObject({ ok: false, status: 400 });
    expect(calls).toHaveLength(1);
});
