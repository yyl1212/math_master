"use client";
import { useContext, useState } from 'react';
import { FeedbackAccountContext } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { FeedbackCommandStatus, feedbackStatusLabels } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import type { Metadata, Receipt, Resolution, TransitionInput } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
const edges = { new: ['new', 'processing', 'waiting_details', 'closed'], processing: ['processing', 'waiting_details', 'resolved', 'closed'], waiting_details: ['waiting_details', 'processing', 'resolved', 'closed'], resolved: ['resolved', 'processing'], closed: ['closed', 'processing'] } as const;
export function ReviewPanel({ ticket, onSuccess }: { ticket: Metadata; onSuccess: (receipt: Receipt) => Promise<void> }) {
  const account = useContext(FeedbackAccountContext), [status, setStatus] = useState<Metadata['status']>(ticket.status), [message, setMessage] = useState(''), [basis, setBasis] = useState<Resolution['kind']>('clarified');
  const [space, setSpace] = useState<'content' | 'question'>(ticket.target.kind === 'instance' ? 'question' : 'content'), [withdrawal, setWithdrawal] = useState(''), [duplicate, setDuplicate] = useState(''), [replacementKind, setReplacementKind] = useState<'knowledge' | 'path' | 'unit' | 'asset' | 'instance'>(ticket.target.part?.kind ?? (ticket.target.kind === 'site' ? 'knowledge' : ticket.target.kind)), [replacementId, setReplacementId] = useState(''), [version, setVersion] = useState(''), [sha, setSha] = useState(''), [publication, setPublication] = useState('');
  const command = useFeedbackCommand(account?.actorId ?? '', async receipt => { await onSuccess(receipt); setMessage(''); });
  if (!ticket.canHandle) return null;
  const terminal = status !== ticket.status && (status === 'resolved' || status === 'closed');
  function makeResolution(): Resolution | null {
    if (!terminal) return null;
    if (basis === 'duplicate') return { kind: basis, withdrawal: null, replacement: null, duplicateOf: duplicate };
    if (basis === 'withdrawn') return { kind: basis, withdrawal: { space, id: withdrawal }, replacement: null, duplicateOf: null };
    if (basis === 'revision_published') {
      const replacement = replacementKind === 'asset' ? { kind: 'asset' as const, identity: null, asset: { id: replacementId, sha256: sha }, publicationId: publication } : { kind: replacementKind, identity: { id: replacementId, version: Number(version), sha256: sha }, asset: null, publicationId: publication };
      return { kind: basis, withdrawal: { space, id: withdrawal }, replacement, duplicateOf: null };
    }
    return { kind: basis, withdrawal: null, replacement: null, duplicateOf: null };
  }
  const resolution = makeResolution();
  const input: TransitionInput = { expectedSequence: ticket.sequence, status, message, resolution }, locked = command.busy || !!command.pending;
  const bases: Resolution['kind'][] = status === 'closed' ? ['duplicate', 'not_reproducible', 'out_of_scope', 'suggestion_recorded'] : ['clarified', 'withdrawn', 'revision_published', ...(ticket.target.kind === 'site' ? ['service_fixed' as const] : [])];
  return <section className={styles.card}><h2>Handle this report</h2><form onSubmit={e => { e.preventDefault(); void command.run({ route: `/api/v1/feedback/review/tickets/${ticket.id}/transition`, input }); }}><fieldset disabled={locked}>
    <label className={styles.field}>Report status<select aria-label="Report status" value={status} onChange={e => { const value = e.target.value as Metadata['status']; setStatus(value); setBasis(value === 'closed' ? 'suggestion_recorded' : 'clarified'); }}>{edges[ticket.status].map(v => <option key={v} value={v}>{feedbackStatusLabels[v]}</option>)}</select></label>
    <label className={styles.field}>Review reply<textarea aria-label="Review reply" rows={6} required value={message} onChange={e => setMessage(e.target.value)}/><small>{[...message].length}/4000 characters</small></label>
    {terminal && <label className={styles.field}>Resolution basis<select aria-label="Resolution basis" value={basis} onChange={e => setBasis(e.target.value as Resolution['kind'])}>{bases.map(v => <option key={v} value={v}>{v.replaceAll('_', ' ')}</option>)}</select></label>}
    {terminal && basis === 'duplicate' && <label className={styles.field}>Duplicate report ID<input value={duplicate} onChange={e => setDuplicate(e.target.value)}/></label>}
    {terminal && ['withdrawn', 'revision_published'].includes(basis) && <><label className={styles.field}>Withdrawal source<select aria-label="Withdrawal source" value={space} onChange={e => setSpace(e.target.value as typeof space)}><option value="content">Content</option><option value="question">Question bank</option></select></label><label className={styles.field}>Withdrawal event ID<input value={withdrawal} onChange={e => setWithdrawal(e.target.value)}/></label></>}
    {terminal && basis === 'revision_published' && <><p>Use an independently approved, already published replacement.</p><label className={styles.field}>Replacement kind<select aria-label="Replacement kind" value={replacementKind} onChange={e => setReplacementKind(e.target.value as typeof replacementKind)}>{['knowledge', 'path', 'unit', 'asset', 'instance'].map(v => <option key={v} value={v}>{v}</option>)}</select></label><label className={styles.field}>Replacement ID<input value={replacementId} onChange={e => setReplacementId(e.target.value)}/></label>{replacementKind !== 'asset' && <label className={styles.field}>Replacement version<input value={version} onChange={e => setVersion(e.target.value)} inputMode="numeric"/></label>}<label className={styles.field}>Replacement SHA256<input value={sha} onChange={e => setSha(e.target.value)}/></label><label className={styles.field}>Replacement publication ID<input value={publication} onChange={e => setPublication(e.target.value)}/></label></>}
    </fieldset><button className="button" disabled={locked || !feedbackInputSchemas.transition.safeParse(input).success}>Save handling result</button><FeedbackCommandStatus command={command}/></form></section>;
}
