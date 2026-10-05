"use client";
import {UiEnum} from "@/lib/i18n/enums";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import { useContext, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { FeedbackAccountContext, useFeedbackRead } from './feedback-account';
import { useFeedbackCommand } from './pending-command';
import { FeedbackCommandStatus, FeedbackState } from './status';
import { feedbackInputSchemas } from '@/lib/feedback/schemas';
import { FeedbackRequestError, type Context, type CreateInput, type Metadata } from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
export function NewFeedbackForm({ context, sourcePath }: { context: Context; sourcePath: string }) {
 const {t}=useUiI18n();

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
  return <section className={styles.workbench}><h1><UiText notice={uiMessage("page.feedback.new",{})}/></h1><p className={styles.metadata}>{currentContext.label}</p><p><UiText notice={uiMessage("new-form.describe.what.happened.and.what.would.help.your.original.report.s.3a0e68",{})}/></p>
    <form onSubmit={e => { e.preventDefault(); setTargetRefreshed(false); void command.run({ route: '/api/v1/feedback/tickets', input }); }}>
      <fieldset disabled={locked}><label className={styles.field}><UiText notice={uiMessage("new-form.category.292c06",{})}/><select aria-label={t("new-form.category.292c06",{})} value={category} onChange={e => setCategory(e.target.value as CreateInput['category'])}>{(currentContext.target.kind === 'site' ? ['technical_issue', 'accessibility', 'typo', 'suggestion'] : ['math_error', 'unclear_explanation', 'typo', 'accessibility', 'technical_issue', 'suggestion']).map(v => <option key={v} value={v}><UiEnum group="feedback.category" value={v}/></option>)}</select></label>
      <label className={styles.field}><UiText notice={uiMessage("new-form.title.7e8cd2",{})}/><input aria-label={t("new-form.title.7e8cd2",{})} value={title} onChange={e => setTitle(e.target.value)} required/><small><UiText notice={uiMessage("new-form.value.120.characters.67f5dd",{v0:uiValue([...title].length)})}/></small></label>
      <label className={styles.field}><UiText notice={uiMessage("new-form.where.on.the.page.2e6785",{})}/><textarea aria-label={t("new-form.where.on.the.page.2e6785",{})} value={location} onChange={e => setLocation(e.target.value)} rows={2}/><small><UiText notice={uiMessage("new-form.value.400.characters.c1548a",{v0:uiValue([...location].length)})}/></small></label>
      <label className={styles.field}><UiText notice={uiMessage("new-form.details.45989d",{})}/><textarea aria-label={t("new-form.details.45989d",{})} value={message} onChange={e => setMessage(e.target.value)} rows={8} required/><small><UiText notice={uiMessage("new-form.value.4000.characters.9daa9f",{v0:uiValue([...message].length)})}/></small></label></fieldset>
      <button className="button" disabled={locked || !feedbackInputSchemas.create.safeParse(input).success}><UiText notice={uiMessage("new-form.submit.report.b41fd5",{})}/></button><FeedbackCommandStatus command={command}/>
      {command.error?.code === 'FEEDBACK_TARGET_STALE' && <button type="button" className="button secondary" disabled={command.busy || refreshingTarget} onClick={() => void refreshTarget()}><UiText notice={uiMessage("new-form.refresh.report.target.42c32c",{})}/></button>}
      {refreshingTarget && <p role="status"><UiText notice={uiMessage("new-form.refreshing.report.target.facb17",{})}/></p>}
      {targetRefreshed && <p role="status"><UiText notice={uiMessage("new-form.report.target.refreshed.review.the.version.before.submitting.agai.33a9e0",{})}/></p>}
      {targetError && <FeedbackState error={targetError} onRetry={() => void refreshTarget()}/>}
    </form></section>;
}
