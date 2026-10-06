import {it,expect} from "vitest";
import {importTopicDraftEnvelope} from "./topic-transfer";
import {draftInput} from "@/lib/content/test-fixtures";
it("imports exact builder sidecars without changing the legacy mathematical draft",()=>{
 const draft=draftInput(),member={knowledge:{id:draft.package.knowledge[0].id,version:1},topicIds:["msc-00a00"],sourceRefs:[{sourceId:"source",workFamilyId:"work",recordId:"original",path:"data.json",sha256:"a".repeat(64)}],sourceBatchSHA:"b".repeat(64)},envelope={kind:"topic-draft",schemaVersion:1,draft,assignments:[member],sourceBatchSHA:member.sourceBatchSHA};
 const imported=importTopicDraftEnvelope(new TextEncoder().encode(JSON.stringify(envelope)));expect(imported.draft).toEqual(draft);expect(imported.assignments).toEqual([member]);
 expect(()=>importTopicDraftEnvelope(new TextEncoder().encode(JSON.stringify({...envelope,actorId:"forged"})))).toThrow();expect(()=>importTopicDraftEnvelope(new TextEncoder().encode(JSON.stringify({...envelope,assignments:[{...member,knowledge:{id:"invented",version:1}}]})))).toThrow();
});
