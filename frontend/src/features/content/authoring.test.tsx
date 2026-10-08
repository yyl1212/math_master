import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, it, expect, vi } from "vitest";
import { DraftEditor } from "./draft-editor";
import { DraftList } from "./draft-list";
import { importDraftInput, exportDraftInput } from "./asset-transfer";
import { draftView, draftInput, fixtureID, fixtureSVG } from "@/lib/content/test-fixtures";
import { contentFailure } from "@/lib/content/schemas";
import type {DraftTopicInput} from "@/lib/taxonomy/types";
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
it.each([1,2])("reconfirms upgraded knowledge from v%s with its saved version while preserving topic candidates",async initialVersion=>{
 const view=draftView();view.package.knowledge[0].version=initialVersion;
 const member={knowledge:{id:view.package.knowledge[0].id,version:1},topicIds:["msc-00a00"],sourceRefs:[{sourceId:"source",workFamilyId:"work",recordId:"original",path:"data.json",sha256:"c".repeat(64)}],sourceBatchSHA:"b".repeat(64)};
 const topics={draftId:view.id,draftRevision:1,assignmentRevision:1,taxonomyVersionId:"a".repeat(64),members:[member],digest:"d".repeat(64),readyToSubmit:false};
 const save=vi.fn(async (input:DraftTopicInput)=>({...topics,members:[input.member],assignmentRevision:2,readyToSubmit:true}));
 render(<DraftEditor initial={view} topics={topics} onSaveTopics={save}/>);
 fireEvent.change(screen.getByLabelText("Specific topic IDs"),{target:{value:"msc-00a01"}});
 if(initialVersion===1){const saved=structuredClone(view);saved.revision=2;saved.package.knowledge[0].version=2;fireEvent.change(screen.getByLabelText("Knowledge 1 title"),{target:{value:"Revised original fractions"}});mocks.request.mockResolvedValue({ok:true,data:saved,status:200,requestId:"a".repeat(32)});fireEvent.click(screen.getByRole("button",{name:"Save draft"}));await waitFor(()=>expect(screen.getByRole("button",{name:"Save topic assignment"})).toBeEnabled());topics.draftRevision=2;}
 fireEvent.click(screen.getByRole("button",{name:"Save topic assignment"}));
 await screen.findByText("Topic assignment saved.");
 expect(save.mock.calls[0][0].member.knowledge).toEqual({id:member.knowledge.id,version:2});
 expect(save.mock.calls[0][0].member.topicIds).toEqual(["msc-00a01"]);
 expect(save.mock.calls[0][0].member.sourceBatchSHA).toBe(member.sourceBatchSHA);
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeEnabled();
});

it("automatically saves imported topic assignments with the new saved draft revision",async()=>{
 const view=draftView(),input=draftInput();
 const batch="b".repeat(64),version="c".repeat(64),kid=input.package.knowledge[0].id;
 const member={knowledge:{id:kid,version:1},topicIds:["msc-00a00"],sourceBatchSHA:batch,sourceRefs:[{sourceId:"original-source",workFamilyId:"original-work",recordId:"original-1",path:"Original/data.json",sha256:"d".repeat(64)}]};
 input.sourceMap=[{knowledge:member.knowledge,batchSha256:batch,relativePath:"Original/data.json",sha256:"d".repeat(64),legacyId:"original-1",note:"Original technical fixture only."}];
 const topics={draftId:view.id,draftRevision:1,assignmentRevision:0,taxonomyVersionId:version,members:[],digest:"e".repeat(64),readyToSubmit:false};
 const saved={...view,revision:2,package:input.package,sourceMap:input.sourceMap};
 mocks.request.mockResolvedValue({ok:true,data:saved,status:200,requestId:"a".repeat(32)});
 const read=vi.fn(async()=>({...topics,draftRevision:2}));
 const save=vi.fn(async(p:DraftTopicInput)=>({...topics,draftRevision:2,assignmentRevision:1,members:[{sourceBatchSHA:p.member.sourceBatchSHA,sourceRefs:p.member.sourceRefs,topicIds:p.member.topicIds,knowledge:p.member.knowledge}],readyToSubmit:true}));
 render(<DraftEditor initial={view} topics={topics} onSaveTopics={save} onReadTopics={read}/>);
 const bytes=new TextEncoder().encode(JSON.stringify({kind:"topic-draft",schemaVersion:1,draft:input,assignments:[member],sourceBatchSHA:batch,taxonomyVersionId:version}));
 const file=new File([bytes],"original-topic-draft.json",{type:"application/json"});Object.defineProperty(file,"arrayBuffer",{value:async()=>bytes.buffer});
 fireEvent.change(screen.getByLabelText("Import DraftInput JSON"),{target:{files:[file]}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"Save draft"})).toBeEnabled());
 await waitFor(()=>expect(screen.getByLabelText("Specific topic IDs")).toHaveValue("msc-00a00"));
 fireEvent.click(screen.getByRole("button",{name:"Save draft"}));
 await screen.findByText("Topic import: 1 assigned, 0 pending, 0 failed.");
 expect(save.mock.calls[0][0]).toEqual({expectedDraftRevision:2,expectedAssignmentRevision:0,taxonomyVersionId:version,member});
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeEnabled();
});

it("retries an uncertain imported assignment without resaving the body or changing its input",async()=>{
 const view=draftView(),input=draftInput(),kid=input.package.knowledge[0].id,batch="b".repeat(64),version="c".repeat(64);
 const member={knowledge:{id:kid,version:1},topicIds:["msc-00a00"],sourceBatchSHA:batch,sourceRefs:[{sourceId:"original-source",workFamilyId:"original-work",recordId:"original-1",path:"Original/data.json",sha256:"d".repeat(64)}]};
 input.sourceMap=[{knowledge:member.knowledge,batchSha256:batch,relativePath:"Original/data.json",sha256:"d".repeat(64),legacyId:"original-1",note:"Original technical fixture only."}];
 const topics={draftId:view.id,draftRevision:1,assignmentRevision:0,taxonomyVersionId:version,members:[],digest:"e".repeat(64),readyToSubmit:false};
 const saved={...view,revision:2,package:input.package,sourceMap:input.sourceMap};mocks.request.mockResolvedValue({ok:true,data:saved,status:200,requestId:"a".repeat(32)});
 let calls=0;const save=vi.fn(async(p:DraftTopicInput)=>{if(++calls===1)throw Error("uncertain network result");return {...topics,draftRevision:2,assignmentRevision:1,members:[p.member],readyToSubmit:true}});
 render(<DraftEditor initial={view} topics={topics} onSaveTopics={save} onReadTopics={async()=>({...topics,draftRevision:2})}/>);
 const bytes=new TextEncoder().encode(JSON.stringify({kind:"topic-draft",schemaVersion:1,draft:input,assignments:[member],sourceBatchSHA:batch,taxonomyVersionId:version}));const file=new File([bytes],"original.json",{type:"application/json"});Object.defineProperty(file,"arrayBuffer",{value:async()=>bytes.buffer});
 fireEvent.change(screen.getByLabelText("Import DraftInput JSON"),{target:{files:[file]}});await waitFor(()=>expect(screen.getByLabelText("Specific topic IDs")).toHaveValue("msc-00a00"));
 fireEvent.click(screen.getByRole("button",{name:"Save draft"}));await screen.findByText("Topic import: 0 assigned, 0 pending, 1 failed.");
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeDisabled();
 fireEvent.click(screen.getByRole("button",{name:"Retry unsaved assignments"}));await screen.findByText("Topic import: 1 assigned, 0 pending, 0 failed.");
 expect(mocks.request).toHaveBeenCalledTimes(1);expect(save).toHaveBeenCalledTimes(2);expect(save.mock.calls[0][0]).toEqual(save.mock.calls[1][0]);
});

function twoImportedMembers(){
 const view=draftView(),input=draftInput(),batch="b".repeat(64),version="c".repeat(64);
 const other=structuredClone(input.package.knowledge[0]);other.id="other-concept";other.title="Other original concept";input.package.knowledge.push(other);
 const members=input.package.knowledge.map((k,i)=>({knowledge:{id:k.id,version:k.version},topicIds:["msc-00a00"],sourceBatchSHA:batch,sourceRefs:[{sourceId:"original-source",workFamilyId:"original-work",recordId:"original-"+i,path:"Original/data.json",sha256:"d".repeat(64)}]}));
 input.sourceMap=members.map(m=>({knowledge:m.knowledge,batchSha256:batch,relativePath:"Original/data.json",sha256:"d".repeat(64),legacyId:m.sourceRefs[0].recordId,note:"Original technical fixture."}));
 const topics={draftId:view.id,draftRevision:1,assignmentRevision:0,taxonomyVersionId:version,members:[] as typeof members,digest:"e".repeat(64),readyToSubmit:false};
 const bytes=new TextEncoder().encode(JSON.stringify({kind:"topic-draft",schemaVersion:1,draft:input,assignments:members,sourceBatchSHA:batch,taxonomyVersionId:version}));const file=new File([bytes],"original.json",{type:"application/json"});Object.defineProperty(file,"arrayBuffer",{value:async()=>bytes.buffer});
 return {view,input,members,topics,file};
}
it("removing an imported member does not leave hidden dirty state blocking submission",async()=>{
 const {view,input,topics,file}=twoImportedMembers();const saved={...view,revision:2,package:{...input.package,knowledge:input.package.knowledge.slice(0,1)},sourceMap:input.sourceMap.slice(0,1)};
 mocks.request.mockResolvedValue({ok:true,data:saved,status:200,requestId:"a".repeat(32)});
 render(<DraftEditor initial={view} topics={topics} onReadTopics={async()=>({...topics,draftRevision:2})} onSaveTopics={async p=>({...topics,draftRevision:2,assignmentRevision:1,members:[p.member],readyToSubmit:true})}/>);
 fireEvent.change(screen.getByLabelText("Import DraftInput JSON"),{target:{files:[file]}});await waitFor(()=>expect(screen.getAllByLabelText("Specific topic IDs")).toHaveLength(1));
 await waitFor(()=>expect(screen.getByLabelText("Knowledge 2 title")).toBeVisible());
 fireEvent.click(screen.getAllByRole("button",{name:/Remove knowledge/i})[1]);
 fireEvent.click(screen.getByRole("button",{name:"Save draft"}));await screen.findByText("Topic import: 1 assigned, 0 pending, 0 failed.");
 expect(screen.getByRole("button",{name:"Submit for review"})).toBeEnabled();
});
it("manual correction after partial import remains the candidate on a later body save",async()=>{
 const {view,input,topics,file}=twoImportedMembers();let revision=2,assignmentRevision=0,failB=true;const stored=new Map<string,DraftTopicInput['member']>();const writes:DraftTopicInput[]=[];
 mocks.request.mockImplementation(async()=>({ok:true,data:{...view,revision:revision++,package:input.package,sourceMap:input.sourceMap},status:200,requestId:"a".repeat(32)}));
 const save=async(p:DraftTopicInput)=>{writes.push(structuredClone(p));if(p.member.knowledge.id==='other-concept'&&failB){failB=false;throw Error('uncertain B')}stored.set(p.member.knowledge.id,p.member);return {...topics,draftRevision:p.expectedDraftRevision,assignmentRevision:++assignmentRevision,members:[...stored.values()],readyToSubmit:stored.size===2}};
 render(<DraftEditor initial={view} topics={topics} onReadTopics={async()=>({...topics,draftRevision:revision-1,assignmentRevision,members:[...stored.values()]})} onSaveTopics={save}/>);
 fireEvent.change(screen.getByLabelText("Import DraftInput JSON"),{target:{files:[file]}});await waitFor(()=>expect(screen.getByLabelText("Knowledge 2 title")).toBeVisible());
 fireEvent.click(screen.getByRole("button",{name:"Save draft"}));await screen.findByText("Topic import: 1 assigned, 0 pending, 1 failed.");
 fireEvent.change(screen.getAllByLabelText("Specific topic IDs")[0],{target:{value:"msc-00a01"}});fireEvent.click(screen.getAllByRole("button",{name:"Save topic assignment"})[0]);await screen.findByText("Topic assignment saved.");
 fireEvent.change(screen.getByLabelText("Knowledge 1 title"),{target:{value:"Original title edited"}});fireEvent.click(screen.getByRole("button",{name:"Save draft"}));await screen.findByText("Topic import: 2 assigned, 0 pending, 0 failed.");
 expect(writes.filter(w=>w.member.knowledge.id===input.package.knowledge[0].id).at(-1)?.member.topicIds).toEqual(["msc-00a01"]);
});
