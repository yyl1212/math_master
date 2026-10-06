"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import {useContext,useEffect,useRef,useState} from 'react';
import {LearningAccountContext} from '../learning/learning-account';
import {CorrectionAccountContext} from './correction-account';
import {correctionClient} from '@/lib/correction/client';
import {withCorrectionDeadline,correctionAwait} from '@/lib/correction/bytes';
import {CorrectionRequestError,type EvidenceRef,type ResultMetadata,type Page} from '@/lib/correction/types';
import {CorrectionStatus,CorrectionState} from './status';
export function EvidenceLink({evidence}:{evidence:EvidenceRef}){
 const {t}=useUiI18n();

 const learning=useContext(LearningAccountContext),correction=useContext(CorrectionAccountContext),account=correction??learning;
 const [page,setPage]=useState<Page<ResultMetadata>|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState<CorrectionRequestError|null>(null),active=useRef<AbortController|null>(null),revision=useRef(0);
 useEffect(()=>{setPage(null);setError(null);setBusy(false);return()=>{revision.current++;active.current?.abort()}},[account?.actorId,evidence.kind,evidence.id]);
 async function load(cursor?:string){if(!account||busy)return;const actorId=account.actorId,n=++revision.current,c=new AbortController();active.current?.abort();active.current=c;setBusy(true);setError(null);
  try{const out=await withCorrectionDeadline(c.signal,s=>correctionAwait(correctionClient.listOwn({...evidence,...(cursor?{cursor}:{})},{actorId,signal:s}),s));if(c.signal.aborted||n!==revision.current)return;if(out.actorId!==actorId){account.invalidate();return}setPage(old=>cursor&&old?{items:[...old.items,...out.data.items],nextCursor:out.data.nextCursor}:out.data)}
  catch(e){if(c.signal.aborted||n!==revision.current)return;const closed=e instanceof CorrectionRequestError?e:new CorrectionRequestError();if(['AUTHENTICATION_REQUIRED','PASSWORD_CHANGE_REQUIRED','FORBIDDEN'].includes(closed.code)){account.invalidate();return}setError(closed)}finally{if(n===revision.current&&!c.signal.aborted)setBusy(false)}
 }
 if(!account)return null;
 return <section aria-label={t("evidence-link.corrections.for.saved.evidence.09b140",{})}><button type="button" className="button secondary" disabled={busy} onClick={()=>void load()}><UiText notice={uiMessage("evidence-link.check.corrections.cddc7d",{})}/></button>{busy&&<p role="status"><UiText notice={uiMessage("evidence-link.checking.corrections.e0e9d6",{})}/></p>}{error&&<CorrectionState error={error} onRetry={()=>void load()}/>} {page&&page.items.length===0&&<p><UiText notice={uiMessage("evidence-link.no.correction.results.yet.be7993",{})}/></p>}{page&&page.items.length>0&&<ul>{page.items.map(v=><li key={v.id}><CorrectionStatus status={v.status}/> · {v.validity} <Link prefetch={false} href={'/corrections/'+v.id}><UiText notice={uiMessage("evidence-link.view.correction.13dc93",{})}/></Link></li>)}</ul>}{page?.nextCursor&&<button type="button" className="button secondary" disabled={busy} onClick={()=>void load(page.nextCursor??undefined)}><UiText notice={uiMessage("evidence-link.more.corrections.56d196",{})}/></button>}</section>
}
