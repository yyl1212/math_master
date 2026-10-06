import {it,expect} from "vitest";
import {studyRouteRequest,studyInputError,resolveStudyRoute} from "./protocol";
it("allows only pinned private study routes with byte bounded literal queries",()=>{
 expect(studyRouteRequest({kind:"knowledge",query:{q:"中文%_",reviewOnly:true}})?.path).toContain("reviewOnly=true");
 for(const route of [{kind:"unknown"},{kind:"begin",id:"../admin"},{kind:"overview",actorId:"forged"},{kind:"knowledge",query:{q:"中".repeat(171)}},{kind:"history",query:{limit:51}}])expect(studyRouteRequest(route as never)).toBeNull();
 expect(resolveStudyRoute(new Request("http://local/api/v2/study/knowledge?q=a&q=b"))).toBe("INVALID_REQUEST");
 expect(resolveStudyRoute(new Request("http://local/api/v2/study/knowledge?q=%FF"))).toBe("INVALID_REQUEST");
});
it("separates control and note envelopes and rejects forged actor inputs",()=>{
 const input={knowledge:{id:"fractions",version:1,sha256:"a".repeat(64)},expectedKnowledgeHead:"11111111-1111-4111-8111-111111111111",expectedSequence:0};
 expect(studyInputError("begin",input)).toBeNull();expect(studyInputError("begin",{...input,actorId:"forged"})).toBe("INVALID_REQUEST");
 expect(studyInputError("saveNote",{knowledge:input.knowledge,expectedRevision:0,body:"😀".repeat(16000)})).toBeNull();
 expect(studyInputError("saveNote",{knowledge:input.knowledge,expectedRevision:0,body:"😀".repeat(16001)})).toBe("INVALID_REQUEST");
});
