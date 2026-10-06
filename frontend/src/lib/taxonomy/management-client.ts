import {getAuthContext} from "../auth/client";
import {contentUUID} from "../content/schemas";
import {taxonomyRouteRequest,readTaxonomyResponse,taxonomyFailure,taxonomyPolicies,topicInputError,abortable} from "./protocol";
import type {TopicManagementRoute,TaxonomyResult,TaxonomyErrorCode} from "./types";
export async function topicManagementRequest<T>(route:TopicManagementRoute,input?:unknown,key?:string,signal?:AbortSignal):Promise<TaxonomyResult<T>>{
 const target=taxonomyRouteRequest(route);if(!target||target.path.startsWith("/api/v2/topics"))return taxonomyFailure("INVALID_REQUEST");const write=target.method!=="GET";
 if(!write&&(input!==undefined||key!==undefined))return taxonomyFailure("INVALID_REQUEST");if(write){const error=topicInputError(route.kind,input);if(error)return taxonomyFailure(error);if(key!==undefined&&!contentUUID.test(key))return taxonomyFailure("INVALID_REQUEST")}
 // 从点击开始计时，账户预检与后续网络请求共用同一预算。
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000),abort=()=>controller.abort();signal?.addEventListener("abort",abort,{once:true});
 try{if(signal?.aborted)return taxonomyFailure();const headers=new Headers({Accept:"application/json"});let body:string|undefined;
  if(write){const context=await abortable(getAuthContext(true),controller.signal);if(!context.ok){const failure=taxonomyFailure(Object.hasOwn(taxonomyPolicies,context.code)?context.code as TaxonomyErrorCode:"SERVICE_UNAVAILABLE");if(context.retryAfter!==undefined)failure.retryAfter=context.retryAfter;return failure};if(!context.data.user)return taxonomyFailure("AUTHENTICATION_REQUIRED");if(context.data.user.mustChangePassword)return taxonomyFailure("PASSWORD_CHANGE_REQUIRED");headers.set("X-CSRF-Token",context.data.csrfToken);headers.set("Content-Type","application/json");headers.set("Idempotency-Key",key??crypto.randomUUID());body=JSON.stringify(input)}
  if(controller.signal.aborted)return taxonomyFailure();const response=await fetch(target.path,{method:target.method,headers,...(body?{body}:{}),credentials:"same-origin",cache:"no-store",redirect:"error",signal:controller.signal});return (await readTaxonomyResponse(response,target.kind,controller.signal)).result as TaxonomyResult<T>;
 }catch{return taxonomyFailure()}finally{clearTimeout(timer);signal?.removeEventListener("abort",abort)}
}
