import { it, expect, vi, afterEach } from 'vitest';
import { proxyNotification } from './notification-proxy';
import { id, otherId, json, metadata } from '../notification/test-fixtures';
vi.mock('server-only', () => ({}));
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.useRealTimers(); });
function setup(f: typeof fetch) { vi.stubGlobal('fetch', f); vi.stubEnv('GO_API_INTERNAL_URL', 'http://127.0.0.1:18080'); vi.stubEnv('AUTH_PUBLIC_ORIGIN', 'http://127.0.0.1:3000'); vi.stubEnv('APP_ENV', 'development'); }
const path = '/api/v1/notifications/' + id + '/read';
function req(raw: string, extra: Record<string, string> = {}) { return new Request('http://127.0.0.1:3000' + path, { method: 'POST', headers: { Cookie: 'other=secret; mm_session_dev=' + 'A'.repeat(43), Origin: 'http://127.0.0.1:3000', 'X-CSRF-Token': 'A'.repeat(43), 'Idempotency-Key': otherId, 'Content-Type': 'application/json', ...extra }, body: raw }); }
it('forwards the exact validated bytes and only private proof headers', async () => { const fetcher = vi.fn<typeof fetch>(async () => json({ status: 200, notificationId: id, readAt: metadata().createdAt }, 200)); setup(fetcher); const raw = ' \n' + '{}' + ' \t'; const r = await proxyNotification(req(raw), [id, 'read']); expect(r.status).toBe(200); expect(r.headers.get('Cache-Control')).toBe('private, no-store'); expect(new TextDecoder().decode(new Uint8Array(await new Response(fetcher.mock.calls[0]?.[1]?.body).arrayBuffer()))).toBe(raw); expect(new Headers(fetcher.mock.calls[0]?.[1]?.headers).get('Cookie')).toBe('mm_session_dev=' + 'A'.repeat(43)); });
it('bounds raw bytes and rejects unknown props and duplicate proof headers before Go', async () => { for (const n of [65536, 65537]) {
    const fetcher = vi.fn<typeof fetch>(async () => json({ status: 200, notificationId: id, readAt: metadata().createdAt }, 200));
    setup(fetcher);
    const raw = '{}';
    expect((await proxyNotification(req(raw + ' '.repeat(n - raw.length)), [id, 'read'])).status).toBe(n === 65536 ? 200 : 400);
} const fetcher = vi.fn(); setup(fetcher); for (const request of [req('{"extra":"private"}'), req('{}', { 'Idempotency-Key': id + ', ' + id }), req('{}', { 'X-CSRF-Token': 'A'.repeat(43) + ', ' + 'A'.repeat(43) }), req('{}', { Origin: 'https://foreign.test' })])
    expect((await proxyNotification(request, [id, 'read'])).status).toBeGreaterThanOrEqual(400); expect((await proxyNotification(req('{}'), ['auth', 'context'])).status).toBe(400); expect(fetcher).not.toHaveBeenCalled(); });
it('sanitizes overlimit private responses and includes asynchronous route params in its deadline', async () => { setup(vi.fn<typeof fetch>(async () => json({ status: 200, notificationId: id, readAt: metadata().createdAt }, 200, id, { 'Content-Length': '2097153' }))); expect((await proxyNotification(req('{}'), [id, 'read'])).status).toBe(503); vi.useFakeTimers(); const fetcher = vi.fn(); setup(fetcher); const pending = proxyNotification(req('{}'), new Promise(() => { })); await vi.advanceTimersByTimeAsync(10001); expect((await pending).status).toBe(503); expect(fetcher).not.toHaveBeenCalled(); });
