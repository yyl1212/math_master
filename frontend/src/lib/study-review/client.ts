import {getAuthContext} from '../auth/client';
import {boundedBytes,withinSignal} from '../note-sharing/response';
import {NoteShareError} from '../note-sharing/types';
import {routeForReview,schemaForReviewPath,reviewProblem} from './schemas';
import {ReviewError,type CommandReceipt,type Session} from './types';
export async function reviewRequest<T>(path:string,method='GET',input?:unknown,key?:string,expectedActor?:string):Promise<T>{
 const route=routeForReview(path.split('?')[0]);if(!route||route.method!==method)throw new ReviewError('INVALID_REQUEST',400);
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);
 try{
  const headers=new Headers({Accept:'application/json'});let actor:string|undefined;
  if(route.private){const context=await withinSignal(getAuthContext(true),controller.signal);if(!context.ok)throw new ReviewError(context.code,context.status);if(!context.data.user)throw new ReviewError('AUTHENTICATION_REQUIRED',401);if(context.data.user.mustChangePassword)throw new ReviewError('PASSWORD_CHANGE_REQUIRED',403);actor=context.data.user.id;if(expectedActor&&actor!==expectedActor)throw new ReviewError('AUTHENTICATION_REQUIRED',401);
   if(method!=='GET'){if(!key)throw new ReviewError('INVALID_REQUEST',400);headers.set('Content-Type','application/json');headers.set('X-CSRF-Token',context.data.csrfToken);headers.set('Idempotency-Key',key)}
  }
  const body=method!=='GET'?JSON.stringify(input):undefined;if(body&&new TextEncoder().encode(body).length>131072)throw new ReviewError('INPUT_TOO_LARGE',413);
  const response=await withinSignal(fetch(path,{method,headers,...(body!==undefined?{body}:{}),credentials:route.private?'same-origin':'omit',cache:'no-store',redirect:'error',signal:controller.signal}),controller.signal);
  let bytes:Uint8Array;try{bytes=await boundedBytes(response,1048576,controller.signal)}catch(e){if(e instanceof NoteShareError&&e.status===413)throw new ReviewError('STUDY_REVIEW_NOT_READY');throw e}
  const raw=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes)) as unknown;
  if(!response.ok){const problem=reviewProblem.safeParse(raw);throw new ReviewError(problem.success?problem.data.error.code:'STUDY_REVIEW_NOT_READY',response.status,problem.success?problem.data.error.sessionId:undefined)}
  const value=schemaForReviewPath(path,method).parse(raw);
  if(route.private){if(!('actorId' in value)||value.actorId!==actor)throw new ReviewError('AUTHENTICATION_REQUIRED',401);const current=await withinSignal(getAuthContext(true),controller.signal);if(!current.ok||!current.data.user||current.data.user.id!==actor||current.data.user.mustChangePassword)throw new ReviewError('AUTHENTICATION_REQUIRED',401)}
  return value as T;
 }catch(e){if(e instanceof ReviewError)throw e;throw new ReviewError(controller.signal.aborted?'REQUEST_INTERRUPTED':'STUDY_REVIEW_NOT_READY')}
 finally{clearTimeout(timer)}
}
export function reviewCommand(path:string,method:string,input:unknown,key:string,actorId:string){return reviewRequest<CommandReceipt>(path,method,input,key,actorId)}
export function readSession(id:string,actorId?:string){return reviewRequest<Session>('/api/v3/study/review/sessions/'+id,'GET',undefined,undefined,actorId)}
