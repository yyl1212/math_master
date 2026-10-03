import { z } from 'zod';
import { CorrectionRequestError } from '../correction/types';
import { uuidSchema, timeSchema, evidenceRefSchema, pageSchema, envelopeSchema, strictPrivatePath, validatePageQuery, readPrivateCorrectionJSON } from '../correction/schemas';
import type { Metadata, NotificationRoute, ReadReceipt } from './types';
export const metadataSchema = z.object({ id: uuidSchema, type: z.enum(['checking', 'corrected', 'retake', 'review_material', 'path_unavailable']), evidence: evidenceRefSchema, caseId: uuidSchema, resultId: uuidSchema.nullable(), createdAt: timeSchema, readAt: timeSchema.nullable() }).strict() satisfies z.ZodType<Metadata>;
export const notificationPageSchema = pageSchema(metadataSchema), unreadCountSchema = z.object({ count: z.number().int().min(0).max(Number.MAX_SAFE_INTEGER) }).strict(), readReceiptSchema = z.object({ status: z.literal(200), notificationId: uuidSchema, readAt: timeSchema }).strict() satisfies z.ZodType<ReadReceipt>;
export const notificationReadInputSchema = z.object({}).strict();
export function resolveNotificationRoute(path: string, method = 'GET'): NotificationRoute { if (path === '/api/v1/notifications/')
    throw new CorrectionRequestError('INVALID_REQUEST'); let p: string[], query: URLSearchParams; if (path === '/api/v1/notifications' || path.startsWith('/api/v1/notifications?')) {
    const v = strictPrivatePath(path.replace('/api/v1/notifications', '/api/v1/notifications/'), '/api/v1/notifications/');
    p = v.parts;
    query = v.query;
}
else {
    const v = strictPrivatePath(path, '/api/v1/notifications/');
    p = v.parts;
    query = v.query;
} let action: NotificationRoute['action'], wanted: NotificationRoute['method'] = 'GET'; if (p.length === 0)
    action = 'list';
else if (p.length === 1 && p[0] === 'count')
    action = 'count';
else if (p.length === 1 && uuidSchema.safeParse(p[0]).success)
    action = 'read';
else if (p.length === 2 && uuidSchema.safeParse(p[0]).success && p[1] === 'read') {
    action = 'markRead';
    wanted = 'POST';
}
else
    throw new CorrectionRequestError('INVALID_REQUEST'); if (method !== wanted)
    throw new CorrectionRequestError('METHOD_NOT_ALLOWED'); validatePageQuery(query, action === 'list'); return { path, method: wanted, action }; }
function schemaFor(action: NotificationRoute['action']) { switch (action) {
    case 'list': return notificationPageSchema;
    case 'count': return unreadCountSchema;
    case 'read': return metadataSchema;
    case 'markRead': return readReceiptSchema;
} }
export async function readNotificationResponse(response: Response, path: string, signal: AbortSignal, method = 'GET') { try {
    const r = resolveNotificationRoute(path, method);
    const out = envelopeSchema(schemaFor(r.action)).parse(await readPrivateCorrectionJSON(response, signal));
    if (response.status !== 200)
        throw new Error();
    return out;
}
catch (e) {
    throw e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
} }
