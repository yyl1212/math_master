import 'server-only';
import { cookies } from 'next/headers';
import { getGoOrigin } from '../api/server-config';
import { getAuthConfig, AuthNotConfiguredError } from '../auth/config';
import { selectAuthCookies } from '../auth/cookies';
import { feedbackAwait, withFeedbackDeadline } from './bytes';
import { readFeedbackResponse, resolveFeedbackRoute } from './schemas';
import { FeedbackRequestError, type Envelope } from './types';

export function readFeedbackServer<T>(path: string): Promise<Envelope<T>> {
  return withFeedbackDeadline(undefined, async active => {
    resolveFeedbackRoute(path);
    let config;
    try { config = getAuthConfig(); } catch (e) { if (e instanceof AuthNotConfiguredError) throw new FeedbackRequestError('FEEDBACK_NOT_CONFIGURED'); throw e; }
    const origin = getGoOrigin();
    const jar = await feedbackAwait(cookies(), active);
    const cookie = selectAuthCookies(jar.toString(), config.production, true);
    const response = await feedbackAwait(fetch(origin + path, { method: 'GET', cache: 'no-store', redirect: 'error', signal: active, headers: { Accept: 'application/json', ...(cookie ? { Cookie: cookie } : {}) } }), active);
    return await feedbackAwait(readFeedbackResponse(response, path, active), active) as Envelope<T>;
  });
}
