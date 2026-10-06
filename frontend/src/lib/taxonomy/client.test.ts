import {beforeEach,it,expect,vi} from "vitest";
import {taxonomyRequest} from "./client";
import {topicManagementRequest} from "./management-client";
import {fixtureID,token,requestID} from "../content/test-fixtures";
const mocks=vi.hoisted(()=>({context:vi.fn()}));vi.mock("../auth/client",()=>({getAuthContext:mocks.context}));
beforeEach(()=>{vi.restoreAllMocks();vi.useRealTimers();mocks.context.mockReset()});
it("public reads omit browser credentials and reject private routes",async()=>{
 const fake=vi.spyOn(globalThis,"fetch").mockResolvedValue(new Response(JSON.stringify({items:[],total:63,limit:20,offset:0,pair:{knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)}}),{headers:{"Content-Type":"application/json","Cache-Control":"private, no-store","X-Request-ID":requestID}}));
 expect((await taxonomyRequest({kind:"listTopics"})).ok).toBe(true);expect(fake.mock.calls[0][1]?.credentials).toBe("omit");
 expect((await taxonomyRequest({kind:"readDraft",id:fixtureID} as never)).ok).toBe(false);expect(fake).toHaveBeenCalledTimes(1);
});
it("counts authentication and publication within one ten-second click budget",async()=>{
 vi.useFakeTimers();mocks.context.mockImplementation(()=>new Promise(resolve=>setTimeout(()=>resolve({ok:true,data:{user:{id:fixtureID,mustChangePassword:false},csrfToken:token}}),9000)));
 const fake=vi.spyOn(globalThis,"fetch").mockImplementation(async(_,init)=>new Promise((_,reject)=>{init?.signal?.addEventListener("abort",()=>reject(new Error("expired")))}));
 const pending=topicManagementRequest({kind:"prepareRelease"},{submissionIds:[],expectedPair:{knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)},reason:"Prepare an isolated technical catalogue."},fixtureID);
 await vi.advanceTimersByTimeAsync(10001);expect((await pending).ok).toBe(false);expect(fake).toHaveBeenCalledTimes(1);vi.useRealTimers();
});
