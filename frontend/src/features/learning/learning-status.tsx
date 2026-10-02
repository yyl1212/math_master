"use client";
import Link from "next/link";
import {useCallback,useEffect,useState} from "react";
import {LearningAccountContext} from "./learning-account";
import {useRouter} from "next/navigation";
import {getAuthContext} from "@/lib/auth/client";
import type {KnowledgeState,LearningResult} from "@/lib/learning/types";
import type {useLearningCommand} from "./pending-command";
import styles from "@/styles/learning.module.css";
export function LearningStatus({state,available=true}:{state:KnowledgeState;available?:boolean}){const labels={unlearned:"Unlearned",learning:"Learning",learned:"Learned","needs-review":"Needs review",mastered:"Mastered"};return <div className={styles.status}><span data-learning-state={state.state}>{labels[state.state]}</span><span>{!available?"Unavailable":state.canEnter?"Unlocked":"Locked"}</span>{state.everUnlocked&&<small>Previously unlocked</small>}</div>}
export function LearningNotice({result}:{result:Extract<LearningResult<never>,{ok:false}>}){return <section className="content-state" role="alert"><h2>{result.code==="AUTHENTICATION_REQUIRED"?"Sign in to keep your learning records":result.code==="LEARNING_NOT_CONFIGURED"?"Learning records are not available yet":"Learning is temporarily unavailable"}</h2><p>{result.message}</p><Link prefetch={false} href={result.code==="AUTHENTICATION_REQUIRED"?"/login":"/account"} className="button secondary">{result.code==="AUTHENTICATION_REQUIRED"?"Sign in":"View account"}</Link></section>}
export function LearningBoundary({actorId,children}:{actorId:string;children:React.ReactNode}) {
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
 return <>{checking&&<p role="status">Checking your learning account…</p>}{owner===actorId&&<LearningAccountContext.Provider key={actorId} value={{actorId,invalidate}}><div hidden={checking}>{children}</div></LearningAccountContext.Provider>}</>;
}
export function CommandFeedback({command}:{command:Pick<ReturnType<typeof useLearningCommand>,"error"|"pending"|"busy"|"retry">}){const e=command.error;return <div aria-live="polite">{command.busy&&<p role="status">Saving…</p>}{e&&<div className={styles.feedback} role="alert"><p>{e.code==="SERVICE_UNAVAILABLE"?"No confirmation received. You can retry the same request.":e.message}</p>{e.formatCode&&<p>Use the format shown beside the question.</p>}{e.retryAt&&<p>Try after {new Date(e.retryAt).toLocaleString("en")}</p>}{e.activeAttempt&&<Link prefetch={false} href={(e.activeAttempt.kind==="practice"?"/practice/":"/assessments/")+e.activeAttempt.id}>Continue current attempt</Link>}{command.pending&&<button type="button" className="button secondary" disabled={command.busy} onClick={()=>void command.retry()}>Retry same request</button>}</div>}</div>}
