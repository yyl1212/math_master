"use client";
import {UiEnum} from "@/lib/i18n/enums";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import { useEffect, useState } from 'react';
import {useRouter} from 'next/navigation';
import { useFeedbackRead } from './feedback-account';
import { FeedbackState, FeedbackStatus } from './status';
import { FeedbackRequestError, type MetadataPage } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function TicketList({ initial, review, route }: { initial: MetadataPage; review: boolean; route?: string }) {
 const {t}=useUiI18n();

  const [page, setPage] = useState(initial), [error, setError] = useState<FeedbackRequestError | null>(null), [busy, setBusy] = useState(false), read = useFeedbackRead(), router=useRouter();
  useEffect(() => { setPage(initial); setError(null); setBusy(false); }, [initial, route]);
  const base = review ? '/review/feedback' : '/feedback';
  async function next() { if (!page.nextCursor || busy) return; setBusy(true); setError(null); try { const path = new URL(route ?? '/api/v1/feedback' + (review ? '/review' : '') + '/tickets', 'http://localhost'); path.searchParams.set('cursor', page.nextCursor); const result = await read<MetadataPage>(path.pathname + path.search); if (result) setPage(result.data); } catch (e) { setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); } finally { setBusy(false); } }
  return <section className={styles.workbench} style={{overflowWrap:"anywhere"}}><h1>{review ? t("ticket-list.feedback.review.queue.55f05a",{}) : t("page.feedback",{})}</h1>{review ? <form key={route} method="GET" className={styles.actions} onSubmit={e=>{e.preventDefault();const data=new FormData(e.currentTarget),params=new URLSearchParams();for(const k of ['status','category']){const v=data.get(k);if(typeof v==='string'&&v)params.set(k,v)}router.push(base+(params.size?'?'+params:''))}}><label className={styles.field}><UiText notice={uiMessage("ticket-list.status.920e41",{})}/><select aria-label={t("ticket-list.status.920e41",{})} name="status" defaultValue={new URL(route ?? '', 'http://localhost').searchParams.get('status') ?? ''}><option value=""><UiText notice={uiMessage("ticket-list.all.statuses.8ee573",{})}/></option>{['new', 'processing', 'waiting_details', 'resolved', 'closed'].map(v => <option key={v} value={v}><UiEnum group="feedback.status" value={v}/></option>)}</select></label><label className={styles.field}><UiText notice={uiMessage("new-form.category.292c06",{})}/><select aria-label={t("new-form.category.292c06",{})} name="category" defaultValue={new URL(route??'','http://localhost').searchParams.get('category')??''}><option value=""><UiText notice={uiMessage("ticket-list.all.categories.9d5097",{})}/></option>{['math_error','unclear_explanation','typo','accessibility','technical_issue','suggestion'].map(v=><option key={v} value={v}><UiEnum group="feedback.category" value={v}/></option>)}</select></label><button className="button secondary"><UiText notice={uiMessage("ticket-list.filter.queue.f927f0",{})}/></button></form> : <Link prefetch={false} className="button secondary" href="/feedback/new?kind=site&area=other"><UiText notice={uiMessage("ticket-list.new.report.16f776",{})}/></Link>}
    {page.items.length ? <ul className={styles.list}>{page.items.map(t => <li key={t.id}><Link prefetch={false} href={base + '/' + t.id}>{t.label}</Link><p><FeedbackStatus status={t.status}/> · <UiEnum group="feedback.category" value={t.category}/><UiText notice={uiMessage("ticket-list.event.5a4db3",{})}/>{t.sequence}</p><small className={styles.metadata}>{t.targetValidity} · {new Date(t.updatedAt).toLocaleString('en')}</small></li>)}</ul> : <p>{review ? t("ticket-list.no.reports.match.this.queue.11ebb7",{}) : t("ticket-list.no.reports.yet.be893e",{})}</p>}
    {page.nextCursor && <button className="button secondary" disabled={busy} onClick={() => void next()}><UiText notice={uiMessage("ticket-list.next.reports.4eb46c",{})}/></button>}{error && <FeedbackState error={error}/>}</section>;
}
