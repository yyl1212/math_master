import { getAuthContext, notifyAuthChanged } from '../auth/client';
import { feedbackAwait, validateFeedbackBytes, withFeedbackDeadline } from './bytes';
import { feedbackUUID, readFeedbackResponse, resolveFeedbackRoute } from './schemas';
import { FeedbackRequestError, type Context, type ContextQuery, type Envelope, type FeedbackCommand, type Receipt } from './types';

export function readFeedback<T>(path: string, signal: AbortSignal): Promise<Envelope<T>> {
  return withFeedbackDeadline(signal, async active => {
    resolveFeedbackRoute(path);
    const response = await feedbackAwait(fetch(path, { method: 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: { Accept: 'application/json' }, signal: active }), active);
    return await feedbackAwait(readFeedbackResponse(response, path, active), active) as Envelope<T>;
  });
}

export function getFeedbackContext(query: ContextQuery, signal: AbortSignal): Promise<Envelope<Context>> {
  if (Object.keys(query).some(k => !['kind', 'id', 'area', 'position', 'partKind', 'partId'].includes(k))) return Promise.reject(new FeedbackRequestError('INVALID_REQUEST'));
  const params = new URLSearchParams();
  for (const k of ['area', 'position', 'partKind', 'partId'] as const) if (query[k] !== undefined) params.set(k, String(query[k]));
  const path = '/api/v1/feedback/contexts/' + query.kind + (query.kind === 'site' ? '' : '/' + query.id) + (params.size ? '?' + params : '');
  return readFeedback<Context>(path, signal);
}

export function sendFeedback(command: FeedbackCommand, signal: AbortSignal): Promise<Envelope<Receipt>> {
  return withFeedbackDeadline(signal, async active => {
    if (Object.keys(command).length !== 4 || !feedbackUUID.test(command.actorId) || !feedbackUUID.test(command.key)) throw new FeedbackRequestError('INVALID_REQUEST');
    const route = resolveFeedbackRoute(command.route, 'POST');
    const body = JSON.stringify(command.input);
    validateFeedbackBytes(new TextEncoder().encode(body), route.action);
    const actorId = command.actorId, key = command.key, path = command.route;
    const context = await feedbackAwait(getAuthContext(true), active);
    if (!context.ok) throw new FeedbackRequestError(context.code === 'AUTHENTICATION_REQUIRED' ? 'AUTHENTICATION_REQUIRED' : context.code === 'PASSWORD_CHANGE_REQUIRED' ? 'PASSWORD_CHANGE_REQUIRED' : 'SERVICE_UNAVAILABLE');
    if (!context.data.user || context.data.user.id !== actorId) { notifyAuthChanged(); throw new FeedbackRequestError('AUTHENTICATION_REQUIRED'); }
    if (context.data.user.mustChangePassword) { notifyAuthChanged(); throw new FeedbackRequestError('PASSWORD_CHANGE_REQUIRED'); }
    const response = await feedbackAwait(fetch(path, { method: 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error', signal: active, headers: { Accept: 'application/json', 'Content-Type': 'application/json', 'X-CSRF-Token': context.data.csrfToken, 'Idempotency-Key': key }, body }), active);
    const result = await feedbackAwait(readFeedbackResponse(response, path, active, 'POST'), active) as Envelope<Receipt>;
    if (result.actorId !== actorId) { notifyAuthChanged(); throw new FeedbackRequestError('AUTHENTICATION_REQUIRED'); }
    return result;
  });
}
