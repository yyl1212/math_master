"use client";
import { useContext, useEffect, useRef, useState } from 'react';
import { getAuthContext } from '@/lib/auth/client';
import { sendFeedback } from '@/lib/feedback/client';
import { feedbackAwait, validateFeedbackBytes, withFeedbackDeadline } from '@/lib/feedback/bytes';
import { feedbackUUID, resolveFeedbackRoute } from '@/lib/feedback/schemas';
import { FeedbackRequestError, type FeedbackCommand, type Receipt } from '@/lib/feedback/types';
import { FeedbackAccountContext } from './feedback-account';
export type FeedbackCommandInput = FeedbackCommand extends infer C ? C extends FeedbackCommand ? Omit<C, 'actorId' | 'key'> : never : never;
function freeze<T>(value: T): T { if (value !== null && typeof value === 'object') { for (const v of Object.values(value)) freeze(v); Object.freeze(value); } return value; }
export function createPendingFeedbackCommand(actorId: string, command: FeedbackCommandInput): FeedbackCommand {
  if (!feedbackUUID.test(actorId)) throw new FeedbackRequestError('INVALID_REQUEST');
  const route = resolveFeedbackRoute(command.route, 'POST'), raw = JSON.stringify(command.input);
  validateFeedbackBytes(new TextEncoder().encode(raw), route.action);
  return freeze({ actorId, key: crypto.randomUUID(), route: command.route, input: JSON.parse(raw) }) as FeedbackCommand;
}
export function useFeedbackCommand(actorId: string, onSuccess: (receipt: Receipt, signal: AbortSignal) => Promise<void>) {
  const account = useContext(FeedbackAccountContext), [pending, setPending] = useState<FeedbackCommand | null>(null), [error, setError] = useState<FeedbackRequestError | null>(null), [busy, setBusy] = useState(false), [confirmed, setConfirmed] = useState(false);
  const live = useRef(true), generation = useRef(0), inFlight = useRef(false), controller = useRef<AbortController | null>(null), success = useRef(onSuccess); success.current = onSuccess;
  const clear = () => { generation.current++; controller.current?.abort(); inFlight.current = false; setPending(null); setConfirmed(false); setError(null); setBusy(false); };
  useEffect(() => { live.current = true; window.addEventListener('math-master:auth-change', clear); return () => { live.current = false; generation.current++; controller.current?.abort(); window.removeEventListener('math-master:auth-change', clear); }; }, [actorId]);
  const perform = async (command: FeedbackCommand) => {
    if (inFlight.current) return;
    if (!account || account.actorId !== actorId || command.actorId !== actorId) { clear(); account?.invalidate(); return; }
    inFlight.current = true; setBusy(true); setError(null); const current = ++generation.current, active = new AbortController(); controller.current = active;
    const valid = () => live.current && generation.current === current;
    try { await withFeedbackDeadline(active.signal, async signal => {
      const context = await feedbackAwait(getAuthContext(true), signal);
      if (!valid()) return;
      if (!context.ok) throw new FeedbackRequestError(context.code === 'AUTHENTICATION_REQUIRED' ? context.code : 'SERVICE_UNAVAILABLE');
      if (context.data.user?.id !== actorId || context.data.user.mustChangePassword) { clear(); account.invalidate(); return; }
      setPending(command);
      const result = await feedbackAwait(sendFeedback(command, signal), signal);
      if (!valid()) return;
      if (result.actorId !== actorId) { clear(); account.invalidate(); return; }
      setConfirmed(true);
      await feedbackAwait(success.current(result.data, signal), signal);
      if (valid() && !signal.aborted) { setPending(null); setConfirmed(false); }
    }); } catch (e) { if (valid()) { const closed = e instanceof FeedbackRequestError ? e : new FeedbackRequestError(); if (['AUTHENTICATION_REQUIRED', 'PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN'].includes(closed.code)) { clear(); account.invalidate(); } else setError(closed); } }
    finally { if (controller.current === active) controller.current = null; if (valid()) { inFlight.current = false; setBusy(false); } }
  };
  const run = async (command: FeedbackCommandInput) => { if (inFlight.current || pending) return; try { await perform(createPendingFeedbackCommand(actorId, command)); } catch (e) { setError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); } };
  const retry = async () => { if (pending) await perform(pending); };
  return { run, retry, clear, busy, pending, confirmed, error };
}
