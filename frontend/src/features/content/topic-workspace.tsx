"use client";
import {useRef,useState} from "react";
import type {DraftView} from "@/lib/content/types";
import type {DraftTopicView,DraftTopicInput,ReleasePage,ReleaseView,PrepareInput,ActivateInput} from "@/lib/taxonomy/types";
import {topicManagementRequest} from "@/lib/taxonomy/management-client";
import {useUiI18n} from "@/lib/i18n/provider";
import {DraftEditor} from "./draft-editor";
import {TopicPublicationPanel} from "./publication-panel";
import styles from "@/styles/content.module.css";
export function TopicDraftEditor({initial,topics,canEdit}:{initial:DraftView;topics:DraftTopicView;canEdit:boolean}){
 const pending=useRef(new Map<string,{signature:string;key:string}>());
 async function save(input:DraftTopicInput){const id=input.member.knowledge.id,signature=JSON.stringify(input);if(pending.current.get(id)?.signature!==signature)pending.current.set(id,{signature,key:crypto.randomUUID()});const result=await topicManagementRequest<DraftTopicView>({kind:"saveDraft",id:initial.id},input,pending.current.get(id)!.key);if(!result.ok)throw new Error("Topic assignment failed.");pending.current.delete(id);return result.data}
 async function read(){const result=await topicManagementRequest<DraftTopicView>({kind:"readDraft",id:initial.id});if(!result.ok)throw new Error("Topic assignment unavailable.");return result.data}
 return <DraftEditor initial={initial} canEdit={canEdit} topics={topics} onSaveTopics={save} onReadTopics={read}/>
}
export function TopicPublicationWorkspace({initial}:{initial:ReleasePage}){
 const {t}=useUiI18n(),[page,setPage]=useState(initial),[selected,setSelected]=useState(initial.items[0]?.id??""),[error,setError]=useState(false),[busy,setBusy]=useState(false),[selectionEpoch,setEpoch]=useState(0);
 async function read(offset=page.offset){const result=await topicManagementRequest<ReleasePage>({kind:"listReleases",query:{limit:page.limit,offset}});if(!result.ok)throw new Error("Topic publication unavailable.");setPage(result.data);setSelected(v=>result.data.items.some(x=>x.id===v)?v:result.data.items[0]?.id??"");return result.data.pair}
 async function prepare(input:PrepareInput,key:string){const result=await topicManagementRequest<ReleaseView>({kind:"prepareRelease"},input,key);if(result.ok){setPage(v=>({...v,items:[result.data,...v.items.filter(x=>x.id!==result.data.id)],total:v.total+1}));setSelected(result.data.id)}return result}
 async function activate(id:string,input:ActivateInput,key:string){const result=await topicManagementRequest<ReleaseView>({kind:"activateRelease",id},input,key);if(result.ok)setPage(v=>({...v,pair:{knowledgeHead:result.data.knowledgePublicationId,taxonomyHead:result.data.id,taxonomyVersionId:result.data.pair.taxonomyVersionId},items:v.items.map(x=>x.id===id?result.data:x)}));return result}
 return <><TopicPublicationPanel key={selectionEpoch} pair={page.pair} initial={page.items.find(x=>x.id===selected)} onPrepare={prepare} onActivate={activate} onRefresh={()=>read()}/><section className={styles.card}><h2>{t("publication-panel.snapshot.history.d81e02",{})}</h2>{page.items.length>0&&<label className={styles.field}>{t("publication-panel.selected.snapshot.b60236",{})}<select value={selected} disabled={busy} onChange={e=>{setSelected(e.target.value);setEpoch(v=>v+1)}}>{page.items.map(p=><option key={p.id} value={p.id}>{p.status} · {p.id}</option>)}</select></label>}<div className={styles.actions}>{[Math.max(0,page.offset-page.limit),page.offset+page.limit].map((offset,i)=><button key={i} type="button" className="button secondary" disabled={busy||(i===0?page.offset===0:offset>=page.total)} onClick={async()=>{setBusy(true);setError(false);try{await read(offset);setEpoch(v=>v+1)}catch{setError(true)}finally{setBusy(false)}}}>{t(i===0?"publication-panel.previous.snapshots.1ead42":"publication-panel.next.snapshots.28f079",{})}</button>)}</div>{error&&<p role="alert">{t("common.unavailable",{})}</p>}</section></>
}
