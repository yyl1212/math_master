"use client";
import {formatUiEnum} from "@/lib/i18n/enums";
import {UiEnum} from "@/lib/i18n/enums";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import { useContext, useEffect, useState } from 'react';
import { FeedbackAccountContext, useFeedbackRead } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { ReviewPanel } from './review-panel';
import { FeedbackCommandStatus, FeedbackState, FeedbackStatus } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import { FeedbackRequestError, type DiscussionPage, type Metadata, type Receipt } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function DiscussionPanel({ ticketId, review = false }: { ticketId: string; review?: boolean }) {
 const {t,locale}=useUiI18n();

  const [page, setPage] = useState<DiscussionPage | null>(null), [error, setError] = useState<FeedbackRequestError | null>(null), [busy, setBusy] = useState(false), read = useFeedbackRead();
  async function load(cursor?: string) { if (busy) return; setBusy(true); setPage(null); setError(null); try { const result = await read<DiscussionPage>('/api/v1/feedback/' + (review ? 'review/' : '') + 'tickets/' + ticketId + '/events?limit=50' + (cursor ? '&cursor=' + cursor : '')); if (result) setPage(result.data); } catch (e) { setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); } finally { setBusy(false); } }
  return <section className={styles.card}><h2><UiText notice={uiMessage("discussion-panel.discussion.5eb6cf",{})}/></h2><button type="button" className="button secondary" disabled={busy} onClick={() => void load()}>{busy ? t("discussion-panel.reading.874bfa",{}) : t("discussion-panel.read.discussion.60768c",{})}</button>
    {page && <div><h3 lang="" style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{page.title}</h3>{page.location && <p style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{page.location}</p>}
      {page.items.map(event => <article className={styles.row} key={event.sequence}><p><UiText notice={uiMessage("discussion-panel.event.133a15",{})}/>{event.sequence} · {event.actor === 'submitter' ? t("discussion-panel.report.author.24263d",{}) : t("discussion-panel.review.team.4433d3",{})} · <FeedbackStatus status={event.to}/></p><p lang="" style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{event.message}</p>{event.resolution && <p><UiText notice={uiMessage("discussion-panel.basis.5690cd",{})}/><UiEnum group="feedback.basis" value={event.resolution.kind}/>{event.resolution.withdrawal && <><UiText notice={uiMessage("discussion-panel.withdrawal.f3cdbc",{})}/>{event.resolution.withdrawal.id}</>}{event.resolution.replacement && <><UiText notice={uiMessage("discussion-panel.replacement.aa0aec",{})}/>{event.resolution.replacement.identity?.id ?? event.resolution.replacement.asset?.id}<UiText notice={uiMessage("discussion-panel.publication.6dbeee",{})}/>{event.resolution.replacement.publicationId}</>}{review && event.resolution.duplicateOf && <><UiText notice={uiMessage("discussion-panel.related.report.db9c9f",{})}/>{event.resolution.duplicateOf}</>}</p>}</article>)}
      {page.nextCursor && <button type="button" className="button secondary" disabled={busy} onClick={() => void load(page.nextCursor!)}><UiText notice={uiMessage("discussion-panel.next.discussion.page.007355",{})}/></button>}</div>}{error && <FeedbackState error={error}/>}</section>;
}
export function FeedbackTicket({ initial, review }: { initial: Metadata; review: boolean }) {
 const {t,locale}=useUiI18n();

  const [ticket, setTicket] = useState(initial), [message, setMessage] = useState(''), [error, setError] = useState<FeedbackRequestError | null>(null), account = useContext(FeedbackAccountContext), read = useFeedbackRead();
  useEffect(() => { setTicket(initial); }, [initial]);
  const path = '/api/v1/feedback/' + (review ? 'review/' : '') + 'tickets/' + initial.id;
  async function refresh(_receipt?: Receipt, signal?: AbortSignal) { const result = await read<Metadata>(path, signal); if (result) { setTicket(result.data); setError(null); } return result?.data ?? null; }
  const command = useFeedbackCommand(account?.actorId ?? '', async (receipt, signal) => { const latest = await refresh(receipt, signal); if (latest) setMessage(''); }), input = { expectedSequence: ticket.sequence, message };
  return <section className={styles.workbench} style={{overflowWrap:"anywhere"}}><Link prefetch={false} href={review ? '/review/feedback' : '/feedback'}><UiText notice={uiMessage("discussion-panel.back.to.reports.b62a50",{})}/></Link><h1><UiText notice={uiMessage("page.review.feedback.id",{})}/></h1><p className={styles.metadata}>{ticket.label}</p><p><FeedbackStatus status={ticket.status}/><UiText notice={uiMessage("ticket-list.event.5a4db3",{})}/>{ticket.sequence} · {ticket.targetValidity}</p>{ticket.resolutionKind && <p><UiText notice={uiMessage("discussion-panel.current.basis.value.9e8023",{v0:formatUiEnum(locale,"feedback.basis",ticket.resolutionKind)})}/></p>}
    <button type="button" className="button secondary" onClick={() => void refresh().catch(e => setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()))}><UiText notice={uiMessage("discussion-panel.reload.report.status.60577f",{})}/></button>{error && <FeedbackState error={error}/>}
    <DiscussionPanel key={"discussion-"+ticket.sequence} ticketId={ticket.id} review={review}/>
    {review && ticket.canHandle ? <ReviewPanel key={"review-"+ticket.id} ticket={ticket} onSuccess={refresh}/> : <form onSubmit={e => { e.preventDefault(); void command.run({ route: `/api/v1/feedback/tickets/${ticket.id}/reply`, input }); }}><label className={styles.field}><UiText notice={uiMessage("discussion-panel.additional.details.00fcfc",{})}/><textarea aria-label={t("discussion-panel.additional.details.00fcfc",{})} required rows={6} disabled={command.busy || !!command.pending} value={message} onChange={e => setMessage(e.target.value)}/><small><UiText notice={uiMessage("new-form.value.4000.characters.9daa9f",{v0:uiValue([...message].length)})}/></small></label><button className="button" disabled={command.busy || !!command.pending || !feedbackInputSchemas.reply.safeParse(input).success}><UiText notice={uiMessage("discussion-panel.add.details.78056f",{})}/></button><FeedbackCommandStatus command={command}/></form>}
  </section>;
}
