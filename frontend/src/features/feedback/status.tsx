"use client";
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import type { useFeedbackCommand } from './pending-command';
import { FeedbackRequestError, type Metadata } from '@/lib/feedback/types';
export const feedbackStatusLabels = { new: 'New', processing: 'In progress', waiting_details: 'Waiting for details', resolved: 'Resolved', closed: 'Closed' };
export function FeedbackStatus({ status }: { status: Metadata['status'] }) { return <span data-feedback-status={status}>{feedbackStatusLabels[status]}</span>; }
export function FeedbackState({ error }: { error: FeedbackRequestError }) {
  const router = useRouter();
  return <section className="content-state" role="alert"><h2>{error.message}</h2>{error.retryAt && <p>Try after {new Date(error.retryAt).toLocaleString('en')}</p>}
    {error.code === 'AUTHENTICATION_REQUIRED' ? <Link prefetch={false} href="/login" className="button">Sign in</Link> : error.code === 'PASSWORD_CHANGE_REQUIRED' || error.code === 'FORBIDDEN' ? <Link prefetch={false} href="/account" className="button secondary">View account</Link> : <button type="button" className="button secondary" onClick={() => router.refresh()}>Reload page</button>}</section>;
}
export function FeedbackCommandStatus({ command }: { command: Pick<ReturnType<typeof useFeedbackCommand>, 'busy' | 'error' | 'pending' | 'retry' | 'clear'> }) {
  return <div aria-live="polite">{command.busy && <p role="status">Saving…</p>}{command.error && <div role="alert"><p>{command.error.code === 'SERVICE_UNAVAILABLE' ? 'No confirmation received. You can retry the same request.' : command.error.message}</p>{command.error.retryAt && <p>Try after {new Date(command.error.retryAt).toLocaleString('en')}</p>}{command.pending && <div className="button-group"><button type="button" className="button secondary" disabled={command.busy} onClick={() => void command.retry()}>Retry same request</button><button type="button" className="button secondary" disabled={command.busy} onClick={command.clear}>Edit as a new request</button></div>}</div>}</div>;
}
