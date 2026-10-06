import {z} from "zod";
import {contentPolicies,contentCanonicalJSON,contentResponseHeaders,contentUUID} from "../content/schemas";
import {readContentBytes} from "../content/bytes";
import {parseContentJSON,validContentString} from "../content/raw-json";
import {topicPageSchema,topicDetailSchema,knowledgePageSchema,draftTopicViewSchema,draftTopicInputSchema,prepareInputSchema,activateInputSchema,releaseViewSchema,releasePageSchema} from "./schemas";
import type {TaxonomyRoute,TopicManagementRoute,TaxonomyResult,TaxonomyErrorCode,TopicQuery} from "./types";
export const taxonomyPolicies={...contentPolicies,TAXONOMY_INVALID:{status:400,message:"Topic classification is invalid."},TAXONOMY_NOT_CONFIGURED:{status:503,message:"Topic classification is temporarily unavailable."},REAUTH_REQUIRED:{status:428,message:"Verify your password before continuing."}} satisfies Record<TaxonomyErrorCode,{status:number;message:string}>;
export const taxonomyFailure=(code:TaxonomyErrorCode="SERVICE_UNAVAILABLE",requestId="unavailable"):Extract<TaxonomyResult<never>,{ok:false}>=>({ok:false,code,...taxonomyPolicies[code],requestId});
export const topicIdPattern=/^msc-\d{2}(?:-\d{2}|[a-z](?:\d{2})?)?$/;
export function normalizeTopicQuery(query:TopicQuery={},kind="listTopics"):TopicQuery|null{
 if(query===null||typeof query!=="object")return null;const allowed=kind==="listTopics"?["q","parentId","kind","level","limit","offset"]:kind==="listKnowledge"?["q","limit","offset"]:["limit","offset"];
 if(Object.keys(query).some(k=>!allowed.includes(k))||query.q!==undefined&&(!validContentString(query.q)||query.q===""||new TextEncoder().encode(query.q).byteLength>512)||query.parentId!==undefined&&!topicIdPattern.test(query.parentId)||query.kind!==undefined&&!["primary","auxiliary","other"].includes(query.kind)||query.level!==undefined&&(!Number.isInteger(query.level)||query.level<1||query.level>3)||query.limit!==undefined&&(!Number.isSafeInteger(query.limit)||query.limit<1||query.limit>100)||query.offset!==undefined&&(!Number.isSafeInteger(query.offset)||query.offset<0||query.offset>100000))return null;return {...query}
}
export function taxonomyRouteRequest(route:TaxonomyRoute|TopicManagementRoute):{path:string;method:string;kind:string}|null{
 if(!route||typeof route!=="object")return null;const keys=Object.keys(route),allowed=(names:string[])=>keys.every(k=>names.includes(k));let path="",method="GET";
 if(route.kind==="listTopics"||route.kind==="listKnowledge"||route.kind==="listReleases"){
  if(!allowed(route.kind==="listKnowledge"?["kind","id","query"]:["kind","query"]))return null;
  if(route.kind==="listKnowledge"&&!topicIdPattern.test(route.id))return null;
  const q=normalizeTopicQuery(route.query,route.kind);if(!q)return null;const params=new URLSearchParams();for(const[key,value]of Object.entries(q)){if(value!==undefined)params.set(key,String(value))}
  path=route.kind==="listReleases"?"/api/v2/admin/publications":route.kind==="listKnowledge"?"/api/v2/topics/"+route.id+"/knowledge":"/api/v2/topics";if(params.size)path+="?"+params;
 }else if(route.kind==="readTopic"){
  if(!allowed(["kind","id"])||!topicIdPattern.test(route.id))return null;path="/api/v2/topics/"+route.id;
 }else if(route.kind==="prepareRelease"){
  if(!allowed(["kind"]))return null;path="/api/v2/admin/publications";method="POST";
 }else if(["readDraft","saveDraft","readSubmission","readRelease","activateRelease"].includes(route.kind)){
  if(!allowed(["kind","id"])||!("id" in route)||!contentUUID.test(route.id))return null;
  if(route.kind==="readDraft"||route.kind==="saveDraft"){path="/api/v2/content/topic-assignments/drafts/"+route.id;if(route.kind==="saveDraft")method="PUT"}
  else if(route.kind==="readSubmission")path="/api/v2/content/topic-assignments/submissions/"+route.id;
  else{path="/api/v2/admin/publications/"+route.id;if(route.kind==="activateRelease"){path+="/activate";method="POST"}}
 }else return null;return {path,method,kind:route.kind};
}
const inputs={saveDraft:draftTopicInputSchema,prepareRelease:prepareInputSchema,activateRelease:activateInputSchema};
export function topicInputError(kind:string,input:unknown):TaxonomyErrorCode|null{
 if(!Object.hasOwn(inputs,kind))return "INVALID_REQUEST";const schema=inputs[kind as keyof typeof inputs];if(!schema.safeParse(input).success)return "INVALID_REQUEST";
 if(new TextEncoder().encode(contentCanonicalJSON(input)).byteLength>8192)return "PAYLOAD_TOO_LARGE";return null;
}
const outputs={listTopics:topicPageSchema,readTopic:topicDetailSchema,listKnowledge:knowledgePageSchema,readDraft:draftTopicViewSchema,saveDraft:draftTopicViewSchema,readSubmission:draftTopicViewSchema,listReleases:releasePageSchema,readRelease:releaseViewSchema,prepareRelease:releaseViewSchema,activateRelease:releaseViewSchema};
const errorSchema=z.object({error:z.object({code:z.enum(Object.keys(taxonomyPolicies) as [TaxonomyErrorCode,...TaxonomyErrorCode[]]),message:z.string().refine(validContentString),requestId:z.string().regex(/^(?:[a-f0-9]{32}|unavailable)$/)}).strict()}).strict().refine(v=>v.error.message===taxonomyPolicies[v.error.code].message);
export async function readTaxonomyResponse(response:Response,kind:string,signal:AbortSignal):Promise<{result:TaxonomyResult<unknown>;payload:unknown}>{
 const requestId=contentResponseHeaders(response);if(response.headers.get("Cache-Control")!=="private, no-store"||!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(response.headers.get("Content-Type")??""))throw new Error("Invalid topic response.");
 const raw=parseContentJSON(await readContentBytes(response,2<<20,signal));if(!response.ok){const error=errorSchema.parse(raw).error;if(taxonomyPolicies[error.code].status!==response.status||error.requestId!==requestId)throw new Error("Invalid topic response.");const result=taxonomyFailure(error.code,requestId);if(error.code==="RATE_LIMITED"){const retry=response.headers.get("Retry-After");if(!retry||!/^(?:[1-9]|[1-5][0-9]|60)$/.test(retry))throw new Error("Invalid retry response.");result.retryAfter=Number(retry)}else if(response.headers.has("Retry-After"))throw new Error("Invalid retry response.");return {result,payload:raw}}
 if(response.status!==(kind==="prepareRelease"?201:200)||response.headers.has("Retry-After")||!Object.hasOwn(outputs,kind))throw new Error("Invalid topic response.");const parsed=outputs[kind as keyof typeof outputs].parse(raw);return {result:{ok:true,data:parsed},payload:parsed};
}
export function abortable<T>(promise:Promise<T>,signal:AbortSignal):Promise<T>{
 return new Promise((resolve,reject)=>{const fail=()=>reject(new Error("Request expired."));if(signal.aborted){fail();return};signal.addEventListener("abort",fail,{once:true});promise.then(resolve,reject).finally(()=>signal.removeEventListener("abort",fail));})
}
export function resolveTopicProxyRoute(request:Request,segments:string[],area:"public"|"assignments"|"publications"):TaxonomyRoute|TopicManagementRoute|TaxonomyErrorCode{
 let route:TaxonomyRoute|TopicManagementRoute;const [id,operation]=segments;
 if(segments.some(s=>s===""))return "NOT_FOUND";
 if(area==="public"){
  if(segments.length===0)route={kind:"listTopics"};else if(segments.length===1)route={kind:"readTopic",id};else if(segments.length===2&&operation==="knowledge")route={kind:"listKnowledge",id};else return "NOT_FOUND";
 }else if(area==="assignments"){
  if(segments.length!==2||!["drafts","submissions"].includes(id))return "NOT_FOUND";route={kind:id==="submissions"?"readSubmission":request.method==="PUT"?"saveDraft":"readDraft",id:operation};
 }else{
  if(segments.length===0)route={kind:request.method==="POST"?"prepareRelease":"listReleases"};else if(segments.length===1)route={kind:"readRelease",id};else if(segments.length===2&&operation==="activate")route={kind:"activateRelease",id};else return "NOT_FOUND";
 }
 const target=taxonomyRouteRequest(route);if(!target)return "INVALID_REQUEST";const url=new URL(request.url);if(url.pathname!==target.path.split("?")[0]||url.href.endsWith("?"))return "INVALID_REQUEST";
 try{decodeURIComponent(url.search)}catch{return "INVALID_REQUEST"}
 if(route.kind==="listTopics"||route.kind==="listKnowledge"||route.kind==="listReleases"){
  const query:Record<string,string|number>={};for(const[key,value]of url.searchParams){if(Object.hasOwn(query,key))return "INVALID_REQUEST";if(["limit","offset","level"].includes(key)){if(!/^[0-9]+$/.test(value))return "INVALID_REQUEST";query[key]=Number(value)}else query[key]=value}
  const normalized=normalizeTopicQuery(query as TopicQuery,route.kind);if(!normalized)return "INVALID_REQUEST";return {...route,query:normalized} as TaxonomyRoute|TopicManagementRoute;
 }
 if(url.search!=="")return "INVALID_REQUEST";return route;
}
