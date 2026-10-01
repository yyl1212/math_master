import { readFileSync } from "node:fs";
import type { DraftInput, DraftView, PublicationView, SubmissionView } from "./types";
export const fixtureID = "11111111-1111-4111-8111-111111111111", otherID = "22222222-2222-4222-8222-222222222222", reviewerID = "33333333-3333-4333-8333-333333333333";
export const requestID = "a".repeat(32), token = "A".repeat(43);
export const fixtureSVG = new Uint8Array(readFileSync("../backend/internal/content/testdata/workflow-ready.svg"));
export function draftInput(): DraftInput {
    return { catalogueVersion: 1, package: JSON.parse(readFileSync("../backend/internal/content/testdata/workflow-ready.json", "utf8")), assetBytes: [{ id: "halves", base64: Buffer.from(fixtureSVG).toString("base64") }], sourceMap: [] };
}
export function draftView(): DraftView {
    const input = draftInput();
    return { id: fixtureID, ownerId: otherID, catalogueSha256: "b".repeat(64), catalogueVersion: 1, revision: 1, status: "editing", package: input.package, sourceMap: [], authorIds: [otherID], legacyUnattributed: false, assets: input.package.assets.map(({ path: _path, ...a }) => a), gate: { structuralErrors: [], completenessErrors: [], humanReviewRequirements: [], structuralTotal: 0, completenessTotal: 0, humanReviewTotal: 0, truncated: false, readyToSubmit: true, digest: "a".repeat(64) }, createdAt: "2026-10-01T00:00:00Z", updatedAt: "2026-10-01T00:00:00Z" };
}
export function submissionView(): SubmissionView {
    const d = draftView();
    return { id: fixtureID, workspaceId: otherID, ownerId: d.ownerId, status: "pending", revision: d.revision, frozen: { catalogueVersion: d.catalogueVersion, catalogueSha256: d.catalogueSha256, package: d.package, sourceMap: d.sourceMap, authorIds: d.authorIds, legacyUnattributed: false, assets: d.assets, frozenDigest: "c".repeat(64) }, gate: d.gate, review: null, createdAt: d.createdAt };
}
export function publicationView(): PublicationView { return { id: fixtureID, status: "draft", manifestSha: "d".repeat(64), createdAt: "2026-10-01T00:00:00Z", manifest: { catalogueVersion: 1, catalogueSha256: "b".repeat(64), baseHead: null, members: [], bindings: [] }, diff: { added: 0, replaced: 0, removed: 0, changes: [] } }; }
export const contentJSON = (body: unknown, status = 200, headers: HeadersInit = {}) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json", "X-Request-ID": requestID, ...headers } });
