"use client";
import Link from 'next/link';
import { useContext, useState } from 'react';
import { FeedbackAccountContext, useFeedbackRead } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { ReviewPanel } from './review-panel';
import { FeedbackCommandStatus, FeedbackState, FeedbackStatus } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import { FeedbackRequestError, type DiscussionPage, type Metadata, type Receipt } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function DiscussionPanel({ ticketId, review = false }: { ticketId: string; review?: boolean }) {
  const [page, setPage] = useState<DiscussionPage | null>(null), [error, setError] = useState<FeedbackRequestError | null>(null), [busy, setBusy] = useState(false), read = useFeedbackRead();
  async function load(cursor?: string) { if (busy) return; setBusy(true); setPage(null); setError(null); try { const result = await read<DiscussionPage>('/api/v1/feedback/' + (review ? 'review/' : '') + 'tickets/' + ticketId + '/events?limit=50' + (cursor ? '&cursor=' + cursor : '')); if (result) setPage(result.data); } catch (e) { setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); } finally { setBusy(false); } }
  return <section className={styles.card}><h2>Discussion</h2><button type="button" className="button secondary" disabled={busy} onClick={() => void load()}>{busy ? 'Reading…' : 'Read discussion'}</button>
    {page && <div><h3 style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{page.title}</h3>{page.location && <p style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{page.location}</p>}
      {page.items.map(event => <article className={styles.row} key={event.sequence}><p>Event {event.sequence} · {event.actor === 'submitter' ? 'Report author' : 'Review team'} · <FeedbackStatus status={event.to}/></p><p style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{event.message}</p>{event.resolution && <p>Basis: {event.resolution.kind.replaceAll('_', ' ')}{event.resolution.withdrawal && <> · Withdrawal {event.resolution.withdrawal.id}</>}{event.resolution.replacement && <> · Replacement {event.resolution.replacement.identity?.id ?? event.resolution.replacement.asset?.id} · Publication {event.resolution.replacement.publicationId}</>}{review && event.resolution.duplicateOf && <> · Related report {event.resolution.duplicateOf}</>}</p>}</article>)}
      {page.nextCursor && <button type="button" className="button secondary" disabled={busy} onClick={() => void load(page.nextCursor!)}>Next discussion page</button>}</div>}{error && <FeedbackState error={error}/>}</section>;
}
export function FeedbackTicket({ initial, review }: { initial: Metadata; review: boolean }) {
  const [ticket, setTicket] = useState(initial), [message, setMessage] = useState(''), [error, setError] = useState<FeedbackRequestError | null>(null), account = useContext(FeedbackAccountContext), read = useFeedbackRead();
  const path = '/api/v1/feedback/' + (review ? 'review/' : '') + 'tickets/' + initial.id;
  async function refresh(_receipt?: Receipt) { const result = await read<Metadata>(path); if (result) { setTicket(result.data); setMessage(''); setError(null); } }
  const command = useFeedbackCommand(account?.actorId ?? '', refresh), input = { expectedSequence: ticket.sequence, message };
  return <main className={styles.workbench}><Link prefetch={false} href={review ? '/review/feedback' : '/feedback'}>Back to reports</Link><h1>Report details</h1><p className={styles.metadata}>{ticket.label}</p><p><FeedbackStatus status={ticket.status}/> · Event {ticket.sequence} · {ticket.targetValidity}</p>{ticket.resolutionKind && <p>Current basis: {ticket.resolutionKind.replaceAll('_', ' ')}</p>}
    <button type="button" className="button secondary" onClick={() => void refresh().catch(e => setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()))}>Reload report status</button>{error && <FeedbackState error={error}/>}
    <DiscussionPanel key={ticket.sequence} ticketId={ticket.id} review={review}/>
    {review && ticket.canHandle ? <ReviewPanel key={ticket.sequence} ticket={ticket} onSuccess={refresh}/> : <form onSubmit={e => { e.preventDefault(); void command.run({ route: `/api/v1/feedback/tickets/${ticket.id}/reply`, input }); }}><label className={styles.field}>Additional details<textarea aria-label="Additional details" required rows={6} disabled={command.busy || !!command.pending} value={message} onChange={e => setMessage(e.target.value)}/><small>{[...message].length}/4000 characters</small></label><button className="button" disabled={command.busy || !!command.pending || !feedbackInputSchemas.reply.safeParse(input).success}>Add details</button><FeedbackCommandStatus command={command}/></form>}
  </main>;
}
