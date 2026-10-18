'use client';
import Link from 'next/link';
import {useRouter} from 'next/navigation';
import {useId,useRef,useState,useEffect,type ReactNode} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {pageModel,type PageMeta} from '@/lib/pagination/model';
import {pageHref} from '@/lib/pagination/query';
import styles from '@/styles/pagination.module.css';
type Navigation={kind:'link';pathname:string;query:Record<string,string>;offsetKey?:string;pageMode?:boolean;pageKey?:string;cursorKey?:string}|{kind:'callback';onPageChange:(offset:number)=>Promise<void>|void};
export type PaginationProps={meta:PageMeta;label:ReactNode;disabled?:boolean;navigation:Navigation};
function Icon({kind}:{kind:'first'|'previous'|'next'|'last'|'jump'}){return <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{kind==='jump'?<path d="M4 12h15m-6-6 6 6-6 6"/>:kind==='first'?<path d="M5 5v14m13-13-7 6 7 6"/>:kind==='last'?<path d="M19 5v14M6 6l7 6-7 6"/>:kind==='previous'?<path d="m15 5-7 7 7 7"/>:<path d="m9 5 7 7-7 7"/>}</svg>}
export function Pagination({meta,label,disabled=false,navigation}:PaginationProps){
 const{t}=useUiI18n(),router=useRouter(),id=useId(),model=pageModel(meta),pending=useRef(false);
 const[value,setValue]=useState(String(model.currentPage||'')),[busy,setBusy]=useState(false),[error,setError]=useState<'invalid'|'failed'|null>(null);
 useEffect(()=>{setValue(String(model.currentPage||''));setError(null)},[model.currentPage,meta.total,meta.limit]);
 const locked=disabled||busy,href=(offset:number)=>navigation.kind==='link'?pageHref(navigation.pathname,navigation.query,offset,{offsetKey:navigation.offsetKey??'offset',pageMode:navigation.pageMode??false,limit:meta.limit,pageKey:navigation.pageKey,cursorKey:navigation.cursorKey}):'';
 async function move(offset:number){if(locked||pending.current)return;pending.current=true;setBusy(true);setError(null);try{if(navigation.kind==='callback')await navigation.onPageChange(offset);else router.push(href(offset))}catch{setError('failed')}finally{pending.current=false;setBusy(false)}}
 const current=Math.min(meta.offset,model.lastOffset);
 const controls=[['first',0],['previous',Math.max(0,current-meta.limit)],['next',Math.min(model.lastOffset,current+meta.limit)],['last',model.lastOffset]] as const;
 return <nav className={styles.paging} aria-labelledby={id} aria-busy={busy}>
  <span id={id} className={styles.hidden}>{label}</span>
  <span className={styles.summary} aria-live="polite">{t('pagination.summary',{page:model.currentPage,pages:model.totalPages,total:meta.total})}</span>
  <div className={styles.controls}>{controls.map(([kind,offset])=>{const name=t(`pagination.${kind}`,{}),off=locked||!meta.total||(kind==='first'||kind==='previous'?current===0:current===model.lastOffset);return navigation.kind==='link'&&!off?<Link className={styles.icon} prefetch={false} href={href(offset)} aria-label={name} title={name} key={kind}><Icon kind={kind}/></Link>:<button className={styles.icon} type="button" aria-label={name} title={name} disabled={off} onClick={()=>void move(offset)} key={kind}><Icon kind={kind}/></button>})}</div>
  <form className={styles.jump} noValidate onSubmit={event=>{event.preventDefault();const page=Number(value);if(!/^[1-9][0-9]*$/.test(value)||!Number.isSafeInteger(page)||page>model.totalPages){setError('invalid');return}void move((page-1)*meta.limit)}}>
   <label className={styles.hidden} htmlFor={id+'-input'}>{t('pagination.number',{})}</label><input id={id+'-input'} type="text" inputMode="numeric" autoComplete="off" value={value} onChange={e=>{setValue(e.target.value);setError(null)}} disabled={locked||!meta.total} aria-invalid={error==='invalid'} aria-describedby={error?id+'-error':undefined}/>
   <button className={styles.icon} type="submit" aria-label={t('pagination.jump',{})} title={t('pagination.jump',{})} disabled={locked||!meta.total}><Icon kind="jump"/></button>
  </form>{error&&<p id={id+'-error'} role="alert" className={styles.error}>{t(error==='invalid'?'pagination.invalid':'pagination.failed',{})}</p>}
 </nav>
}
