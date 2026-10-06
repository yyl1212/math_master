import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, it, expect, vi } from "vitest";
import { DraftEditor } from "./draft-editor";
import { DraftList } from "./draft-list";
import { importDraftInput, exportDraftInput } from "./asset-transfer";
import { draftView, draftInput, fixtureID, fixtureSVG } from "@/lib/content/test-fixtures";
import { contentFailure } from "@/lib/content/schemas";
import { AuthStatus } from "@/components/auth-status";
const mocks = vi.hoisted(() => ({ request: vi.fn(), asset: vi.fn(), refresh: vi.fn(), push: vi.fn(), context: vi.fn() }));
vi.mock("@/lib/content/client", () => ({ contentRequest: mocks.request, readContentAsset: mocks.asset }));
vi.mock("next/navigation", () => ({ useRouter: () => mocks, usePathname: () => "/editor" }));
vi.mock("@/lib/auth/client", () => ({ getAuthContext: mocks.context }));
beforeEach(() => { Object.values(mocks).forEach(m => m.mockReset()); mocks.asset.mockResolvedValue({ ok: true, data: fixtureSVG }); mocks.context.mockResolvedValue({ ok: true, data: { user: { id: fixtureID, username: "content_editor", roles: ["learner", "editor"], mustChangePassword: false } } }); });
it("TestAuthoringForms", async () => {
    const view = draftView();
    view.package.paths = [{ id: "fractions-path", version: 1, domainIds: ["elementary-mathematics"], title: "Original fractions path", titleZh: "分数路线", nodes: [{ id: "fractions", version: 1 }] }];
    view.gate.readyToSubmit = false;
    view.gate.completenessTotal = 1;
    view.gate.completenessErrors = [{ code: "MISSING", path: "statement", message: "Complete statement." }];
    render(<DraftEditor initial={view}/>);
    for (const label of ["Catalogue version", "Package ID", "Package version", "Knowledge 1 title", "Knowledge 1 Chinese title", "Knowledge 1 statement", "Knowledge 1 scope", "Knowledge 1 system", "Knowledge 1 proof", "Unit 1 version", "Unit 1 angle 1 body", "Path 1 version", "Knowledge 1 source 1 author"]) {
        expect(screen.getByLabelText(label)).toBeVisible();
    }
    expect(screen.getByRole("button", { name: "Submit for review" })).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Knowledge 1 statement"), { target: { value: "Unsaved mathematical statement." } });
    mocks.request.mockResolvedValue(contentFailure("DRAFT_CONFLICT"));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("This draft has changed."));
    expect(screen.getByLabelText("Knowledge 1 statement")).toHaveValue("Unsaved mathematical statement.");
    expect(mocks.request.mock.calls[0][1].expectedRevision).toBe(1);
});
it("TestAuthoringJSONAndPrivatePreview", async () => {
    const view = draftView();
    const imported = importDraftInput(new TextEncoder().encode(JSON.stringify(draftInput())));
    expect(imported).toEqual(draftInput());
    const exported = await exportDraftInput({ catalogueVersion: view.catalogueVersion, package: view.package, sourceMap: view.sourceMap }, { kind: "draft", id: view.id });
    expect(JSON.parse(exported)).toEqual(draftInput());
    expect(exported).not.toMatch(/csrf|password|cookie|authorIds/i);
    expect(mocks.asset).toHaveBeenCalledWith({ kind: "draft", id: view.id }, view.assets[0].sha256);
    const invalid = draftInput();
    invalid.assetBytes[0].base64 = btoa('<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>');
    expect(() => importDraftInput(new TextEncoder().encode(JSON.stringify(invalid)))).toThrow();
    render(<DraftEditor initial={view}/>);
    expect(screen.getByRole("img").getAttribute("src")).toBe(`/api/v1/content/drafts/${view.id}/assets/${view.assets[0].sha256}`);
});
it("TestContentKeyboardAndNavigation", async () => {
    const { rerender } = render(<AuthStatus />);
    expect(await screen.findByRole("link", { name: "Edit content" })).toHaveAttribute("href", "/editor");
    expect(screen.queryByRole("link", { name: "Review content" })).not.toBeInTheDocument();
    mocks.context.mockResolvedValue({ ok: true, data: { user: { id: fixtureID, username: "content_admin", roles: ["learner", "reviewer", "admin"], mustChangePassword: false } } });
    window.dispatchEvent(new Event("math-master:auth-change"));
    await screen.findByRole("link", { name: "Review content" });
    expect(screen.getByRole("link", { name: "Publish content" })).toHaveAttribute("href", "/admin/publications");
    expect(screen.queryByRole("link", { name: "Edit content" })).not.toBeInTheDocument();
    rerender(<AuthStatus />);
    expect(document.body.textContent).not.toMatch(/mastered|points|learning progress/i);
});

it("keeps unsaved edits in the editor until the saved reading entry is available", async () => {
    const view = draftView();
    render(<DraftEditor initial={view}/>);
    expect(screen.queryByRole("link", { name: "Read saved draft" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Read saved draft" })).toHaveAttribute("href", "/editor/drafts/11111111-1111-4111-8111-111111111111/preview");
    fireEvent.change(screen.getByLabelText("Knowledge 1 statement"), { target: { value: "A saved statement for the reader." } });
    expect(screen.queryByRole("link", { name: "Read saved draft" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read saved draft" })).toBeDisabled();
    const saved = structuredClone(view);
    saved.revision = 2;
    saved.package.knowledge[0].statement = "A saved statement for the reader.";
    mocks.request.mockResolvedValue({ ok: true, data: saved, status: 200, requestId: "a".repeat(32) });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await screen.findByRole("link", { name: "Read saved draft" });
    expect(screen.getByText(/Revision 2 · editing/)).toBeVisible();
});
it("opens the saved reader directly from the workspace list", () => {
    const view = draftView();
    render(<DraftList initial={{ items: [{ id: view.id, ownerId: view.ownerId, packageId: view.package.id, packageVersion: 1, catalogueVersion: 1, createdAt: view.createdAt, revision: 1, status: "editing", structuralTotal: 0, completenessTotal: 0, updatedAt: view.updatedAt }], total: 1, limit: 20, offset: 0 }} canEdit={false}/>);
    expect(screen.queryByRole("link", { name: "Read saved draft" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Read saved draft" })).toHaveAttribute("href", "/editor/drafts/11111111-1111-4111-8111-111111111111/preview");
});
it("requires the current saved revision topic assignments before submitting",()=>{
 const view=draftView();render(<DraftEditor initial={view} topics={{draftId:view.id,draftRevision:1,assignmentRevision:0,taxonomyVersionId:"a".repeat(64),members:[],digest:"d".repeat(64),readyToSubmit:false}} onSaveTopics={async()=>{throw new Error("unavailable")}}/>);
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeDisabled();
 expect(screen.getByRole("button",{name:"Save topic assignment"})).toBeVisible();
});
it("blocks submission while a topic selection has unsaved changes",()=>{
 const view=draftView(),member={knowledge:{id:view.package.knowledge[0].id,version:1},topicIds:["msc-00a00"],sourceRefs:[],sourceBatchSHA:"b".repeat(64)};
 render(<DraftEditor initial={view} topics={{draftId:view.id,draftRevision:1,assignmentRevision:1,taxonomyVersionId:"a".repeat(64),members:[member],digest:"d".repeat(64),readyToSubmit:true}} onSaveTopics={async()=>{throw new Error("unavailable")}}/>);
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeEnabled();
 fireEvent.change(screen.getByLabelText("Specific topic IDs"),{target:{value:"msc-00a01"}});
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeDisabled();
});
