"use client";
import {formatUiEnum} from "@/lib/i18n/enums";
import {UiEnum} from "@/lib/i18n/enums";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import { useContext, useState } from 'react';
import { FeedbackAccountContext } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { FeedbackCommandStatus, feedbackStatusLabels } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import type { Metadata, Receipt, Resolution, TransitionInput } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
const edges = { new: ['new', 'processing', 'waiting_details', 'closed'], processing: ['processing', 'waiting_details', 'resolved', 'closed'], waiting_details: ['waiting_details', 'processing', 'resolved', 'closed'], resolved: ['resolved', 'processing'], closed: ['closed', 'processing'] } as const;
export function ReviewPanel({ ticket, onSuccess }: { ticket: Metadata; onSuccess: (receipt: Receipt, signal: AbortSignal) => Promise<Metadata | null> }) {
 const {t,locale}=useUiI18n();

  const account = useContext(FeedbackAccountContext), [status, setStatus] = useState<Metadata['status']>(ticket.status), [message, setMessage] = useState(''), [basis, setBasis] = useState<Resolution['kind']>('clarified');
  const [space, setSpace] = useState<'content' | 'question'>(ticket.target.kind === 'instance' ? 'question' : 'content'), [withdrawal, setWithdrawal] = useState(''), [duplicate, setDuplicate] = useState(''), [replacementKind, setReplacementKind] = useState<'knowledge' | 'path' | 'unit' | 'asset' | 'instance'>(ticket.target.part?.kind ?? (ticket.target.kind === 'site' ? 'knowledge' : ticket.target.kind)), [replacementId, setReplacementId] = useState(''), [version, setVersion] = useState(''), [sha, setSha] = useState(''), [publication, setPublication] = useState('');
  const command = useFeedbackCommand(account?.actorId ?? '', async (receipt, signal) => {
    const latest = await onSuccess(receipt, signal);
    if (latest) { setMessage(''); setStatus(latest.status); setBasis(latest.status === 'closed' ? 'suggestion_recorded' : 'clarified'); setWithdrawal(''); setDuplicate(''); setReplacementId(''); setVersion(''); setSha(''); setPublication(''); }
  });
  if (!ticket.canHandle) return null;
  const allowedStatuses = edges[ticket.status] as readonly Metadata['status'][], statusAllowed = allowedStatuses.includes(status);
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
  return <section className={styles.card}><h2><UiText notice={uiMessage("review-panel.handle.this.report.3fef67",{})}/></h2><form onSubmit={e => { e.preventDefault(); void command.run({ route: `/api/v1/feedback/review/tickets/${ticket.id}/transition`, input }); }}><fieldset disabled={locked}>
    <label className={styles.field}><UiText notice={uiMessage("review-panel.report.status.22a134",{})}/><select aria-label={t("review-panel.report.status.22a134",{})} value={status} onChange={e => { const value = e.target.value as Metadata['status']; setStatus(value); setBasis(value === 'closed' ? 'suggestion_recorded' : 'clarified'); }}>{!statusAllowed && <option value={status} disabled><UiText notice={uiMessage("review-panel.value.choose.an.available.status.bb7389",{v0:formatUiEnum(locale,"feedback.status",status)})}/></option>}{allowedStatuses.map(v => <option key={v} value={v}><UiEnum group="feedback.status" value={v}/></option>)}</select></label>
    <label className={styles.field}><UiText notice={uiMessage("review-panel.review.reply.6bf2f2",{})}/><textarea aria-label={t("review-panel.review.reply.6bf2f2",{})} rows={6} required value={message} onChange={e => setMessage(e.target.value)}/><small><UiText notice={uiMessage("new-form.value.4000.characters.9daa9f",{v0:uiValue([...message].length)})}/></small></label>
    {terminal && <label className={styles.field}><UiText notice={uiMessage("review-panel.resolution.basis.47ef7d",{})}/><select aria-label={t("review-panel.resolution.basis.47ef7d",{})} value={basis} onChange={e => setBasis(e.target.value as Resolution['kind'])}>{bases.map(v => <option key={v} value={v}><UiEnum group="feedback.basis" value={v}/></option>)}</select></label>}
    {terminal && basis === 'duplicate' && <label className={styles.field}><UiText notice={uiMessage("review-panel.duplicate.report.id.2eed31",{})}/><input value={duplicate} onChange={e => setDuplicate(e.target.value)}/></label>}
    {terminal && ['withdrawn', 'revision_published'].includes(basis) && <><label className={styles.field}><UiText notice={uiMessage("review-panel.withdrawal.source.77a1ed",{})}/><select aria-label={t("review-panel.withdrawal.source.77a1ed",{})} value={space} onChange={e => setSpace(e.target.value as typeof space)}><option value="content"><UiText notice={uiMessage("review-panel.content.47bd29",{})}/></option><option value="question"><UiText notice={uiMessage("review-panel.question.bank.d8c022",{})}/></option></select></label><label className={styles.field}><UiText notice={uiMessage("review-panel.withdrawal.event.id.935c33",{})}/><input value={withdrawal} onChange={e => setWithdrawal(e.target.value)}/></label></>}
    {terminal && basis === 'revision_published' && <><p><UiText notice={uiMessage("review-panel.use.an.independently.approved.already.published.replacement.b34ac2",{})}/></p><label className={styles.field}><UiText notice={uiMessage("review-panel.replacement.kind.cdfd5f",{})}/><select aria-label={t("review-panel.replacement.kind.cdfd5f",{})} value={replacementKind} onChange={e => setReplacementKind(e.target.value as typeof replacementKind)}>{['knowledge', 'path', 'unit', 'asset', 'instance'].map(v => <option key={v} value={v}>{v}</option>)}</select></label><label className={styles.field}><UiText notice={uiMessage("review-panel.replacement.id.114847",{})}/><input value={replacementId} onChange={e => setReplacementId(e.target.value)}/></label>{replacementKind !== 'asset' && <label className={styles.field}><UiText notice={uiMessage("review-panel.replacement.version.bd1b6d",{})}/><input value={version} onChange={e => setVersion(e.target.value)} inputMode="numeric"/></label>}<label className={styles.field}><UiText notice={uiMessage("review-panel.replacement.sha256.1002ff",{})}/><input value={sha} onChange={e => setSha(e.target.value)}/></label><label className={styles.field}><UiText notice={uiMessage("review-panel.replacement.publication.id.6723c8",{})}/><input value={publication} onChange={e => setPublication(e.target.value)}/></label></>}
    </fieldset><button className="button" disabled={locked || !statusAllowed || !feedbackInputSchemas.transition.safeParse(input).success}><UiText notice={uiMessage("review-panel.save.handling.result.fe5bce",{})}/></button><FeedbackCommandStatus command={command}/></form></section>;
}
