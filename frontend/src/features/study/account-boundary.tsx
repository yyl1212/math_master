"use client";
import {createContext,useCallback,useEffect,useState} from "react";
import {useRouter} from "next/navigation";
import {getAuthContext} from "@/lib/auth/client";
import {useUiI18n} from "@/lib/i18n/provider";
export const StudyAccountContext=createContext<{actorId:string;invalidate:()=>void}|null>(null);
export function StudyAccountBoundary({actorId,children}:{actorId:string;children:React.ReactNode}){
 const {t}=useUiI18n(),router=useRouter(),[owner,setOwner]=useState<string|null>(null),[checking,setChecking]=useState(true);
 const invalidate=useCallback(()=>{setOwner(null);setChecking(true);router.refresh()},[router]);
 useEffect(()=>{let live=true,generation=0;const verify=()=>{const g=++generation;setChecking(true);void getAuthContext(true).then(value=>{if(!live||g!==generation)return;if(value.ok&&value.data.user?.id===actorId&&!value.data.user.mustChangePassword){setOwner(actorId);setChecking(false)}else invalidate()}).catch(()=>{if(live&&g===generation)invalidate()})};const changed=()=>{setOwner(null);verify()},visible=()=>{if(document.visibilityState==="visible")verify()};verify();window.addEventListener("math-master:auth-change",changed);window.addEventListener("focus",verify);document.addEventListener("visibilitychange",visible);let channel:BroadcastChannel|undefined;try{if(typeof BroadcastChannel!=="undefined"){channel=new BroadcastChannel("math-master-auth");channel.onmessage=event=>{if(event.data==="changed")changed()}}}catch{};return()=>{live=false;generation++;window.removeEventListener("math-master:auth-change",changed);window.removeEventListener("focus",verify);document.removeEventListener("visibilitychange",visible);channel?.close()}},[actorId,invalidate]);
 return <>{checking&&<p role="status">{t("study.checking",{})}</p>}{owner===actorId&&<StudyAccountContext.Provider key={actorId} value={{actorId,invalidate}}><div hidden={checking}>{children}</div></StudyAccountContext.Provider>}</>;
}
