import 'server-only';
import { cookies } from 'next/headers';
import { readServerSession } from '@/lib/auth/server-client';
import { readFeedbackServer } from '@/lib/feedback/server-client';
import { feedbackUUID, resolveFeedbackRoute } from '@/lib/feedback/schemas';
import { FeedbackRequestError, type ContextQuery, type Envelope } from '@/lib/feedback/types';
export async function feedbackPage<T>(path: string, review = false): Promise<Envelope<T>> {
  const session = await readServerSession((await cookies()).toString());
  if (!session.ok) throw new FeedbackRequestError(session.code === 'AUTHENTICATION_REQUIRED' ? session.code : 'SERVICE_UNAVAILABLE');
  if (!session.data) throw new FeedbackRequestError('AUTHENTICATION_REQUIRED');
  if (session.data.mustChangePassword) throw new FeedbackRequestError('PASSWORD_CHANGE_REQUIRED');
  if (review && !session.data.roles.some(r => r === 'reviewer' || r === 'admin')) throw new FeedbackRequestError('FORBIDDEN');
  const result = await readFeedbackServer<T>(path);
  if (result.actorId !== session.data.id) throw new FeedbackRequestError('AUTHENTICATION_REQUIRED');
  return result;
}
export function feedbackListPath(query: Record<string, string | string[] | undefined>, review: boolean): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) { if (typeof value !== 'string' || !value || value === '' || !['limit', 'cursor', ...(review ? ['status', 'category'] : [])].includes(key)) throw new FeedbackRequestError('INVALID_REQUEST'); params.set(key, value); }
  const path = '/api/v1/feedback/' + (review ? 'review/' : '') + 'tickets' + (params.size ? '?' + params : ''); resolveFeedbackRoute(path); return path;
}
export function feedbackSourcePath(query: Record<string, string | string[] | undefined>): string {
  for (const [key, value] of Object.entries(query)) if (!['kind', 'id', 'area', 'position', 'partKind', 'partId'].includes(key) || typeof value !== 'string' || !value) throw new FeedbackRequestError('INVALID_REQUEST');
  const kind = query.kind as ContextQuery['kind'];
  if (!['site', 'managed-knowledge', 'knowledge', 'path', 'practice', 'assessment'].includes(kind) || kind === 'site' && query.id !== undefined || kind !== 'site' && typeof query.id !== 'string') throw new FeedbackRequestError('INVALID_REQUEST');
  const params = new URLSearchParams(); for (const k of ['area', 'position', 'partKind', 'partId']) if (typeof query[k] === 'string') params.set(k, query[k]);
  const path = '/api/v1/feedback/contexts/' + kind + (kind === 'site' ? '' : '/' + query.id) + (params.size ? '?' + params : ''); resolveFeedbackRoute(path); return path;
}
export function feedbackTicketPath(id: string, review: boolean): string { if (!feedbackUUID.test(id)) throw new FeedbackRequestError('NOT_FOUND'); return '/api/v1/feedback/' + (review ? 'review/' : '') + 'tickets/' + id; }
export function feedbackPageError(e: unknown) { const closed=e instanceof FeedbackRequestError?e:new FeedbackRequestError();return {code:closed.code,message:closed.message,...(closed.retryAt?{retryAt:closed.retryAt}:{})}; }
