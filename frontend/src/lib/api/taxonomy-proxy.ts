import "server-only";
import {parseGoOrigin} from "./server-config";
import {taxonomyFailure,taxonomyRouteRequest,resolveTopicProxyRoute,readTaxonomyResponse} from "../taxonomy/protocol";
import type {TaxonomyErrorCode} from "../taxonomy/types";
export const topicProxyHeaders=()=>new Headers({"Cache-Control":"private, no-store","Content-Type":"application/json","X-Content-Type-Options":"nosniff","X-Request-ID":"unavailable"});
export function topicProxyError(code:TaxonomyErrorCode="SERVICE_UNAVAILABLE"):Response{const result=taxonomyFailure(code);return Response.json({error:{code:result.code,message:result.message,requestId:result.requestId}},{status:result.status,headers:topicProxyHeaders()})}
export function createTaxonomyProxy(rawOrigin:string,fetcher:typeof fetch=fetch){
 const origin=parseGoOrigin(rawOrigin);
 return async(request:Request,segments:string[]):Promise<Response>=>{
  const route=resolveTopicProxyRoute(request,segments,"public");if(typeof route==="string")return topicProxyError(route);const target=taxonomyRouteRequest(route)!;
  if(request.method!=="GET"){const r=topicProxyError("METHOD_NOT_ALLOWED");r.headers.set("Allow","GET");return r}if(request.body!==null)return topicProxyError("INVALID_REQUEST");
  const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000),abort=()=>controller.abort();request.signal.addEventListener("abort",abort,{once:true});
  try{if(request.signal.aborted)return topicProxyError();const response=await fetcher(origin+target.path,{method:"GET",headers:{Accept:"application/json"},cache:"no-store",redirect:"error",signal:controller.signal}),parsed=await readTaxonomyResponse(response,target.kind,controller.signal),headers=topicProxyHeaders();headers.set("X-Request-ID",response.headers.get("X-Request-ID")!);if(!parsed.result.ok&&parsed.result.retryAfter!==undefined)headers.set("Retry-After",String(parsed.result.retryAfter));return Response.json(parsed.payload,{status:response.status,headers})}catch{return topicProxyError()}finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort)}
 }
}
