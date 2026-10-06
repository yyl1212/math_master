import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, it, expect, vi } from "vitest";
import { PublicationPanel } from "./publication-panel";
import { WithdrawalPanel } from "./withdrawal-panel";
import { publicationView, fixtureID } from "@/lib/content/test-fixtures";
import { contentFailure } from "@/lib/content/schemas";
const mocks = vi.hoisted(() => ({ request: vi.fn(), auth: vi.fn(), refresh: vi.fn(), push: vi.fn() }));
vi.mock("@/lib/content/client", () => ({ contentRequest: mocks.request }));
vi.mock("@/lib/auth/client", () => ({ authRequest: mocks.auth }));
vi.mock("next/navigation", () => ({ useRouter: () => mocks }));
const page = () => ({ items: [publicationView()], total: 1, limit: 20, offset: 0, head: null });
beforeEach(() => { Object.values(mocks).forEach(m => m.mockReset()); mocks.request.mockResolvedValue({ ok: true, data: { items: [], total: 0, limit: 20, offset: 0 } }); });
it("TestPublicationAndWithdrawalUI", async () => { render(<PublicationPanel initial={page()}/>); expect(screen.getByText("No content has been published yet.")).toBeVisible(); expect(screen.getByText(/Added: 0/)).toBeVisible(); expect(screen.getByText(/Replaced: 0/)).toBeVisible(); expect(screen.getByText(/Removed: 0/)).toBeVisible(); fireEvent.change(screen.getByLabelText("Publication reason"), { target: { value: "Activate independently reviewed content." } }); mocks.request.mockResolvedValue(contentFailure("REAUTHENTICATION_REQUIRED")); const button = screen.getByRole("button", { name: "Activate snapshot" }); button.focus(); fireEvent.click(button); await screen.findByRole("dialog"); fireEvent.change(screen.getByLabelText("Your password"), { target: { value: "technical test password" } }); mocks.auth.mockResolvedValue({ ok: true, data: { validUntil: "2026-10-01T12:00:00Z" } }); const writes = mocks.request.mock.calls.length; fireEvent.click(screen.getByRole("button", { name: "Verify password" })); await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument()); expect(mocks.request).toHaveBeenCalledTimes(writes); expect(button).toHaveFocus(); expect(mocks.auth.mock.calls[0]).toEqual([{ kind: "reauth" }, { password: "technical test password" }]); });
it("TestContentDialogCancel", async () => { render(<PublicationPanel initial={page()}/>); fireEvent.change(screen.getByLabelText("Publication reason"), { target: { value: "Activate independently reviewed content." } }); mocks.request.mockResolvedValue(contentFailure("REAUTHENTICATION_REQUIRED")); fireEvent.click(screen.getByRole("button", { name: "Activate snapshot" })); await screen.findByRole("dialog"); const calls = mocks.request.mock.calls.length; fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(screen.queryByRole("dialog")).not.toBeInTheDocument(); expect(mocks.request).toHaveBeenCalledTimes(calls); expect(mocks.auth).not.toHaveBeenCalled(); });
it("TestWithdrawalStaleRequiresNewPreview", async () => { const p = page(); p.head = fixtureID as never; p.items[0].status = "published"; render(<WithdrawalPanel initial={p}/>); fireEvent.change(screen.getByLabelText("Target kind"), { target: { value: "knowledge" } }); fireEvent.change(screen.getByLabelText("Target ID"), { target: { value: "fractions" } }); fireEvent.change(screen.getByLabelText("Target version"), { target: { value: "1" } }); mocks.request.mockResolvedValueOnce({ ok: true, data: { currentHead: fixtureID, target: { kind: "knowledge", id: "fractions", version: 1 }, diff: { added: 0, replaced: 0, removed: 0, changes: [] } } }); fireEvent.click(screen.getByRole("button", { name: "Preview withdrawal" })); await screen.findByText("Withdrawal preview"); fireEvent.change(screen.getByLabelText("Withdrawal reason"), { target: { value: "Correct a verified mathematical problem." } }); mocks.request.mockResolvedValue(contentFailure("PUBLICATION_STALE")); fireEvent.click(screen.getByRole("button", { name: "Withdraw version" })); await waitFor(() => expect(screen.getByRole("button", { name: "Withdraw version" })).toBeDisabled()); expect(screen.queryByText("Withdrawal complete.")).not.toBeInTheDocument(); });

it("TestWithdrawalRefreshReadsHeadOutsideHistoryPage",async()=>{
 const head=publicationView();head.status="published";head.manifest.members=[{identity:{kind:"knowledge",id:"active-knowledge",version:1,packageId:"original",packageVersion:1,sha256:"a".repeat(64)},evidence:{submissionId:fixtureID,decisionId:fixtureID,frozenDigest:"b".repeat(64),inheritedFrom:null}}];
 render(<WithdrawalPanel initial={{items:[],total:2,limit:1,offset:0,head:fixtureID}}/>);
 mocks.request.mockImplementation(async(route)=>route.kind==="readPublication"?{ok:true,data:head}:{ok:true,data:{items:[],total:2,limit:1,offset:0,head:fixtureID}});
 fireEvent.click(screen.getByRole("button",{name:"Refresh head and clear preview"}));
 await waitFor(()=>expect(mocks.request).toHaveBeenCalledWith({kind:"readPublication",id:fixtureID}));
 expect(screen.getByLabelText("Published member")).toHaveTextContent("knowledge: active-knowledge v1");
});
it("TestPublicationPagerUsesEffectiveLimitAndSelectsReturnedSnapshot",async()=>{
 const old=publicationView(),next={...publicationView(),id:"22222222-2222-4222-8222-222222222222"};
 render(<PublicationPanel initial={{items:[old],total:3,limit:1,offset:0,head:null}}/>);
 mocks.request.mockResolvedValue({ok:true,data:{items:[next],total:3,limit:1,offset:1,head:null}});
 fireEvent.click(screen.getByRole("button",{name:"Next snapshots"}));
 await waitFor(()=>expect(mocks.request).toHaveBeenCalledWith({kind:"listPublications",query:{limit:1,offset:1}}));
 expect(screen.getByLabelText("Selected snapshot")).toHaveValue(next.id);expect(screen.getByText(/Manifest SHA:/)).toBeVisible();
});
import {TopicPublicationPanel} from "./publication-panel";
import {DiffPanel} from "./diff-panel";
const topicPair={knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)};
const topicRelease={id:fixtureID,status:"draft" as const,pair:topicPair,manifestSHA:"b".repeat(64),assignmentsSHA:"c".repeat(64),knowledgePublicationId:null,diff:{added:[{id:"fractions",version:1,sha256:"d".repeat(64)}],removed:[],changedTopicMemberships:[{knowledge:{id:"numbers",version:1,sha256:"e".repeat(64)},oldTopicIds:["msc-00a00"],newTopicIds:["msc-00a01"]}]},createdAt:"2026-10-01T00:00:00Z"};
it("shows accurate topic additions and membership changes",()=>{
 render(<DiffPanel diff={publicationView().diff} topics={topicRelease.diff}/>);
 expect(screen.queryByText("fractions v1")).toBeVisible();
 expect(screen.queryByText("msc-00a00 → msc-00a01")).toBeVisible();
});
it("prepares an empty topic catalogue and never activates automatically after password verification",async()=>{
 let prepared:unknown,writes=0;
 render(<TopicPublicationPanel pair={topicPair} onPrepare={async inValue=>{prepared=inValue;return {ok:true,data:topicRelease}}} onActivate={async()=>{writes++;return {ok:false,status:403,code:"REAUTH_REQUIRED",message:"reauth",requestId:"a"}}}/>);
 fireEvent.change(screen.getByLabelText("Publication reason"),{target:{value:"Publish an empty isolated catalogue first."}});
 fireEvent.click(screen.getByRole("button",{name:"Prepare paired publication"}));
 await screen.findByRole("button",{name:"Activate paired publication"});
 expect(prepared).toEqual({submissionIds:[],expectedPair:topicPair,reason:"Publish an empty isolated catalogue first."});
 fireEvent.click(screen.getByRole("button",{name:"Activate paired publication"}));
 await screen.findByRole("dialog");
 fireEvent.change(screen.getByLabelText("Your password"),{target:{value:"technical test password"}});
 mocks.auth.mockResolvedValue({ok:true,data:{validUntil:"2026-10-01T12:00:00Z"}});
 fireEvent.click(screen.getByRole("button",{name:"Verify password"}));
 await waitFor(()=>expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
 expect(writes).toBe(1);
});
