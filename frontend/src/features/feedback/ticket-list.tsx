"use client";
import Link from 'next/link';
import { useState } from 'react';
import {useRouter} from 'next/navigation';
import { useFeedbackRead } from './feedback-account';
import { FeedbackState, FeedbackStatus } from './status';
import { FeedbackRequestError, type MetadataPage } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function TicketList({ initial, review, route }: { initial: MetadataPage; review: boolean; route?: string }) {
  const [page, setPage] = useState(initial), [error, setError] = useState<FeedbackRequestError | null>(null), [busy, setBusy] = useState(false), read = useFeedbackRead(), router=useRouter();
  const base = review ? '/review/feedback' : '/feedback';
  async function next() { if (!page.nextCursor || busy) return; setBusy(true); setError(null); try { const path = new URL(route ?? '/api/v1/feedback' + (review ? '/review' : '') + '/tickets', 'http://localhost'); path.searchParams.set('cursor', page.nextCursor); const result = await read<MetadataPage>(path.pathname + path.search); if (result) setPage(result.data); } catch (e) { setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); } finally { setBusy(false); } }
  return <section className={styles.workbench}><h1>{review ? 'Feedback review queue' : 'My reports'}</h1>{review ? <form method="GET" className={styles.actions} onSubmit={e=>{e.preventDefault();const data=new FormData(e.currentTarget),params=new URLSearchParams();for(const k of ['status','category']){const v=data.get(k);if(typeof v==='string'&&v)params.set(k,v)}router.push(base+(params.size?'?'+params:''))}}><label className={styles.field}>Status<select name="status" defaultValue={new URL(route ?? '', 'http://localhost').searchParams.get('status') ?? ''}><option value="">All statuses</option>{['new', 'processing', 'waiting_details', 'resolved', 'closed'].map(v => <option key={v} value={v}>{v.replaceAll('_', ' ')}</option>)}</select></label><label className={styles.field}>Category<select name="category" defaultValue={new URL(route??'','http://localhost').searchParams.get('category')??''}><option value="">All categories</option>{['math_error','unclear_explanation','typo','accessibility','technical_issue','suggestion'].map(v=><option key={v} value={v}>{v.replaceAll('_',' ')}</option>)}</select></label><button className="button secondary">Filter queue</button></form> : <Link prefetch={false} className="button secondary" href="/feedback/new?kind=site&area=other">New report</Link>}
    {page.items.length ? <ul className={styles.list}>{page.items.map(t => <li key={t.id}><Link prefetch={false} href={base + '/' + t.id}>{t.label}</Link><p><FeedbackStatus status={t.status}/> · {t.category.replaceAll('_', ' ')} · Event {t.sequence}</p><small className={styles.metadata}>{t.targetValidity} · {new Date(t.updatedAt).toLocaleString('en')}</small></li>)}</ul> : <p>{review ? 'No reports match this queue.' : 'No reports yet.'}</p>}
    {page.nextCursor && <button className="button secondary" disabled={busy} onClick={() => void next()}>Next reports</button>}{error && <FeedbackState error={error}/>}</section>;
}
