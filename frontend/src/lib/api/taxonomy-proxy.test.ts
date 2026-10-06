import {it,expect} from "vitest";
import {createTaxonomyProxy} from "./taxonomy-proxy";
import {createTopicManagementProxy} from "./topic-management-proxy";
import {fixtureID,token,requestID} from "../content/test-fixtures";
const headers={"Content-Type":"application/json","X-Request-ID":requestID,"Cache-Control":"private, no-store"},pair={knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)};
const member={knowledge:{id:"fractions",version:1},topicIds:["msc-00a00"],sourceRefs:[{sourceId:"source",workFamilyId:"work",recordId:"original",path:"Original/data.json",sha256:"b".repeat(64)}],sourceBatchSHA:"c".repeat(64)};
it("forwards only a fixed public route and never identity headers",async()=>{
 let sent:RequestInit|undefined,target="";const proxy=createTaxonomyProxy("http://127.0.0.1:8080",async(url,options)=>{target=String(url);sent=options;return new Response(JSON.stringify({items:[],total:63,limit:20,offset:0,pair}),{headers})});
 const response=await proxy(new Request("http://127.0.0.1:3000/api/v2/topics?q=%E6%B1%89",{headers:{Cookie:`mm_session_dev=${token}`,Authorization:"private"}}),[]);
 expect(response.status).toBe(200);expect(target).toBe("http://127.0.0.1:8080/api/v2/topics?q=%E6%B1%89");expect(new Headers(sent?.headers).has("Cookie")).toBe(false);expect(new Headers(sent?.headers).has("Authorization")).toBe(false);
 expect((await proxy(new Request("http://127.0.0.1:3000/api/v2/topics?limit=20&limit=21"),[])).status).toBe(400);
 expect((await proxy(new Request("http://127.0.0.1:3000/api/v2/topics/msc-00/arbitrary"),["msc-00","arbitrary"])).status).toBe(404);
});
it("validates management bodies and forwards only the selected session and proof",async()=>{
 let sent:RequestInit|undefined,calls=0;const proxy=createTopicManagementProxy("http://127.0.0.1:8080",{publicOrigin:"http://127.0.0.1:3000",production:false},async(_,options)=>{calls++;sent=options;return new Response(JSON.stringify({draftId:fixtureID,draftRevision:1,assignmentRevision:1,taxonomyVersionId:pair.taxonomyVersionId,members:[member],digest:"d".repeat(64),readyToSubmit:true}),{headers})});
 const request=(body:string)=>new Request("http://127.0.0.1:3000/api/v2/content/topic-assignments/drafts/"+fixtureID,{method:"PUT",body,headers:{Origin:"http://127.0.0.1:3000","Content-Type":"application/json","X-CSRF-Token":token,"Idempotency-Key":fixtureID,Cookie:`foreign=private; mm_session_dev=${token}; mm_preauth_dev=${token}`,Authorization:"private"}});
 const input={expectedDraftRevision:1,expectedAssignmentRevision:0,taxonomyVersionId:pair.taxonomyVersionId,member};
 expect((await proxy(request(JSON.stringify({...input,actorId:"forged"})),["drafts",fixtureID],"assignments")).status).toBe(400);expect(calls).toBe(0);
 expect((await proxy(request(JSON.stringify(input)),["drafts",fixtureID],"assignments")).status).toBe(200);expect(new Headers(sent?.headers).get("Cookie")).toBe(`mm_session_dev=${token}`);expect(new Headers(sent?.headers).has("Authorization")).toBe(false);expect(new Headers(sent?.headers).get("Idempotency-Key")).toBe(fixtureID);
});
