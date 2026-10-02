"use client";
import {useEffect,useRef,useState} from "react";
import {getAuthContext} from "@/lib/auth/client";
import {requestLearning,bindLearningInput} from "@/lib/learning/client";
import {validateLearningBytes,LearningInputError} from "@/lib/learning/bytes";
import {learningFailure,learningRouteRequest,learningUUID} from "@/lib/learning/schemas";
import type {LearningResult,LearningRoute} from "@/lib/learning/types";
export type PendingLearningCommand={key:string;route:LearningRoute;input:Readonly<unknown>;actorId:string};
function freeze<T>(v:T):T{if(v!==null&&typeof v==="object"){for(const item of Object.values(v))freeze(item);Object.freeze(v)}return v}
export function createPendingLearningCommand(route:LearningRoute,input:unknown,actorId:string):PendingLearningCommand{if(!learningUUID.test(actorId)||learningRouteRequest(route)?.method!=="POST")throw new Error("Invalid learning command.");const raw=JSON.stringify(input);validateLearningBytes(new TextEncoder().encode(raw),route.kind);const value=freeze(JSON.parse(raw));bindLearningInput(value,actorId);return Object.freeze({key:crypto.randomUUID(),route:freeze(structuredClone(route)),input:value,actorId})}
export const pendingForActor=(command:PendingLearningCommand|null,actorId:string|null)=>command?.actorId===actorId?command:null;
export function useLearningCommand<T>(onSuccess:(value:T)=>void){const[pending,setPending]=useState<PendingLearningCommand|null>(null),[error,setError]=useState<Extract<LearningResult<never>,{ok:false}>|null>(null),[busy,setBusy]=useState(false);const inFlight=useRef(false),live=useRef(true),generation=useRef(0),controller=useRef<AbortController|null>(null),success=useRef(onSuccess);success.current=onSuccess;
 useEffect(()=>{live.current=true;const clear=()=>{generation.current++;controller.current?.abort();inFlight.current=false;setPending(null);setError(null);setBusy(false)};window.addEventListener("math-master:auth-change",clear);return()=>{live.current=false;generation.current++;controller.current?.abort();window.removeEventListener("math-master:auth-change",clear)}},[]);
 const clear=()=>{generation.current++;controller.current?.abort();inFlight.current=false;setPending(null);setError(null);setBusy(false)};
 const send=async(command:PendingLearningCommand,g:number)=>{if(!live.current||generation.current!==g)return;setPending(command);setError(null);controller.current=new AbortController();const result=await requestLearning<T>(command.route,command.input,command.key,controller.current.signal);if(!live.current||generation.current!==g)return;if(result.ok){setPending(null);success.current(result.data)}else setError(result);inFlight.current=false;setBusy(false)};
 const run=async(route:LearningRoute,input:unknown)=>{if(inFlight.current)return;inFlight.current=true;const g=++generation.current;setBusy(true);setError(null);try{const context=await getAuthContext();if(!live.current||generation.current!==g)return;if(!context.ok||context.data.user===null){setError(learningFailure(context.ok?"AUTHENTICATION_REQUIRED":context.code));inFlight.current=false;setBusy(false);return};await send(createPendingLearningCommand(route,input,context.data.user.id),g)}catch(e){if(live.current&&generation.current===g){setError(learningFailure(e instanceof LearningInputError?e.code:undefined));inFlight.current=false;setBusy(false)}}};
 const retry=async()=>{if(!pending||inFlight.current)return;inFlight.current=true;setBusy(true);await send(pending,++generation.current)};
 return {run,retry,clear,busy,pending,error};
}
