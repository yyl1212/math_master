import {getAuthContext} from "../auth/client";
import {contentUUID} from "../content/schemas";
import {abortable} from "../taxonomy/protocol";
import {studyRouteRequest,readStudyResponse,studyFailure,studyPolicies,studyInputError} from "./protocol";
import type {StudyRoute,StudyResult,StudyErrorCode} from "./types";
export async function studyRequest<T>(route:StudyRoute,input?:unknown,key?:string,actorId?:string,signal?:AbortSignal):Promise<StudyResult<T>>{
 const target=studyRouteRequest(route);if(!target)return studyFailure("INVALID_REQUEST");const write=target.method!=="GET";
 if(!write&&(input!==undefined||key!==undefined))return studyFailure("INVALID_REQUEST");if(write){const error=studyInputError(route.kind,input);if(error)return studyFailure(error);if(!key||!contentUUID.test(key)||!actorId||!contentUUID.test(actorId))return studyFailure("INVALID_REQUEST")}
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000),abort=()=>controller.abort();signal?.addEventListener("abort",abort,{once:true});
 try{if(signal?.aborted)return studyFailure();const context=await abortable(getAuthContext(true),controller.signal);if(!context.ok){const failure=studyFailure(Object.hasOwn(studyPolicies,context.code)?context.code as StudyErrorCode:"SERVICE_UNAVAILABLE");if(context.retryAfter!==undefined)failure.retryAfter=context.retryAfter;return failure};const user=context.data.user;if(!user||actorId!==undefined&&user.id!==actorId)return studyFailure("AUTHENTICATION_REQUIRED");if(user.mustChangePassword)return studyFailure("PASSWORD_CHANGE_REQUIRED");const headers=new Headers({Accept:"application/json"});let body:string|undefined;if(write){headers.set("X-CSRF-Token",context.data.csrfToken);headers.set("Content-Type","application/json");headers.set("Idempotency-Key",key!);body=JSON.stringify(input)}
 const response=await fetch(target.path,{method:target.method,headers,...(body!==undefined?{body}:{}),credentials:"same-origin",cache:"no-store",redirect:"error",signal:controller.signal});const result=(await readStudyResponse(response,target.kind,controller.signal)).result as StudyResult<T>;
 if(result.ok&&(!result.data||typeof result.data!=="object"||!("actorId"in result.data)||result.data.actorId!==user.id))return studyFailure("AUTHENTICATION_REQUIRED");return result;
 }catch{return studyFailure()}finally{clearTimeout(timer);signal?.removeEventListener("abort",abort)}
}
