"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { getAuthContext } from '@/lib/auth/client';
import { readFeedback } from '@/lib/feedback/client';
import { feedbackAwait, withFeedbackDeadline } from '@/lib/feedback/bytes';
import { FeedbackRequestError, type Envelope } from '@/lib/feedback/types';
import { FeedbackState } from './status';
export const FeedbackAccountContext = createContext<{ actorId: string; invalidate: () => void } | null>(null);
type AccountProps = { actorId: string; review?: boolean; children: React.ReactNode };
export function FeedbackAccountProvider(props: AccountProps) {
  // Actor changes replace the entire private subtree before a new identity can verify.
  return <FeedbackAccountBoundary key={props.actorId} {...props}/>;
}
function FeedbackAccountBoundary({ actorId, review = false, children }: AccountProps) {
  const [verified, setVerified] = useState(false), [error, setError] = useState<FeedbackRequestError | null>(null);
  const invalid = useRef(false), [attempt, setAttempt] = useState(0);
  const invalidate = useCallback(() => { invalid.current = true; setVerified(false); setError(new FeedbackRequestError('AUTHENTICATION_REQUIRED')); }, []);
  useEffect(() => {
    let live = true, revision = 0;
    // New server children are a fresh authenticated snapshot; focus alone never
    // revives a subtree invalidated by an account change.
    invalid.current = false;
    const verify = () => { const current = ++revision; void withFeedbackDeadline(undefined, async signal => {
      const context = await feedbackAwait(getAuthContext(true), signal);
      if (!live || current !== revision || invalid.current) return;
      if (!context.ok) throw new FeedbackRequestError(context.code === 'AUTHENTICATION_REQUIRED' ? context.code : 'SERVICE_UNAVAILABLE');
      if (context.data.user?.id !== actorId) { invalidate(); return; }
      if (context.data.user.mustChangePassword) throw new FeedbackRequestError('PASSWORD_CHANGE_REQUIRED');
      if (review && !context.data.user.roles.some(r => r === 'reviewer' || r === 'admin')) throw new FeedbackRequestError('FORBIDDEN');
      setError(null); setVerified(true);
    }).catch(e => { if (live && current === revision) { setVerified(false); setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); } }); };
    const changed = () => { revision++; invalidate(); };
    const visible = () => { if (document.visibilityState === 'visible') verify(); };
    verify(); window.addEventListener('math-master:auth-change', changed); window.addEventListener('focus', verify); document.addEventListener('visibilitychange', visible);
    let channel: BroadcastChannel | undefined;
    try { if (typeof BroadcastChannel !== 'undefined') { channel = new BroadcastChannel('math-master-auth'); channel.onmessage = e => { if (e.data === 'changed') window.dispatchEvent(new Event('math-master:auth-change')); }; } } catch { /* The command also verifies current identity. */ }
    return () => { live = false; revision++; window.removeEventListener('math-master:auth-change', changed); window.removeEventListener('focus', verify); document.removeEventListener('visibilitychange', visible); channel?.close(); };
  }, [actorId, review, invalidate, children, attempt]);
  if (error) return <FeedbackState error={error} onRetry={() => setAttempt(n => n + 1)}/>;
  return verified ? <FeedbackAccountContext.Provider value={{ actorId, invalidate }}>{children}</FeedbackAccountContext.Provider> : <p role="status"><UiText notice={uiMessage("feedback-account.checking.your.feedback.account.08c898",{})}/></p>;
}
// Every delivery is tied to the server-rendered actor and the mounted page.
export function useFeedbackRead() {
  const account = useContext(FeedbackAccountContext), live = useRef(true), generation = useRef(0), controller = useRef<AbortController | null>(null);
  useEffect(() => { live.current = true; return () => { live.current = false; generation.current++; controller.current?.abort(); }; }, []);
  return async <T,>(path: string, signal?: AbortSignal): Promise<Envelope<T> | null> => {
    if (!account) return null;
    controller.current?.abort(); const active = new AbortController(), current = ++generation.current; controller.current = active;
    try {
      const result = await readFeedback<T>(path, signal ? AbortSignal.any([active.signal, signal]) : active.signal);
      if (!live.current || current !== generation.current || active.signal.aborted || signal?.aborted) return null;
      if (result.actorId !== account.actorId) { account.invalidate(); return null; }
      return result;
    } catch (e) {
      if (!live.current || current !== generation.current || active.signal.aborted || signal?.aborted) return null;
      if (e instanceof FeedbackRequestError && ['AUTHENTICATION_REQUIRED', 'PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN'].includes(e.code)) { account.invalidate(); return null; }
      throw e;
    }
  };
}
