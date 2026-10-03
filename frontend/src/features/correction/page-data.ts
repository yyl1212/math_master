import 'server-only';
import { cookies } from 'next/headers';
import { readServerSession } from '@/lib/auth/server-client';
import { correctionAwait, withCorrectionDeadline } from '@/lib/correction/bytes';
import { CorrectionRequestError, type Envelope, type ReadAccess, type PageQuery } from '@/lib/correction/types';
import { queryPath } from '@/lib/correction/schemas';
export function correctionPage<T>(read: (access: ReadAccess) => Promise<Envelope<T>>, management = false, external?: AbortSignal): Promise<Envelope<T>> {
    return withCorrectionDeadline(external, async (signal) => {
        const jar = await correctionAwait(cookies(), signal), session = await correctionAwait(readServerSession(jar.toString()), signal);
        if (!session.ok)
            throw new CorrectionRequestError(session.code === 'AUTHENTICATION_REQUIRED' ? 'AUTHENTICATION_REQUIRED' : 'SERVICE_UNAVAILABLE');
        const user = session.data;
        if (!user)
            throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
        if (user.mustChangePassword)
            throw new CorrectionRequestError('PASSWORD_CHANGE_REQUIRED');
        if (management && !user.roles.some(r => ['editor', 'reviewer', 'admin'].includes(r)))
            throw new CorrectionRequestError('FORBIDDEN');
        const result = await correctionAwait(read({ actorId: user.id, signal }), signal);
        if (result.actorId !== user.id)
            throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
        return result;
    });
}
export function correctionPageQuery(query: Record<string, string | string[] | undefined>): PageQuery {
    let limit: number | undefined, cursor: string | undefined;
    for (const [key, value] of Object.entries(query)) {
        if (typeof value !== 'string' || !value)
            throw new CorrectionRequestError('INVALID_REQUEST');
        if (key === 'limit') {
            if (!/^[1-9][0-9]*$/.test(value))
                throw new CorrectionRequestError('INVALID_REQUEST');
            limit = Number(value);
        }
        else if (key === 'cursor')
            cursor = value;
        else
            throw new CorrectionRequestError('INVALID_REQUEST');
    }
    const out = { ...(limit === undefined ? {} : { limit }), ...(cursor === undefined ? {} : { cursor }) };
    queryPath('/api/v1/corrections/cases', out);
    return out;
}
export function correctionPageError(e: unknown) { const closed = e instanceof CorrectionRequestError ? e : new CorrectionRequestError(); return { code: closed.code, message: closed.message, ...(closed.retryAt ? { retryAt: closed.retryAt } : {}) }; }
