import 'server-only';
import { getGoOrigin } from './server-config';
import { getAuthConfig, AuthNotConfiguredError } from '../auth/config';
import { selectAuthCookies } from '../auth/cookies';
import { secretPattern } from '../auth/schemas';
import { feedbackAwait, readFeedbackBytes, validateFeedbackBytes, withFeedbackDeadline } from '../feedback/bytes';
import { feedbackUUID, readFeedbackResponse, resolveFeedbackRoute } from '../feedback/schemas';
import { FeedbackRequestError } from '../feedback/types';

function privateHeaders(requestId = 'unavailable'): Headers {
  return new Headers({ 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff', 'X-Request-ID': requestId });
}
export function feedbackProxyError(error = new FeedbackRequestError()): Response {
  const headers = privateHeaders(error.requestId);
  if (error.retryAt) headers.set('Retry-After', String(Math.max(1, Math.ceil((Date.parse(error.retryAt) - Date.now()) / 1000))));
  return Response.json({ error: { code: error.code, message: error.message, requestId: error.requestId, ...(error.retryAt ? { retryAt: error.retryAt } : {}) } }, { status: error.status, headers });
}

export async function proxyFeedback(request: Request, segments: readonly string[]): Promise<Response> {
  try {
    return await withFeedbackDeadline(request.signal, async active => {
      const url = new URL(request.url), path = url.pathname + url.search;
      const route = resolveFeedbackRoute(path, request.method);
      if (url.pathname !== '/api/v1/feedback/' + segments.join('/')) throw new FeedbackRequestError('INVALID_REQUEST');
      let config;
      try { config = getAuthConfig(); } catch (e) { if (e instanceof AuthNotConfiguredError) throw new FeedbackRequestError('FEEDBACK_NOT_CONFIGURED'); throw e; }
      const origin = getGoOrigin(), write = route.method === 'POST';
      if (request.headers.get('Sec-Fetch-Site') === 'cross-site' || (write || request.headers.has('Origin')) && request.headers.get('Origin') !== config.publicOrigin || request.headers.get('X-CSRF-Token')?.includes(',') || write && !secretPattern.test(request.headers.get('X-CSRF-Token') ?? '')) throw new FeedbackRequestError('CSRF_FAILED');
      if (request.headers.get('Idempotency-Key')?.includes(',') || write && !feedbackUUID.test(request.headers.get('Idempotency-Key') ?? '')) throw new FeedbackRequestError('INVALID_REQUEST');
      const headers = new Headers({ Accept: 'application/json' });
      const cookie = selectAuthCookies(request.headers.get('Cookie') ?? '', config.production, true);
      if (cookie) headers.set('Cookie', cookie);
      for (const name of ['Origin', 'X-CSRF-Token', 'Sec-Fetch-Site']) { const v = request.headers.get(name); if (v !== null) headers.set(name, v); }
      let body: ArrayBuffer | undefined;
      if (write) {
        if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(request.headers.get('Content-Type') ?? '')) throw new FeedbackRequestError('INVALID_REQUEST');
        let raw: Uint8Array;
        try { raw = await readFeedbackBytes(new Response(request.body, { headers: request.headers.has('Content-Length') ? { 'Content-Length': request.headers.get('Content-Length')! } : {} }), 65536, active); }
        catch { throw new FeedbackRequestError(active.aborted ? 'SERVICE_UNAVAILABLE' : 'INVALID_REQUEST'); }
        validateFeedbackBytes(raw, route.action);
        body = raw.slice().buffer as ArrayBuffer;
        headers.set('Content-Type', 'application/json'); headers.set('Idempotency-Key', request.headers.get('Idempotency-Key')!);
      } else if (request.body !== null) throw new FeedbackRequestError('INVALID_REQUEST');
      const response = await feedbackAwait(fetch(origin + path, { method: route.method, cache: 'no-store', redirect: 'error', signal: active, headers, ...(body === undefined ? {} : { body }) }), active);
      const result = await feedbackAwait(readFeedbackResponse(response, path, active, route.method), active);
      return Response.json(result, { status: response.status, headers: privateHeaders(response.headers.get('X-Request-ID')!) });
    });
  } catch (e) { return feedbackProxyError(e instanceof FeedbackRequestError ? e : new FeedbackRequestError()); }
}
