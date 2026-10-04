import { it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { z } from 'zod';
import { validateNotificationBytes } from './bytes';
import { metadataSchema, resolveNotificationRoute, readNotificationResponse } from './schemas';
import { metadata, id, json, cursor } from './test-fixtures';
it('all shared Go notification boundary cases agree', () => { const corpus = z.object({ version: z.literal(1), requests: z.array(z.object({ name: z.string(), domain: z.string(), action: z.string(), raw: z.string(), valid: z.boolean() })) }).parse(JSON.parse(readFileSync('../api/correction-boundary-cases.json', 'utf8'))); for (const c of corpus.requests.filter(c => c.domain === 'notification')) {
    let accepted = true;
    try {
        validateNotificationBytes(new TextEncoder().encode(c.raw));
    }
    catch {
        accepted = false;
    }
    expect(accepted, c.name).toBe(c.valid);
} });
it('notification metadata requires explicit nulls and no body or other actor', () => { expect(metadataSchema.safeParse(metadata()).success).toBe(true); for (const k of ['resultId', 'readAt']) {
    const v: Record<string, unknown> = { ...metadata() };
    delete v[k];
    expect(metadataSchema.safeParse(v).success).toBe(false);
} for (const k of ['title', 'message', 'ownerId', 'leaseToken', 'answer'])
    expect(metadataSchema.safeParse({ ...metadata(), [k]: 'secret' }).success).toBe(false); });
it('validates its four fixed routes and canonical owner-scoped cursor structure', () => { expect(resolveNotificationRoute('/api/v1/notifications?cursor=' + cursor()).action).toBe('list'); expect(resolveNotificationRoute('/api/v1/notifications/count').action).toBe('count'); expect(resolveNotificationRoute('/api/v1/notifications/' + id + '/read', 'POST').action).toBe('markRead'); for (const p of ['/api/v1/notifications?limit=01', '/api/v1/notifications?cursor=e30', '/api/v1/notifications?cursor=x&cursor=y', '/api/v1/notifications/count?limit=1', '/api/v1/notifications/' + id + '/detail'])
    expect(() => resolveNotificationRoute(p)).toThrow(); });
it('does not accept free-text notifications or impossible read receipts', async () => { const signal = new AbortController().signal; await expect(readNotificationResponse(json({ ...metadata(), message: 'private' }), '/api/v1/notifications/' + id, signal)).rejects.toMatchObject({ code: 'SERVICE_UNAVAILABLE' }); await expect(readNotificationResponse(json({ status: 201, notificationId: id, readAt: metadata().createdAt }), '/api/v1/notifications/' + id + '/read', signal, 'POST')).rejects.toMatchObject({ code: 'SERVICE_UNAVAILABLE' }); });
it('rejects empty raw query pairs and the noncanonical trailing slash root', () => { for (const path of ['/api/v1/notifications/', '/api/v1/notifications?limit=1&', '/api/v1/notifications?cursor=%'])
    expect(() => resolveNotificationRoute(path)).toThrow(); });
