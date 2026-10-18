'use client';
import Link from 'next/link';
import {useId,useRef,useState,useEffect,type ReactNode} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {pageModel,type PageMeta} from '@/lib/pagination/model';
import {pageHref} from '@/lib/pagination/query';
import styles from '@/styles/pagination.module.css';
type Navigation={kind:'link';pathname:string;query:Record<string,string>;offsetKey?:string;pageMode?:boolean;pageKey?:string;cursorKey?:string}|{kind:'callback';onPageChange:(offset:number)=>Promise<void>|void};
export type PaginationProps={meta:PageMeta;label:ReactNode;disabled?:boolean;navigation:Navigation};
function Icon({kind}:{kind:'first'|'previous'|'next'|'last'|'jump'}){return <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{kind==='jump'?<path d="M4 12h15m-6-6 6 6-6 6"/>:kind==='first'?<path d="M5 5v14m13-13-7 6 7 6"/>:kind==='last'?<path d="M19 5v14M6 6l7 6-7 6"/>:kind==='previous'?<path d="m15 5-7 7 7 7"/>:<path d="m9 5 7 7-7 7"/>}</svg>}
export function Pagination({meta,label,disabled=false,navigation}:PaginationProps){
 const{t}=useUiI18n(),jumpTarget=useRef<HTMLAnchorElement>(null),correctionTarget=useRef<HTMLAnchorElement>(null),id=useId(),model=pageModel(meta),pending=useRef(false);
 const[value,setValue]=useState(String(model.currentPage||'')),[busy,setBusy]=useState(false),[error,setError]=useState<'invalid'|'failed'|null>(null);
 const signature=`${meta.total}:${meta.limit}:${meta.offset}`,attempted=useRef<string|null>(null),correctionTimer=useRef<ReturnType<typeof setTimeout>|null>(null),[failedCorrection,setFailedCorrection]=useState<string|null>(null);
 const latestSignature=useRef(signature);latestSignature.current=signature;
 useEffect(()=>{setValue(String(model.currentPage||''));setError(null)},[model.currentPage,meta.total,meta.limit,meta.offset]);
 useEffect(()=>()=>{if(correctionTimer.current!==null)clearTimeout(correctionTimer.current)},[signature]);
 const correcting=model.needsCorrection&&failedCorrection!==signature;
 const locked=disabled||busy||correcting,href=(offset:number)=>navigation.kind==='link'?pageHref(navigation.pathname,navigation.query,offset,{offsetKey:navigation.offsetKey??'offset',pageMode:navigation.pageMode??false,limit:meta.limit,pageKey:navigation.pageKey,cursorKey:navigation.cursorKey}):'';
 async function move(offset:number,automatic=false){if((locked&&!automatic)||pending.current)return;pending.current=true;setBusy(true);setError(null);try{if(navigation.kind==='callback')await navigation.onPageChange(offset);else jumpTarget.current?.click()}catch{setError('failed')}finally{pending.current=false;setBusy(false)}}
 useEffect(()=>{
  if(!model.needsCorrection){attempted.current=null;return}
  if(disabled||busy||attempted.current===signature)return;
  attempted.current=signature;setFailedCorrection(null);
  const failed=()=>{if(latestSignature.current===signature){setFailedCorrection(signature);setError('failed')}};
  if(navigation.kind==='callback')void move(model.lastOffset,true).then(failed);
  else{correctionTarget.current?.click();correctionTimer.current=setTimeout(failed,10_000)}
 // Each metadata window gets one corrective fetch; a failed request requires an explicit retry.
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[signature,model.needsCorrection,model.lastOffset,disabled,busy]);
 const current=Math.min(meta.offset,model.lastOffset);
 const controls=[['first',0],['previous',Math.max(0,current-meta.limit)],['next',Math.min(model.lastOffset,current+meta.limit)],['last',model.lastOffset]] as const;
 return <nav className={styles.paging} aria-labelledby={id} aria-busy={busy||correcting}>
  <span id={id} className={styles.hidden}>{label}</span>
  <span className={styles.summary} aria-live="polite">{model.needsCorrection?t(correcting?'pagination.correcting':'pagination.failed',{}):t('pagination.summary',{page:model.currentPage,pages:model.totalPages,total:meta.total})}</span>
  <div className={styles.controls}>{controls.map(([kind,offset])=>{const name=t(`pagination.${kind}`,{}),off=locked||!meta.total||(kind==='first'||kind==='previous'?current===0:current===model.lastOffset);return navigation.kind==='link'&&!off?<Link className={styles.icon} prefetch={false} href={href(offset)} aria-label={name} title={name} key={kind}><Icon kind={kind}/></Link>:<button className={styles.icon} type="button" aria-label={name} title={name} disabled={off} onClick={()=>void move(offset)} key={kind}><Icon kind={kind}/></button>})}</div>
  {navigation.kind==='link'&&<Link ref={jumpTarget} className={styles.hidden} tabIndex={-1} aria-hidden="true" prefetch={false} href={href(/^[1-9][0-9]*$/.test(value)&&Number.isSafeInteger(Number(value))&&Number(value)<=model.totalPages?(Number(value)-1)*meta.limit:current)}>{t('pagination.jump',{})}</Link>}
  {navigation.kind==='link'&&model.needsCorrection&&<Link ref={correctionTarget} className={styles.hidden} tabIndex={-1} aria-hidden="true" prefetch={false} href={href(model.lastOffset)}>{t('pagination.jump',{})}</Link>}
  <form className={styles.jump} noValidate onSubmit={event=>{event.preventDefault();const page=Number(value);if(!/^[1-9][0-9]*$/.test(value)||!Number.isSafeInteger(page)||page>model.totalPages){setError('invalid');return}void move((page-1)*meta.limit)}}>
   <label className={styles.hidden} htmlFor={id+'-input'}>{t('pagination.number',{})}</label><input id={id+'-input'} type="text" inputMode="numeric" autoComplete="off" value={value} onChange={e=>{setValue(e.target.value);setError(null)}} disabled={locked||!meta.total} aria-invalid={error==='invalid'} aria-describedby={error?id+'-error':undefined}/>
   <button className={styles.icon} type="submit" aria-label={t('pagination.jump',{})} title={t('pagination.jump',{})} disabled={locked||!meta.total}><Icon kind="jump"/></button>
  </form>{error&&<p id={id+'-error'} role="alert" className={styles.error}>{t(error==='invalid'?'pagination.invalid':'pagination.failed',{})}</p>}
 </nav>
}
