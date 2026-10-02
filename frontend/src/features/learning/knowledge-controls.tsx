"use client";
import Link from "next/link";
import {useEffect,useState} from "react";
import {useRouter} from "next/navigation";
import type {KnowledgeDetail,KnowledgeState} from "@/lib/learning/types";
import {useLearningCommand} from "./pending-command";
import {LearningStatus,CommandFeedback} from "./learning-status";
import styles from "@/styles/learning.module.css";
export function KnowledgeControls({detail,children}:{detail:KnowledgeDetail;children?:React.ReactNode}){const[state,setState]=useState(detail.state),router=useRouter();useEffect(()=>setState(detail.state),[detail.state]);const command=useLearningCommand<KnowledgeState>(v=>{setState(v);router.refresh()});const input={knowledge:state.knowledge,expectedKnowledgeHead:detail.knowledgeHead};const historical=detail.objectives.length===0&&detail.questionHead===null;
 return <section className={styles.panel} aria-label="Personal knowledge record"><h2>{historical?"Historical learning record":"Your learning record"}</h2><LearningStatus state={state}/>{historical?<p>This is a saved version. <Link prefetch={false} href={"/knowledge/"+state.knowledge.id}>Review the current explanation</Link>.</p>:<><p>Read freely. Your learning record changes when you choose an action below.</p>{state.canEnter?<div className={styles.actions}>{state.startedAt===null?<button className="button" disabled={command.busy} onClick={()=>void command.run({kind:"startKnowledge",id:state.knowledge.id},input)}>Start learning</button>:!state.completionValid?<button className="button" disabled={command.busy} onClick={()=>void command.run({kind:"completeKnowledge",id:state.knowledge.id},input)}>Mark as learned</button>:<span>Reading completion recorded.</span>}</div>:<p>Work through the prerequisites, or use a diagnostic assessment to check your understanding.</p>}{state.prerequisites.length>0&&<ul>{state.prerequisites.map(p=><li key={p.knowledge.id}><Link prefetch={false} href={"/knowledge/"+p.knowledge.id}>{p.knowledge.id.replaceAll("-"," ")}</Link> · {p.qualified?"Ready":"Needs assessment"}</li>)}</ul>}{children}</>}
 <Link prefetch={false} href={`/learning-history?knowledge=${state.knowledge.id}&version=${state.knowledge.version}`}>View learning history</Link><CommandFeedback command={command}/></section>
}
