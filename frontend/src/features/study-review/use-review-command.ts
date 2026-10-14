'use client';
import {useCallback,useContext,useEffect,useRef,useState} from 'react';
import {StudyAccountContext} from '@/features/study/account-boundary';
import {makePendingReviewCommand,confirmPendingReviewCommand,type PendingReviewCommand} from '@/lib/study-review/pending-command';
import {readSession} from '@/lib/study-review/client';
import {ReviewError,type Session,type Ref} from '@/lib/study-review/types';
import type {StaticMessageKey} from '@/lib/i18n/types';
export function reviewErrorKey(e:ReviewError):StaticMessageKey {
 if(e.status===401)return 'studyReview.error.signIn';if(e.code==='PASSWORD_CHANGE_REQUIRED')return 'studyReview.error.forced';
 if(e.code==='REVIEW_REFERENCE_TOO_LARGE')return 'studyReview.error.referenceTooLarge';if(e.code==='REVIEW_MATERIAL_CHANGED')return 'studyReview.changedHint';if(e.code==='REVIEW_KNOWLEDGE_UNAVAILABLE')return 'studyReview.availableFalse';if(e.code==='REVIEW_SESSION_ACTIVE')return 'studyReview.error.active';
 if(e.status===409)return 'studyReview.error.conflict';if([400,413,422].includes(e.status))return 'studyReview.error.invalid';return 'studyReview.error.notReady';
}
export function useReviewCommand(actorId:string,onSession?:(session:Session)=>void,onIdentityLost?:()=>void){
 const boundary=useContext(StudyAccountContext),pending=useRef<PendingReviewCommand|null>(null),live=useRef(true),generation=useRef(0),locked=useRef(false),callback=useRef(onSession),lost=useRef(onIdentityLost);callback.current=onSession;lost.current=onIdentityLost;
 const[busy,setBusy]=useState(false),[authorized,setAuthorized]=useState(true),[notice,setNotice]=useState<StaticMessageKey|null>(null),[unconfirmed,setUnconfirmed]=useState(false),[confirmedRead,setConfirmedRead]=useState(false),[contentBlocked,setContentBlocked]=useState(false),[activeId,setActiveId]=useState<string|null>(null);
 const invalidate=boundary?.invalidate;
 const clearIdentity=useCallback(()=>{generation.current++;pending.current=null;locked.current=false;setBusy(false);setAuthorized(false);setNotice('studyReview.signedOut');setUnconfirmed(false);setConfirmedRead(false);setActiveId(null);setContentBlocked(true);lost.current?.();invalidate?.()},[invalidate]);
 useEffect(()=>{live.current=true;const changed=()=>clearIdentity();window.addEventListener('math-master:auth-change',changed);return()=>{live.current=false;generation.current++;pending.current=null;window.removeEventListener('math-master:auth-change',changed)}},[clearIdentity]);
 const run=useCallback(async(path?:string,method?:string,input?:unknown,knowledge?:Ref):Promise<Session|null>=>{
  if(!authorized||locked.current)return null;const job=pending.current??(path&&method?makePendingReviewCommand(actorId,path,method,input,knowledge):null);if(!job)return null;
  pending.current=job;locked.current=true;const g=generation.current;setBusy(true);setNotice(null);setActiveId(null);
  try{const result=await confirmPendingReviewCommand(job);if(!live.current||g!==generation.current)return null;
   if(!result.session){setConfirmedRead(true);setUnconfirmed(true);setNotice('studyReview.confirmedRead');return null}
   pending.current=null;setConfirmedRead(false);setUnconfirmed(false);setContentBlocked(false);callback.current?.(result.session);return result.session;
  }catch(e){if(!live.current||g!==generation.current)return null;const error=e instanceof ReviewError?e:new ReviewError('STUDY_REVIEW_NOT_READY');if(error.status===401||error.status===403){clearIdentity();return null}
   if([400,404,409,413,422].includes(error.status)){pending.current=null;setUnconfirmed(false);setConfirmedRead(false);setNotice(reviewErrorKey(error));if(error.sessionId)setActiveId(error.sessionId);
    if(error.status===409||error.status===404){setContentBlocked(true);if(job.action!=='start'){try{const current=await readSession(job.resource,actorId);if(live.current&&g===generation.current){callback.current?.(current);setContentBlocked(false)}}catch(readError){if(live.current&&g===generation.current&&readError instanceof ReviewError&&(readError.status===401||readError.status===403)){clearIdentity();return null}}}}
   }else{setUnconfirmed(true);setNotice(job.receipt?'studyReview.confirmedRead':'studyReview.uncertain');setConfirmedRead(!!job.receipt)}
   return null;
  }finally{if(live.current&&g===generation.current){locked.current=false;setBusy(false)}}
 },[actorId,authorized,clearIdentity]);
 const refresh=useCallback(async(id:string):Promise<Session|null>=>{
  if(!authorized||locked.current||pending.current)return null;locked.current=true;setBusy(true);setContentBlocked(true);const g=generation.current;
  try{const current=await readSession(id,actorId);if(!live.current||g!==generation.current)return null;callback.current?.(current);setContentBlocked(false);setNotice(null);return current}
  catch(e){if(live.current&&g===generation.current){if(e instanceof ReviewError&&(e.status===401||e.status===403))clearIdentity();else setNotice('studyReview.error.notReady')}return null}
  finally{if(live.current&&g===generation.current){locked.current=false;setBusy(false)}}
 },[actorId,authorized,clearIdentity]);
 return {run,retry:()=>run(),refresh,busy,authorized,notice,unconfirmed,confirmedRead,contentBlocked,activeId};
}
export type ReviewCommandFlow=ReturnType<typeof useReviewCommand>;
