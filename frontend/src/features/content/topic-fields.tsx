"use client";
import {useState,useRef} from "react";
import type {DraftTopicView,DraftTopicInput,AssignmentInput} from "@/lib/taxonomy/types";
import {assignmentInputSchema} from "@/lib/taxonomy/schemas";
import {useUiI18n} from "@/lib/i18n/provider";
import styles from "@/styles/content.module.css";
type Props={draftId:string;draftRevision:number;value:DraftTopicView;member:AssignmentInput;disabled?:boolean;onDirty?:()=>void;onSaved:(input:DraftTopicInput)=>Promise<DraftTopicView>};
export function TopicFields({draftRevision,value,member,disabled,onSaved,onDirty}:Props){
 const {t}=useUiI18n(),[ids,setIds]=useState(member.topicIds.join(", ")),[version,setVersion]=useState(value.taxonomyVersionId),[batch,setBatch]=useState(member.sourceBatchSHA),[sources,setSources]=useState(JSON.stringify(member.sourceRefs,null,2)),[busy,setBusy]=useState(false),[notice,setNotice]=useState<"saved"|"failed"|"invalid"|null>(null),pending=useRef(false);
 async function save(){
  if(pending.current||disabled)return;
  let next:AssignmentInput;
  try{next=assignmentInputSchema.parse({...member,topicIds:ids.split(",").map(v=>v.trim()).filter(Boolean),sourceBatchSHA:batch,sourceRefs:JSON.parse(sources)});if(!/^[a-f0-9]{64}$/.test(version))throw new Error("invalid");}catch{setNotice("invalid");return}
  pending.current=true;setBusy(true);setNotice(null);
  try{await onSaved({expectedDraftRevision:draftRevision,expectedAssignmentRevision:value.assignmentRevision,taxonomyVersionId:version,member:next});setNotice("saved");}catch{setNotice("failed");}finally{pending.current=false;setBusy(false);}
 }
 function edit(update:()=>void){update();setNotice(null);onDirty?.()}
 return <section className={styles.card}><h3>{t("topic.assignment.title",{})} · {member.knowledge.id} v{member.knowledge.version}</h3><p>{t("topic.assignment.pending",{})}</p><fieldset disabled={disabled||busy}><label className={styles.field}>{t("topic.assignment.ids",{})}<input value={ids} onChange={e=>edit(()=>setIds(e.target.value))}/></label><label className={styles.field}>{t("topic.assignment.version",{})}<input value={version} maxLength={64} onChange={e=>edit(()=>setVersion(e.target.value))}/></label><label className={styles.field}>{t("topic.assignment.batch",{})}<input value={batch} maxLength={64} onChange={e=>edit(()=>setBatch(e.target.value))}/></label><label className={styles.field}>{t("topic.assignment.sources",{})}<textarea value={sources} maxLength={7000} rows={8} onChange={e=>edit(()=>setSources(e.target.value))}/></label><p className={styles.metadata}>{member.sourceRefs.map(r=>r.recordId).join(", ")}</p><button type="button" className="button secondary" onClick={()=>void save()}>{t("topic.assignment.save",{})}</button></fieldset>{notice&&<p role={notice==="saved"?"status":"alert"}>{t(notice==="saved"?"topic.assignment.saved":notice==="failed"?"topic.assignment.failed":"topic.assignment.invalid",{})}</p>}</section>
}
