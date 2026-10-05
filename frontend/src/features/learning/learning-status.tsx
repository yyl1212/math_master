"use client";
import {UiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import {useCallback,useEffect,useState} from "react";
import {LearningAccountContext} from "./learning-account";
import {useRouter} from "next/navigation";
import {getAuthContext} from "@/lib/auth/client";
import type {KnowledgeState,LearningResult} from "@/lib/learning/types";
import type {useLearningCommand} from "./pending-command";
import styles from "@/styles/learning.module.css";
export function LearningStatus({state,available=true}:{state:KnowledgeState;available?:boolean}){
 const {t}=useUiI18n();
const labels={unlearned:"Unlearned",learning:"Learning",learned:"Learned","needs-review":"Needs review",mastered:"Mastered"};return <div className={styles.status}><span data-learning-state={state.state}><UiEnum group="learning.state" value={state.state}/></span><span>{!available?t("learning-status.unavailable.ca1844",{}):state.canEnter?t("learning-status.unlocked.531c7b",{}):t("learning-status.locked.a424e3",{})}</span>{state.everUnlocked&&<small><UiText notice={uiMessage("learning-status.previously.unlocked.ecd216",{})}/></small>}</div>}
export function LearningNotice({result}:{result:Extract<LearningResult<never>,{ok:false}>}){
 const {t}=useUiI18n();
return <section className="content-state" role="alert"><h2>{result.code==="AUTHENTICATION_REQUIRED"?t("learning-status.sign.in.to.keep.your.learning.records.c763b1",{}):result.code==="LEARNING_NOT_CONFIGURED"?t("learning-status.learning.records.are.not.available.yet.d0dd91",{}):t("learning-status.learning.is.temporarily.unavailable.5d2180",{})}</h2><p><UiText notice={uiError("learning",result)}/></p><Link prefetch={false} href={result.code==="AUTHENTICATION_REQUIRED"?"/login":"/account"} className="button secondary">{result.code==="AUTHENTICATION_REQUIRED"?t("page.login",{}):t("auth-state.view.account.407143",{})}</Link></section>}
export function LearningBoundary({actorId,children}:{actorId:string;children:React.ReactNode}) {
 const {t}=useUiI18n();

 const router=useRouter(),[owner,setOwner]=useState<string|null>(null),[checking,setChecking]=useState(true);
 const invalidate=useCallback(()=>{setOwner(null);setChecking(true);router.refresh()},[router]);
 useEffect(()=>{
  let live=true,revision=0;
  const verify=()=>{
   const current=++revision;setChecking(true);
   void getAuthContext(true).then(v=>{
    if(!live||revision!==current)return;
    if(v.ok&&v.data.user?.id===actorId&&!v.data.user.mustChangePassword){setOwner(actorId);setChecking(false)}
    else invalidate();
   });
  };
  const changed=()=>{setOwner(null);verify()};
  const visible=()=>{if(document.visibilityState==="visible")verify()};
  verify();
  window.addEventListener("math-master:auth-change",changed);
  window.addEventListener("focus",verify);
  document.addEventListener("visibilitychange",visible);
  let channel:BroadcastChannel|undefined;
  if(typeof BroadcastChannel!=="undefined"){
   try{
    channel=new BroadcastChannel("math-master-auth");
    channel.onmessage=e=>{if(e.data==="changed")window.dispatchEvent(new Event("math-master:auth-change"))};
   }catch{/* Command-time identity checks remain mandatory. */}
  }
  return()=>{live=false;window.removeEventListener("math-master:auth-change",changed);window.removeEventListener("focus",verify);document.removeEventListener("visibilitychange",visible);channel?.close()};
 },[actorId,invalidate]);
 return <>{checking&&<p role="status"><UiText notice={uiMessage("learning-status.checking.your.learning.account.3df2f8",{})}/></p>}{owner===actorId&&<LearningAccountContext.Provider key={actorId} value={{actorId,invalidate}}><div hidden={checking}>{children}</div></LearningAccountContext.Provider>}</>;
}
export function CommandFeedback({command}:{command:Pick<ReturnType<typeof useLearningCommand>,"error"|"pending"|"busy"|"retry">}){
 const {t}=useUiI18n();
const e=command.error;return <div aria-live="polite">{command.busy&&<p role="status"><UiText notice={uiMessage("learning-status.saving.23e392",{})}/></p>}{e&&<div className={styles.feedback} role="alert"><p>{e.code==="SERVICE_UNAVAILABLE"?t("learning-status.no.confirmation.received.you.can.retry.the.same.request.e68eb0",{}):<UiText notice={uiError("learning",e)}/>}</p>{e.formatCode&&<p><UiText notice={uiMessage("learning-status.use.the.format.shown.beside.the.question.949e24",{})}/></p>}{e.retryAt&&<p><UiText notice={uiMessage("learning-status.try.after.value.ea24ae",{v0:uiValue(new Date(e.retryAt).toLocaleString("en"))})}/></p>}{e.activeAttempt&&<Link prefetch={false} href={(e.activeAttempt.kind==="practice"?"/practice/":"/assessments/")+e.activeAttempt.id}><UiText notice={uiMessage("learning-status.continue.current.attempt.9545db",{})}/></Link>}{command.pending&&<button type="button" className="button secondary" disabled={command.busy} onClick={()=>void command.retry()}><UiText notice={uiMessage("learning-status.retry.same.request.16003a",{})}/></button>}</div>}</div>}
