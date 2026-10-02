"use client";
import { useContext, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { FeedbackAccountContext, useFeedbackRead } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { FeedbackCommandStatus, FeedbackState } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import { FeedbackRequestError, type Context, type CreateInput, type Metadata } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function NewFeedbackForm({ context, sourcePath }: { context: Context; sourcePath: string }) {
  const account = useContext(FeedbackAccountContext), router = useRouter(), read = useFeedbackRead();
  const [currentContext, setCurrentContext] = useState(context), [refreshingTarget, setRefreshingTarget] = useState(false), [targetError, setTargetError] = useState<FeedbackRequestError | null>(null), [targetRefreshed, setTargetRefreshed] = useState(false);
  useEffect(() => { setCurrentContext(context); }, [context]);
  const [title, setTitle] = useState(''), [message, setMessage] = useState(''), [location, setLocation] = useState(''), [category, setCategory] = useState<CreateInput['category']>(context.target.kind === 'site' ? 'technical_issue' : 'math_error');
  const command = useFeedbackCommand(account?.actorId ?? '', async (receipt, signal) => { const latest = await read<Metadata>('/api/v1/feedback/tickets/' + receipt.ticket.id, signal); if (latest) router.push('/feedback/' + latest.data.id); });
  const input: CreateInput = { target: currentContext.target, source: currentContext.source, category, title, message, location }, locked = command.busy || !!command.pending || refreshingTarget;
  async function refreshTarget() {
    if (refreshingTarget || command.busy) return;
    setRefreshingTarget(true); setTargetError(null);
    try {
      const latest = await read<Context>(sourcePath);
      if (latest) { setCurrentContext(latest.data); setTargetRefreshed(true); command.clear(); }
    } catch (e) { setTargetError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); }
    finally { setRefreshingTarget(false); }
  }
  return <section className={styles.workbench}><h1>Report a problem</h1><p className={styles.metadata}>{currentContext.label}</p><p>Describe what happened and what would help. Your original report stays in the discussion history.</p>
    <form onSubmit={e => { e.preventDefault(); setTargetRefreshed(false); void command.run({ route: '/api/v1/feedback/tickets', input }); }}>
      <fieldset disabled={locked}><label className={styles.field}>Category<select aria-label="Category" value={category} onChange={e => setCategory(e.target.value as CreateInput['category'])}>{(currentContext.target.kind === 'site' ? ['technical_issue', 'accessibility', 'typo', 'suggestion'] : ['math_error', 'unclear_explanation', 'typo', 'accessibility', 'technical_issue', 'suggestion']).map(v => <option key={v} value={v}>{v.replaceAll('_', ' ')}</option>)}</select></label>
      <label className={styles.field}>Title<input aria-label="Title" value={title} onChange={e => setTitle(e.target.value)} required/><small>{[...title].length}/120 characters</small></label>
      <label className={styles.field}>Where on the page?<textarea aria-label="Where on the page?" value={location} onChange={e => setLocation(e.target.value)} rows={2}/><small>{[...location].length}/400 characters</small></label>
      <label className={styles.field}>Details<textarea aria-label="Details" value={message} onChange={e => setMessage(e.target.value)} rows={8} required/><small>{[...message].length}/4000 characters</small></label></fieldset>
      <button className="button" disabled={locked || !feedbackInputSchemas.create.safeParse(input).success}>Submit report</button><FeedbackCommandStatus command={command}/>
      {command.error?.code === 'FEEDBACK_TARGET_STALE' && <button type="button" className="button secondary" disabled={command.busy || refreshingTarget} onClick={() => void refreshTarget()}>Refresh report target</button>}
      {refreshingTarget && <p role="status">Refreshing report target…</p>}
      {targetRefreshed && <p role="status">Report target refreshed. Review the version before submitting again.</p>}
      {targetError && <FeedbackState error={targetError} onRetry={() => void refreshTarget()}/>}
    </form></section>;
}
