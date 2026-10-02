"use client";
import { useContext, useState } from 'react';
import { useRouter } from 'next/navigation';
import { FeedbackAccountContext, useFeedbackRead } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { FeedbackCommandStatus } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import type { Context, CreateInput, Metadata } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function NewFeedbackForm({ context }: { context: Context }) {
  const account = useContext(FeedbackAccountContext), router = useRouter(), read = useFeedbackRead();
  const [title, setTitle] = useState(''), [message, setMessage] = useState(''), [location, setLocation] = useState(''), [category, setCategory] = useState<CreateInput['category']>(context.target.kind === 'site' ? 'technical_issue' : 'math_error');
  const command = useFeedbackCommand(account?.actorId ?? '', async receipt => { const latest = await read<Metadata>('/api/v1/feedback/tickets/' + receipt.ticket.id); if (latest) router.push('/feedback/' + latest.data.id); });
  const input: CreateInput = { target: context.target, source: context.source, category, title, message, location }, locked = command.busy || !!command.pending;
  return <section className={styles.workbench}><h1>Report a problem</h1><p className={styles.metadata}>{context.label}</p><p>Describe what happened and what would help. Your original report stays in the discussion history.</p>
    <form onSubmit={e => { e.preventDefault(); void command.run({ route: '/api/v1/feedback/tickets', input }); }}>
      <fieldset disabled={locked}><label className={styles.field}>Category<select aria-label="Category" value={category} onChange={e => setCategory(e.target.value as CreateInput['category'])}>{(context.target.kind === 'site' ? ['technical_issue', 'accessibility', 'typo', 'suggestion'] : ['math_error', 'unclear_explanation', 'typo', 'accessibility', 'technical_issue', 'suggestion']).map(v => <option key={v} value={v}>{v.replaceAll('_', ' ')}</option>)}</select></label>
      <label className={styles.field}>Title<input aria-label="Title" value={title} onChange={e => setTitle(e.target.value)} required/><small>{[...title].length}/120 characters</small></label>
      <label className={styles.field}>Where on the page?<textarea aria-label="Where on the page?" value={location} onChange={e => setLocation(e.target.value)} rows={2}/><small>{[...location].length}/400 characters</small></label>
      <label className={styles.field}>Details<textarea aria-label="Details" value={message} onChange={e => setMessage(e.target.value)} rows={8} required/><small>{[...message].length}/4000 characters</small></label></fieldset>
      <button className="button" disabled={locked || !feedbackInputSchemas.create.safeParse(input).success}>Submit report</button><FeedbackCommandStatus command={command}/>
    </form></section>;
}
