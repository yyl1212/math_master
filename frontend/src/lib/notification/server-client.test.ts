import { it, expect, vi, afterEach } from 'vitest';
import { notificationServer } from './server-client';
import { id, json, metadata } from './test-fixtures';
vi.mock('server-only', () => ({}));
const jar = vi.hoisted(() => ({ cookies: vi.fn() }));
vi.mock('next/headers', () => ({ cookies: jar.cookies }));
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.useRealTimers(); vi.resetAllMocks(); });
function setup() { vi.stubEnv('GO_API_INTERNAL_URL', 'http://127.0.0.1:18080'); vi.stubEnv('AUTH_PUBLIC_ORIGIN', 'http://127.0.0.1:3000'); vi.stubEnv('APP_ENV', 'development'); }
it('reads current session cookies separately with no SSR cache', async () => { setup(); const fetcher = vi.fn<typeof fetch>(async () => json(metadata())); vi.stubGlobal('fetch', fetcher); for (const c of ['A', 'B']) {
    jar.cookies.mockResolvedValue({ toString: () => `other=secret; mm_session_dev=${c.repeat(43)}` });
    await notificationServer.read(id);
} expect(fetcher.mock.calls.map(c => new Headers(c[1]?.headers).get('Cookie'))).toEqual(['mm_session_dev=' + 'A'.repeat(43), 'mm_session_dev=' + 'B'.repeat(43)]); expect(fetcher.mock.calls[0]?.[1]).toMatchObject({ cache: 'no-store', redirect: 'error' }); });
it('bounds the whole SSR request including a hanging cookies read', async () => { setup(); vi.useFakeTimers(); jar.cookies.mockReturnValue(new Promise(() => { })); const fetcher = vi.fn(); vi.stubGlobal('fetch', fetcher); const observed = notificationServer.read(id).catch(e => e); await vi.advanceTimersByTimeAsync(10001); expect(await observed).toMatchObject({ code: 'SERVICE_UNAVAILABLE' }); expect(fetcher).not.toHaveBeenCalled(); });
