'use client';
import styles from '@/styles/content.module.css';
import {useEffect,useId,useRef,useState} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {previewSource,applyImport,readImport} from '@/lib/knowledge-admin/client';
import {KnowledgeError,type Preview,type Receipt,type ApplyInput} from '@/lib/knowledge-admin/types';
type Phase='selected'|'ready'|'done'|'invalid'|'uncertain';
type Job={name:string;importId?:string;file?:File;previewKey:string;applyKey:string;preview?:Preview;input?:ApplyInput;receipt?:Receipt;selected:number[];phase:Phase;error?:string};
const storageKey='managed-import-batch';
export function UploadKnowledge({onComplete,actorId=''}:{onComplete:()=>void;actorId?:string}){
 const fileId=useId(),fileLabelId=fileId+'-label',fileButtonId=fileId+'-button',fileStatusId=fileId+'-status';
 const{t}=useUiI18n();const[jobs,setJobs]=useState<Job[]>([]),[publish,setPublish]=useState(true),[busy,setBusy]=useState(false),[ready,setReady]=useState(false),[recoverable,setRecoverable]=useState(false);const current=useRef<Job[]>([]),generation=useRef(0);
 const refresh=(mode=publish)=>{setJobs([...current.current]);const saved=current.current.filter(job=>(job.preview||job.importId)&&job.phase!=='done').map(job=>({name:job.name,importId:job.preview?.importId??job.importId,applyKey:job.applyKey,input:job.input}));if(saved.length){sessionStorage.setItem(storageKey,JSON.stringify({actorId,publish:mode,jobs:saved}));setRecoverable(true)}else{sessionStorage.removeItem(storageKey);setRecoverable(false)}};
 useEffect(()=>{setReady(true);try{const saved=JSON.parse(sessionStorage.getItem(storageKey)??'null');setRecoverable(saved?.actorId===actorId&&Array.isArray(saved?.jobs));if(saved&&saved.actorId!==actorId)sessionStorage.removeItem(storageKey)}catch{sessionStorage.removeItem(storageKey)}const reset=()=>{generation.current++;current.current=[];setJobs([]);setBusy(false);setRecoverable(false);sessionStorage.removeItem(storageKey)};window.addEventListener('math-master:auth-change',reset);return()=>{generation.current++;window.removeEventListener('math-master:auth-change',reset)}},[actorId]);
 function failure(job:Job,error:unknown){const definite=error instanceof KnowledgeError&&error.status<500;job.phase=definite?'invalid':'uncertain';job.error=definite?'managed.invalidInput':'managed.uncertain';}
 async function inspect(){const token=generation.current;setBusy(true);for(const job of current.current){if(token!==generation.current)break;if(job.preview||job.phase==='done'||!job.file)continue;if(job.file.size>64*1024*1024){job.phase='invalid';job.error='managed.files';refresh();continue}try{const preview=await previewSource(job.file,job.previewKey);if(token!==generation.current)break;job.preview=preview;job.selected=preview.items.filter(item=>item.action==='create'||item.action==='link').map(item=>item.index);job.phase='ready';job.error=undefined;}catch(error){if(token!==generation.current)break;failure(job,error);}refresh();}if(token===generation.current)setBusy(false);}
 async function apply(){const token=generation.current;setBusy(true);let updated=false;for(const job of current.current){if(token!==generation.current)break;if(!job.preview||job.phase==='done'||job.phase==='invalid')continue;const checkReceipt=job.phase==='uncertain';job.input??={selectedIndexes:job.selected,publish,previewToken:job.preview.previewToken};job.phase='uncertain';refresh();try{let receipt:Receipt|undefined;if(checkReceipt){const status=await readImport(job.preview.importId);if(token!==generation.current)break;receipt=status.receipt??undefined;}receipt??=await applyImport(job.preview.importId,job.input,job.applyKey);if(token!==generation.current)break;job.receipt=receipt;job.phase='done';job.error=undefined;updated=true;}catch(error){if(token!==generation.current)break;failure(job,error);}refresh();}if(token===generation.current){setBusy(false);if(updated)onComplete();}}
 async function recover(){
  const token=generation.current;setBusy(true);
  try{
   let saved;try{saved=JSON.parse(sessionStorage.getItem(storageKey)??'null')}catch{sessionStorage.removeItem(storageKey);setRecoverable(false);return}
   if(!saved||saved.actorId!==actorId||!Array.isArray(saved.jobs)||saved.jobs.length>100)return;
   const mode=typeof saved.publish==='boolean'?saved.publish:publish;setPublish(mode);
   const recovered:Job[]=[];
   for(const item of saved.jobs){
    if(token!==generation.current)return;
    const job:Job={name:item.name,importId:item.importId,previewKey:'',applyKey:item.applyKey,input:item.input,selected:item.input?.selectedIndexes??[],phase:'uncertain'};
    try{
     const status=await readImport(item.importId);if(token!==generation.current)return;
     job.preview=status.preview;job.receipt=status.receipt??undefined;
     job.selected=item.input?.selectedIndexes??status.preview.items.filter((entry:Preview['items'][number])=>entry.action==='create'||entry.action==='link').map((entry:Preview['items'][number])=>entry.index);
     job.phase=status.receipt?'done':item.input?'uncertain':'ready';
    }catch{if(token!==generation.current)return;job.error='managed.uncertain'}
    recovered.push(job);
   }
   current.current=recovered;refresh(mode);onComplete();
  }finally{if(token===generation.current)setBusy(false)}
 }

 const pending=jobs.some(job=>job.phase==='ready'||job.phase==='uncertain'),totals=jobs.reduce((counts,job)=>{const value=job.receipt?.counts??job.preview?.counts;for(const key of Object.keys(counts)as(keyof typeof counts)[])counts[key]+=value?.[key]??0;return counts;},{createdKnowledge:0,linkedTopics:0,skippedItems:0,conflicts:0,invalidItems:0});
 const fileDisabled=!ready||busy||pending,fileCount=jobs.filter(job=>job.file).length;
 return <section className={'panel ui-form upload-form '+styles.uploadPanel}><h2>{t('managed.upload',{})}</h2>
  {recoverable&&(jobs.length===0||jobs.some(job=>!job.preview&&job.importId))&&<><p>{t('managed.pendingImport',{})}</p><button className="button secondary" disabled={busy} onClick={()=>void recover()}>{t('managed.recoverImport',{})}</button></>}
  <div className="file-field"><label id={fileLabelId} htmlFor={fileId}>{t('managed.files',{})}</label><div className="file-picker">
   <span className="file-picker-control" data-disabled={fileDisabled}><span id={fileButtonId} className="button secondary">{t('form.file.choose',{})}</span><input id={fileId} aria-labelledby={fileButtonId+' '+fileLabelId} aria-describedby={fileStatusId} type="file" accept="application/json,.json" multiple disabled={fileDisabled} onChange={event=>{generation.current++;current.current=Array.from(event.target.files??[]).map(file=>({name:file.name,file,previewKey:crypto.randomUUID(),applyKey:crypto.randomUUID(),selected:[],phase:'selected'}));refresh()}}/></span>
   <span id={fileStatusId} className="file-picker-status" role="status">{fileCount===0?t('form.file.empty',{}):fileCount===1?t('form.file.single',{}):t('form.file.count',{count:fileCount})}</span>
  </div></div>
  <label className="form-check"><input type="checkbox" checked={publish} disabled={busy||jobs.some(job=>!!job.input)} onChange={event=>{setPublish(event.target.checked);if(current.current.length)refresh(event.target.checked)}}/>{t('managed.uploadPublish',{})}</label>
  <div className="form-actions"><button className="button" disabled={busy||jobs.length===0||pending||jobs.every(job=>job.phase==='done')} onClick={()=>void inspect()}>{t(publish?'managed.uploadPublish':'managed.saveOnly',{})}</button></div>
  {jobs.some(job=>job.preview)&&<><h3>{t('managed.preview',{})}</h3><p>{t('managed.new',{})}: {totals.createdKnowledge} · {t('managed.linked',{})}: {totals.linkedTopics} · {t('managed.skipped',{})}: {totals.skippedItems} · {t('managed.conflicts',{})}: {totals.conflicts} · {t('managed.invalid',{})}: {totals.invalidItems}</p></>}
  <ul>{jobs.map((job,index)=><li key={job.previewKey||index} style={{marginBlock:12,overflowWrap:'anywhere'}}><strong>{job.name}</strong>{job.phase==='done'?<p role="status">{t('streamline.upload.done',{})}</p>:job.error?<p role="alert">{t(job.error as 'managed.invalidInput'|'managed.uncertain'|'managed.files',{})}</p>:null}{job.preview&&<details><summary>{t('streamline.upload.details',{})}</summary><ul>{job.preview.items.map(item=><li key={item.index}>{item.externalId} · {item.action}{item.errorPath&&' · '+item.errorPath}{(item.action==='create'||item.action==='link')&&<label className="form-check"><input type="checkbox" disabled={busy||!!job.input||job.phase==='done'} checked={job.selected.includes(item.index)} onChange={event=>{job.selected=event.target.checked?[...job.selected,item.index]:job.selected.filter(n=>n!==item.index);refresh()}}/>{t('managed.selected',{})}</label>}</li>)}</ul></details>}</li>)}</ul>
  {pending&&<button className="button" disabled={busy} onClick={()=>void apply()}>{t('streamline.upload.apply',{})}</button>}
  {jobs.some(job=>job.phase==='uncertain')&&<button className="button secondary" disabled={busy} onClick={()=>{if(jobs.some(job=>job.phase==='uncertain'&&!job.preview&&job.importId))void recover();else if(jobs.some(job=>job.phase==='uncertain'&&!job.preview))void inspect();else void apply()}}>{t('managed.retry',{})}</button>}
 </section>;
}
