import {render,screen,fireEvent,waitFor} from "@testing-library/react";
import {it,expect,vi} from "vitest";
import {TopicDraftEditor} from "./topic-workspace";
import {draftView} from "@/lib/content/test-fixtures";
const mocks=vi.hoisted(()=>({request:vi.fn(),push:vi.fn(),refresh:vi.fn()}));
vi.mock("@/lib/taxonomy/management-client",()=>({topicManagementRequest:mocks.request}));vi.mock("next/navigation",()=>({useRouter:()=>mocks}));
it("retries an uncertain assignment write with the same immutable input key",async()=>{
 const draft=draftView(),member={knowledge:{id:draft.package.knowledge[0].id,version:1},topicIds:["msc-00a00"],sourceRefs:[{sourceId:"source",workFamilyId:"work",recordId:"original",path:"data.json",sha256:"b".repeat(64)}],sourceBatchSHA:"c".repeat(64)},topics={draftId:draft.id,draftRevision:1,assignmentRevision:0,taxonomyVersionId:"a".repeat(64),members:[member],digest:"d".repeat(64),readyToSubmit:false};
 mocks.request.mockResolvedValue({ok:false,status:503,code:"SERVICE_UNAVAILABLE",message:"Service temporarily unavailable.",requestId:"unavailable"});
 render(<TopicDraftEditor initial={draft} topics={topics} canEdit/>);fireEvent.click(screen.getByRole("button",{name:"Save topic assignment"}));await screen.findByRole("alert");fireEvent.click(screen.getByRole("button",{name:"Save topic assignment"}));await waitFor(()=>expect(mocks.request).toHaveBeenCalledTimes(2));expect(mocks.request.mock.calls[0][2]).toBe(mocks.request.mock.calls[1][2]);expect(mocks.request.mock.calls[0][1]).toEqual(mocks.request.mock.calls[1][1]);
});
import {TopicPublicationWorkspace} from "./topic-workspace";
it("keeps the publication reason after preparing a new candidate",async()=>{
 const pair={knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)},release={id:"22222222-2222-4222-8222-222222222222",status:"draft",pair,manifestSHA:"b".repeat(64),assignmentsSHA:"c".repeat(64),knowledgePublicationId:null,diff:{added:[],removed:[],changedTopicMemberships:[]},createdAt:"2026-10-01T00:00:00Z"};
 mocks.request.mockResolvedValue({ok:true,data:release});render(<TopicPublicationWorkspace initial={{items:[],total:0,limit:20,offset:0,pair}}/>);
 fireEvent.change(screen.getByLabelText("Publication reason"),{target:{value:"Prepare an original isolated catalogue."}});fireEvent.click(screen.getByRole("button",{name:"Prepare paired publication"}));await screen.findByRole("button",{name:"Activate paired publication"});expect(screen.getByLabelText("Publication reason")).toHaveValue("Prepare an original isolated catalogue.");expect(screen.getByRole("button",{name:"Activate paired publication"})).toBeEnabled();
});
