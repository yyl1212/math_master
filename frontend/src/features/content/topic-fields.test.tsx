import {fireEvent,render,screen,waitFor} from "@testing-library/react";
import {it,expect} from "vitest";
import {TopicFields} from "./topic-fields";
import type {DraftTopicView,AssignmentInput} from "@/lib/taxonomy/types";
import {UiLocaleProvider,useUiI18n} from "@/lib/i18n/provider";
const member:AssignmentInput={knowledge:{id:"fractions",version:2},topicIds:["msc-00a00"],sourceBatchSHA:"b".repeat(64),sourceRefs:[{sourceId:"source-one",workFamilyId:"work-one",recordId:"original.42",path:"Foundation/record.json",sha256:"c".repeat(64)}]};
const view:DraftTopicView={draftId:"11111111-1111-4111-8111-111111111111",draftRevision:3,assignmentRevision:1,taxonomyVersionId:"a".repeat(64),members:[member],digest:"d".repeat(64),readyToSubmit:false};
it("keeps topic choices and states incomplete when assignment saving fails",async()=>{
 render(<TopicFields draftId={view.draftId} draftRevision={3} value={view} member={member} onSaved={async()=>{throw new Error("unavailable")}}/>);
 fireEvent.change(screen.getByLabelText("Specific topic IDs"),{target:{value:"msc-00a01"}});
 fireEvent.click(screen.getByRole("button",{name:"Save topic assignment"}));
 await waitFor(()=>expect(screen.getByRole("alert")).toHaveTextContent("Topic assignment was not saved."));
 expect(screen.getByLabelText("Specific topic IDs")).toHaveValue("msc-00a01");
 expect(screen.queryByText("Topic assignment saved.")).not.toBeInTheDocument();
});
it("saves the exact knowledge, source identity and current revisions without submitting a parent form",async()=>{
 let submitted=false,received:unknown;
 render(<form onSubmit={e=>{e.preventDefault();submitted=true}}><TopicFields draftId={view.draftId} draftRevision={4} value={view} member={member} onSaved={async input=>{received=input;return {...view,draftRevision:4,assignmentRevision:2,readyToSubmit:true}}}/></form>);
 fireEvent.click(screen.getByRole("button",{name:"Save topic assignment"}));
 await screen.findByText("Topic assignment saved.");
 expect(submitted).toBe(false);
 expect(received).toEqual({expectedDraftRevision:4,expectedAssignmentRevision:1,taxonomyVersionId:view.taxonomyVersionId,member});
 expect(screen.getByText(/original.42/,{selector:"p"})).toBeVisible();
});
function Switch(){const {setLocale}=useUiI18n();return <button onClick={()=>setLocale("zh-CN")}>switch</button>}
it("preserves selected topics when switching the interface language",()=>{
 render(<UiLocaleProvider initialLocale="en"><Switch/><TopicFields draftId={view.draftId} draftRevision={3} value={view} member={member} onSaved={async()=>view}/></UiLocaleProvider>);
 fireEvent.change(screen.getByLabelText("Specific topic IDs"),{target:{value:"msc-00a02"}});
 fireEvent.click(screen.getByRole("button",{name:"switch"}));
 expect(screen.getByLabelText("具体主题编号")).toHaveValue("msc-00a02");
 expect(screen.getByText(/original.42/,{selector:"p"})).toBeVisible();
});
