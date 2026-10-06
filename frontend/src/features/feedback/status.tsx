"use client";
import {UiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import type { useFeedbackCommand } from './pending-command';
import { FeedbackRequestError, type Metadata } from '@/lib/feedback/types';
export const feedbackStatusLabels = { new: 'New', processing: 'In progress', waiting_details: 'Waiting for details', resolved: 'Resolved', closed: 'Closed' };
export function FeedbackStatus({ status }: { status: Metadata['status'] }) { return <span data-feedback-status={status}><UiEnum group="feedback.status" value={status}/></span>; }
export function FeedbackState({ error, onRetry }: { error: Pick<FeedbackRequestError, "code" | "message" | "retryAt">; onRetry?: () => void }) {
  const router = useRouter();
  return <section className="content-state" role="alert"><h2><UiText notice={uiError("feedback",error)}/></h2>{error.retryAt && <p><UiText notice={uiMessage("learning-status.try.after.value.ea24ae",{v0:uiValue(new Date(error.retryAt).toLocaleString('en'))})}/></p>}
    {error.code === 'AUTHENTICATION_REQUIRED' ? <Link prefetch={false} href="/login" className="button"><UiText notice={uiMessage("page.login",{})}/></Link> : error.code === 'PASSWORD_CHANGE_REQUIRED' || error.code === 'FORBIDDEN' ? <Link prefetch={false} href="/account" className="button secondary"><UiText notice={uiMessage("auth-state.view.account.407143",{})}/></Link> : <button type="button" className="button secondary" onClick={onRetry ?? (() => router.refresh())}><UiText notice={uiMessage("status.reload.page.437d0d",{})}/></button>}</section>;
}
export function FeedbackCommandStatus({ command }: { command: Pick<ReturnType<typeof useFeedbackCommand>, 'busy' | 'error' | 'pending' | 'confirmed' | 'retry' | 'clear'> }) {
 const {t}=useUiI18n();

  return <div aria-live="polite">{command.busy && <p role="status"><UiText notice={uiMessage("learning-status.saving.23e392",{})}/></p>}{command.error && <div role="alert"><p>{command.confirmed ? t("status.your.change.was.saved.retry.the.same.request.to.load.the.latest.s.01d912",{}) : command.error.code === 'SERVICE_UNAVAILABLE' ? t("learning-status.no.confirmation.received.you.can.retry.the.same.request.e68eb0",{}) : <UiText notice={uiError("feedback",command.error)}/>}</p>{command.error.retryAt && <p><UiText notice={uiMessage("learning-status.try.after.value.ea24ae",{v0:uiValue(new Date(command.error.retryAt).toLocaleString('en'))})}/></p>}{command.pending && <div className="button-group"><button type="button" className="button secondary" disabled={command.busy} onClick={() => void command.retry()}><UiText notice={uiMessage("learning-status.retry.same.request.16003a",{})}/></button>{!command.confirmed && <button type="button" className="button secondary" disabled={command.busy} onClick={command.clear}><UiText notice={uiMessage("status.edit.as.a.new.request.356c78",{})}/></button>}</div>}</div>}</div>;
}
