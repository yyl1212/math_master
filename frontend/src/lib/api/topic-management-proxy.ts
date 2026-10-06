import "server-only";
import {parseGoOrigin} from "./server-config";
import {parseAuthOrigin} from "../auth/config";
import {selectAuthCookies} from "../auth/cookies";
import {secretPattern} from "../auth/schemas";
import {contentUUID} from "../content/schemas";
import {readContentBytes,ContentByteLimitError} from "../content/bytes";
import {parseContentJSON} from "../content/raw-json";
import {taxonomyRouteRequest,resolveTopicProxyRoute,readTaxonomyResponse,topicInputError} from "../taxonomy/protocol";
import {topicProxyHeaders,topicProxyError} from "./taxonomy-proxy";
export function topicManagementPreflight(request:Request,segments:string[],area:"assignments"|"publications"):Response|null{
 const route=resolveTopicProxyRoute(request,segments,area);if(typeof route==="string")return topicProxyError(route);const target=taxonomyRouteRequest(route)!;if(request.method!==target.method){const response=topicProxyError("METHOD_NOT_ALLOWED");response.headers.set("Allow",area==="assignments"&&segments[0]==="drafts"?"GET, PUT":area==="publications"&&segments.length===0?"GET, POST":target.method);return response};return null;
}
export function createTopicManagementProxy(rawOrigin:string,options:{publicOrigin:string;production:boolean},fetcher:typeof fetch=fetch){
 const origin=parseGoOrigin(rawOrigin),publicOrigin=parseAuthOrigin(options.publicOrigin,options.production);
 return async(request:Request,segments:string[],area:"assignments"|"publications"):Promise<Response>=>{
  const first=topicManagementPreflight(request,segments,area);if(first)return first;const route=resolveTopicProxyRoute(request,segments,area);if(typeof route==="string")return topicProxyError(route);const target=taxonomyRouteRequest(route)!,write=target.method!=="GET";
  if(request.headers.get("Sec-Fetch-Site")==="cross-site"||write&&request.headers.get("Origin")!==publicOrigin||!write&&request.headers.has("Origin")&&request.headers.get("Origin")!==publicOrigin||write&&!secretPattern.test(request.headers.get("X-CSRF-Token")??""))return topicProxyError("CSRF_FAILED");
  if(write&&!contentUUID.test(request.headers.get("Idempotency-Key")??""))return topicProxyError("INVALID_REQUEST");
  const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000),abort=()=>controller.abort();request.signal.addEventListener("abort",abort,{once:true});
  try{if(request.signal.aborted)return topicProxyError();const headers=new Headers({Accept:"application/json"}),cookie=selectAuthCookies(request.headers.get("Cookie")??"",options.production,true);if(cookie)headers.set("Cookie",cookie);for(const name of ["Origin","X-CSRF-Token","Sec-Fetch-Site"]){const value=request.headers.get(name);if(value!==null)headers.set(name,value)}
   let body:ArrayBuffer|undefined;
   if(write){if(!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(request.headers.get("Content-Type")??""))return topicProxyError("INVALID_REQUEST");headers.set("Content-Type","application/json");headers.set("Idempotency-Key",request.headers.get("Idempotency-Key")!);
    let bytes:Uint8Array;try{bytes=await readContentBytes(new Response(request.body,{headers:request.headers.has("Content-Length")?{"Content-Length":request.headers.get("Content-Length")!}:{}}),8192,controller.signal)}catch(error){return topicProxyError(controller.signal.aborted?"SERVICE_UNAVAILABLE":error instanceof ContentByteLimitError?"PAYLOAD_TOO_LARGE":"INVALID_REQUEST")}
    try{const error=topicInputError(route.kind,parseContentJSON(bytes));if(error)return topicProxyError(error)}catch{return topicProxyError("INVALID_REQUEST")};body=bytes.slice().buffer as ArrayBuffer;
   }else if(request.body!==null||request.headers.has("Idempotency-Key"))return topicProxyError("INVALID_REQUEST");
   const response=await fetcher(origin+target.path,{method:target.method,headers,...(body?{body}:{}),cache:"no-store",redirect:"error",signal:controller.signal}),parsed=await readTaxonomyResponse(response,target.kind,controller.signal),output=topicProxyHeaders();output.set("X-Request-ID",response.headers.get("X-Request-ID")!);if(!parsed.result.ok&&parsed.result.retryAfter!==undefined)output.set("Retry-After",String(parsed.result.retryAfter));return Response.json(parsed.payload,{status:response.status,headers:output})
  }catch{return topicProxyError()}finally{clearTimeout(timer);request.signal.removeEventListener("abort",abort)}
 }
}
