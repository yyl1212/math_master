import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, it, expect, vi } from "vitest";
import { DiffPanel } from "./diff-panel";
import { fixtureID, otherID, sha } from "./test-fixtures";
const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("@/lib/question/client", () => ({ requestQuestion: mocks.request }));
beforeEach(() => { mocks.request.mockReset(); });
it.each(["listChanges", "listMembers"] as const)("Old%sPageCannotOverwriteSelectedSnapshot", async kind => {
 let resolve!: (v: unknown) => void; let oldSignal: AbortSignal | undefined;
 const delayed = new Promise(r => { resolve = r; });
 const page = (route: { id: string; kind: string }, label: string) => {
 const identity = { kind: "template", id: label, version: 1, sha256: sha, packageId: "original-bank", packageVersion: 1 };
 const item = route.kind === "listChanges" ? { kind: "template", id: label, reason: "Addition", before: null, after: identity } : { identity, evidence: { submissionId: fixtureID, decisionId: otherID, frozenDigest: sha, inheritedFrom: null } };
 return { ok: true, data: { publicationId: route.id, manifestSha: sha, items: [item], total: 2, limit: 1, offset: 0 } };
 };
 mocks.request.mockImplementation((route, _input, _key, signal) => {
 if (route.id === fixtureID && route.kind === kind && route.query.offset === 1) { oldSignal = signal; return delayed; }
 return Promise.resolve(page(route, route.id === fixtureID ? "old-first" : "new-current"));
 });
 const view = render(<DiffPanel publicationId={fixtureID} manifestSha={sha}/>);
 const next = kind === "listChanges" ? "Next changes" : "Next members";
 await waitFor(() => expect(screen.getByRole("button", { name: next })).toBeEnabled()); fireEvent.click(screen.getByRole("button", { name: next }));
 view.rerender(<DiffPanel publicationId={otherID} manifestSha={sha}/>);
 await waitFor(() => expect(screen.getAllByText(/new-current/)).toHaveLength(2));
 await act(async () => resolve(page({ id: fixtureID, kind }, "old-delayed")));
 expect(screen.queryByText(/old-delayed/)).not.toBeInTheDocument();
 expect(screen.getAllByText(/new-current/)).toHaveLength(2);
 expect(oldSignal?.aborted).toBe(true);
});
