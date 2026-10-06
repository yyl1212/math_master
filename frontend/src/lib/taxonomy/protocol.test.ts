import {it,expect} from "vitest";
import {taxonomyRouteRequest,readTaxonomyResponse,topicInputError} from "./protocol";
const pair={knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)};
const headers={"Content-Type":"application/json","X-Request-ID":"a".repeat(32),"Cache-Control":"private, no-store"};
it("bounds literal UTF-8 queries and rejects arbitrary routes",()=>{
 expect(taxonomyRouteRequest({kind:"listTopics",query:{q:"汉".repeat(170)+"aa",limit:100}})?.path).toContain("/api/v2/topics?q=");
 expect(taxonomyRouteRequest({kind:"listTopics",query:{q:"汉".repeat(171)}})).toBeNull();
 expect(taxonomyRouteRequest({kind:"readTopic",id:"../private"})).toBeNull();
 expect(taxonomyRouteRequest({kind:"listTopics",query:{limit:101}})).toBeNull();
});
it("accepts a bounded topic page and rejects private proof, unknown fields and foreign cookies",async()=>{
 const page={items:[],total:63,limit:20,offset:0,pair},signal=new AbortController().signal;
 expect((await readTaxonomyResponse(new Response(JSON.stringify(page),{headers}),"listTopics",signal)).result).toEqual({ok:true,data:page});
 for(const response of [new Response(JSON.stringify({...page,sourceRefs:[]}),{headers}),new Response(JSON.stringify(page),{headers:{...headers,"Set-Cookie":"foreign=private"}}),new Response('x'.repeat((2<<20)+1),{headers})]){await expect(readTaxonomyResponse(response,"listTopics",signal)).rejects.toThrow()}
});
it("maps fixed taxonomy errors and does not accept backend diagnostic messages",async()=>{
 const body={error:{code:"TAXONOMY_NOT_CONFIGURED",message:"Topic classification is temporarily unavailable.",requestId:"a".repeat(32)}},signal=new AbortController().signal;
 expect((await readTaxonomyResponse(new Response(JSON.stringify(body),{status:503,headers}),"listTopics",signal)).result.ok).toBe(false);
 body.error.message="private SQL credential";await expect(readTaxonomyResponse(new Response(JSON.stringify(body),{status:503,headers}),"listTopics",signal)).rejects.toThrow();
});
it("control requests include Go escaping within the 8 KiB budget",()=>{
 const value={submissionIds:[],expectedPair:pair,reason:"<".repeat(1000)};
 expect(topicInputError("prepareRelease",value)).toBeNull();
 expect(topicInputError("prepareRelease",{...value,actorId:"forged"})).toBe("INVALID_REQUEST");
});
