import 'server-only';
import {getGoOrigin} from '../api/server-config';
import {getAuthConfig} from '../auth/config';
import {selectAuthCookies} from '../auth/cookies';
import {readServerSession} from '../auth/server-client';
import {boundedBytes,withinSignal} from '../note-sharing/response';
import {NoteShareError} from '../note-sharing/types';
import {routeForReview,schemaForReviewPath,reviewProblem} from './schemas';
import {ReviewError,type ReviewResult} from './types';
export async function readReview<T>(path:string,cookie=''):Promise<ReviewResult<T>>{
 const route=routeForReview(path.split('?')[0]);if(!route||route.method!=='GET')return {ok:false,status:400,code:'INVALID_REQUEST'};
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);
 try{
  let actor:string|undefined,selected='';if(route.private){const user=await withinSignal(readServerSession(cookie),controller.signal);if(!user.ok)return {ok:false,status:user.status,code:user.code};if(!user.data)return {ok:false,status:401,code:'AUTHENTICATION_REQUIRED'};if(user.data.mustChangePassword)return {ok:false,status:403,code:'PASSWORD_CHANGE_REQUIRED'};actor=user.data.id;selected=selectAuthCookies(cookie,getAuthConfig().production,true)}
  const response=await withinSignal(fetch(getGoOrigin()+path,{cache:'no-store',redirect:'error',signal:controller.signal,headers:{Accept:'application/json',...(selected?{Cookie:selected}:{})}}),controller.signal);
  let bytes:Uint8Array;try{bytes=await boundedBytes(response,1048576,controller.signal)}catch(e){if(e instanceof NoteShareError&&e.status===413)throw new ReviewError('STUDY_REVIEW_NOT_READY');throw e}
  const raw=JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes)) as unknown;if(!response.ok){const problem=reviewProblem.safeParse(raw);return {ok:false,status:response.status,code:problem.success?problem.data.error.code:'STUDY_REVIEW_NOT_READY'}}
  const value=schemaForReviewPath(path).parse(raw);
  if(actor){if(!('actorId' in value)||value.actorId!==actor)return {ok:false,status:401,code:'AUTHENTICATION_REQUIRED'};const current=await withinSignal(readServerSession(cookie),controller.signal);if(!current.ok||!current.data||current.data.id!==actor||current.data.mustChangePassword)return {ok:false,status:401,code:'AUTHENTICATION_REQUIRED'}}
  return {ok:true,data:value as T};
 }catch(e){return {ok:false,status:e instanceof ReviewError?e.status:503,code:controller.signal.aborted?'REQUEST_INTERRUPTED':e instanceof ReviewError?e.code:'STUDY_REVIEW_NOT_READY'}}finally{clearTimeout(timer)}
}

export const readNumberedReview=readReview;
