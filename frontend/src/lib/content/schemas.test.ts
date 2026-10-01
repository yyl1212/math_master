import { it, expect } from "vitest";
import { parseContentJSON, draftViewSchema, submissionViewSchema, publicationViewSchema, validContentInput, contentRouteRequest } from "./schemas";
import { draftInput, draftView, submissionView, publicationView, fixtureID } from "./test-fixtures";
it("TestContentSchemasExactDTO", () => {
    expect(draftViewSchema.safeParse(draftView()).success).toBe(true);
    expect(submissionViewSchema.safeParse(submissionView()).success).toBe(true);
    expect(publicationViewSchema.safeParse(publicationView()).success).toBe(true);
    expect(draftViewSchema.safeParse({ ...draftView(), passwordPHC: "private" }).success).toBe(false);
    const mutated = draftView();
    mutated.package.knowledge[0] = { ...mutated.package.knowledge[0], authorIds: [fixtureID] } as never;
    expect(draftViewSchema.safeParse(mutated).success).toBe(false);
    expect(submissionViewSchema.safeParse({ ...submissionView(), status: "approved", review: null }).success).toBe(false);
    expect(validContentInput("prepareRelease", { submissionIds: [fixtureID], reason: "A technical reason." })).toBe(false);
    expect(validContentInput("prepareRelease", { submissionIds: [fixtureID], expectedHead: null, reason: "A technical reason." })).toBe(true);
    expect(validContentInput("createDraft", draftInput())).toBe(true);
    expect(contentRouteRequest({ kind: "readDraft", id: "https://untrusted.example" })).toBe(null);
    expect(contentRouteRequest({ kind: "readDraft", id: fixtureID, url: "https://untrusted.example" } as never)).toBe(null);
});
it("TestContentRawJSONExactBoundary", () => {
    for (const raw of ['{"expectedHead":null,"expectedHead":null}', '{"expectedHead":null,"ExpectedHead":null}', '{"value":"\\ud800"}', '{"value":"\\udc00"}', '{"value":"\\u0000"}', '{"value":{"x":1,"X":2}}', '{} {}', '{"n":1.0}', '{"n":1e0}', '{"n":9007199254740993}']) {
        expect(() => parseContentJSON(new TextEncoder().encode(raw))).toThrow();
    }
    for (const depth of [32, 33]) {
        const raw = '{"payload":' + '['.repeat(depth - 1) + '0' + ']'.repeat(depth - 1) + '}';
        if (depth === 32)
            expect(parseContentJSON(new TextEncoder().encode(raw))).toBeTruthy();
        else
            expect(() => parseContentJSON(new TextEncoder().encode(raw))).toThrow();
    }
    expect(() => parseContentJSON(new Uint8Array([123, 34, 120, 34, 58, 34, 255, 34, 125]))).toThrow();
    expect(parseContentJSON(new TextEncoder().encode('{"value":"\\ud83d\\ude00"}'))).toEqual({ value: "😀" });
});
