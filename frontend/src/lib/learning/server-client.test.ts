import { it, expect, vi, afterEach } from "vitest";
import { getLearningClient } from "./server-client";
import { learningJSON, practice, fixtureID, token } from "./test-fixtures";
const jar = vi.hoisted(() => ({ value: "" }));
vi.mock("next/headers", () => ({ cookies: async () => ({ toString: () => jar.value }) }));
afterEach(() => vi.unstubAllEnvs());
it("SSR reads fresh per-user cookies with no-store and no answer endpoint for active pages", async () => { vi.stubEnv("GO_API_INTERNAL_URL", "http://127.0.0.1:8080"); vi.stubEnv("AUTH_PUBLIC_ORIGIN", "http://127.0.0.1:3000"); vi.stubEnv("APP_ENV", "development"); const seen: RequestInit[] = []; const urls: string[] = []; vi.spyOn(globalThis, "fetch").mockImplementation(async (u, i) => { urls.push(String(u)); seen.push(i!); return learningJSON(practice()); }); const client = getLearningClient(); jar.value = `foreign=private; mm_session_dev=${token}`; expect((await client.readPractice(fixtureID)).ok).toBe(true); jar.value = ""; expect((await client.readPractice(fixtureID)).ok).toBe(true); expect(new Headers(seen[0].headers).get("Cookie")).toBe(`mm_session_dev=${token}`); expect(new Headers(seen[1].headers).has("Cookie")).toBe(false); for (const i of seen)
    expect(i).toMatchObject({ cache: "no-store", redirect: "error", method: "GET" }); expect(urls.every(u => u.endsWith("/practice/" + fixtureID))).toBe(true); });
