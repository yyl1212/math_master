import { it, expect, vi, afterEach } from 'vitest';
import { proxyCorrection } from './correction-proxy';
import { id, otherId, json, caseInput, caseMetadata } from '../correction/test-fixtures';
vi.mock('server-only', () => ({}));
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.useRealTimers(); });
function setup(f: typeof fetch) { vi.stubGlobal('fetch', f); vi.stubEnv('GO_API_INTERNAL_URL', 'http://127.0.0.1:18080'); vi.stubEnv('AUTH_PUBLIC_ORIGIN', 'http://127.0.0.1:3000'); vi.stubEnv('APP_ENV', 'development'); }
const path = '/api/v1/corrections/cases';
function req(raw: string, extra: Record<string, string> = {}) { return new Request('http://127.0.0.1:3000' + path, { method: 'POST', headers: { Cookie: 'other=secret; mm_session_dev=' + 'A'.repeat(43), Origin: 'http://127.0.0.1:3000', 'X-CSRF-Token': 'A'.repeat(43), 'Idempotency-Key': otherId, 'Content-Type': 'application/json', ...extra }, body: raw }); }
it('forwards the exact validated bytes and only private proof headers', async () => { const fetcher = vi.fn<typeof fetch>(async () => json({ status: 201, case: caseMetadata(), plan: null, job: null }, 201)); setup(fetcher); const raw = ' \n' + JSON.stringify(caseInput()) + ' \t'; const r = await proxyCorrection(req(raw), ['cases']); expect(r.status).toBe(201); expect(r.headers.get('Cache-Control')).toBe('private, no-store'); expect(new TextDecoder().decode(new Uint8Array(await new Response(fetcher.mock.calls[0]?.[1]?.body).arrayBuffer()))).toBe(raw); expect(new Headers(fetcher.mock.calls[0]?.[1]?.headers).get('Cookie')).toBe('mm_session_dev=' + 'A'.repeat(43)); });
it('bounds raw bytes and rejects unknown props and duplicate proof headers before Go', async () => { for (const n of [65536, 65537]) {
    const fetcher = vi.fn<typeof fetch>(async () => json({ status: 201, case: caseMetadata(), plan: null, job: null }, 201));
    setup(fetcher);
    const raw = JSON.stringify(caseInput());
    expect((await proxyCorrection(req(raw + ' '.repeat(n - raw.length)), ['cases'])).status).toBe(n === 65536 ? 201 : 400);
} const fetcher = vi.fn(); setup(fetcher); for (const request of [req('{"extra":"private"}'), req(JSON.stringify(caseInput()), { 'Idempotency-Key': id + ', ' + id }), req(JSON.stringify(caseInput()), { 'X-CSRF-Token': 'A'.repeat(43) + ', ' + 'A'.repeat(43) }), req(JSON.stringify(caseInput()), { Origin: 'https://foreign.test' })])
    expect((await proxyCorrection(request, ['cases'])).status).toBeGreaterThanOrEqual(400); expect((await proxyCorrection(req(JSON.stringify(caseInput())), ['auth', 'context'])).status).toBe(400); expect(fetcher).not.toHaveBeenCalled(); });
it('sanitizes overlimit private responses and includes asynchronous route params in its deadline', async () => { setup(vi.fn<typeof fetch>(async () => json({ status: 201, case: caseMetadata(), plan: null, job: null }, 201, id, { 'Content-Length': '2097153' }))); expect((await proxyCorrection(req(JSON.stringify(caseInput())), ['cases'])).status).toBe(503); vi.useFakeTimers(); const fetcher = vi.fn(); setup(fetcher); const pending = proxyCorrection(req(JSON.stringify(caseInput())), new Promise(() => { })); await vi.advanceTimersByTimeAsync(10001); expect((await pending).status).toBe(503); expect(fetcher).not.toHaveBeenCalled(); });

it('corrected SVG uses the exact private result path and validates bytes and all security headers', async () => {
 const {createHash}=await import('node:crypto');const svg='<svg xmlns="http://www.w3.org/2000/svg"><title>Original correction illustration</title></svg>';
 const sha=createHash('sha256').update(svg).digest('hex'),p='/api/v1/corrections/results/'+id+'/assets/'+sha;
 const good=()=>new Response(svg,{headers:{'Content-Type':'image/svg+xml','Content-Security-Policy':"sandbox; default-src 'none'",'Cache-Control':'private, no-store','X-Content-Type-Options':'nosniff','X-Request-ID':'a'.repeat(32)}});
 const fetcher=vi.fn<typeof fetch>(async()=>good());setup(fetcher);
 const r=await proxyCorrection(new Request('http://127.0.0.1:3000'+p,{headers:{Cookie:'other=private; mm_session_dev='+'A'.repeat(43)}}),['results',id,'assets',sha]);
 expect(r.status).toBe(200);expect(await r.text()).toBe(svg);expect(r.headers.get('Content-Security-Policy')).toBe("sandbox; default-src 'none'");expect(r.headers.get('Cache-Control')).toBe('private, no-store');
 expect(fetcher.mock.calls[0]?.[0]).toBe('http://127.0.0.1:18080'+p);expect(new Headers(fetcher.mock.calls[0]?.[1]?.headers).get('Cookie')).toBe('mm_session_dev='+'A'.repeat(43));
 for(const mutate of [(v:Response)=>v.headers.delete('Content-Security-Policy'),(v:Response)=>v.headers.set('Set-Cookie','private=bad'),(v:Response)=>v.headers.set('Content-Length','1048577')]){const v=good();mutate(v);setup(vi.fn<typeof fetch>(async()=>v));expect((await proxyCorrection(new Request('http://127.0.0.1:3000'+p),['results',id,'assets',sha])).status).toBe(503)}
 const badSha='b'.repeat(64);setup(vi.fn<typeof fetch>(async()=>good()));expect((await proxyCorrection(new Request('http://127.0.0.1:3000'+p.replace(sha,badSha)),['results',id,'assets',badSha])).status).toBe(503);
});
it('corrected SVG stream is cancelled within the same ten-second request deadline', async () => {
 vi.useFakeTimers();let cancelled=false;const stream=new ReadableStream<Uint8Array>({cancel(){cancelled=true}}),sha='a'.repeat(64),p='/api/v1/corrections/results/'+id+'/assets/'+sha;
 setup(vi.fn<typeof fetch>(async()=>new Response(stream,{headers:{'Content-Type':'image/svg+xml','Content-Security-Policy':"sandbox; default-src 'none'",'Cache-Control':'private, no-store','X-Content-Type-Options':'nosniff','X-Request-ID':'a'.repeat(32)}})));
 const pending=proxyCorrection(new Request('http://127.0.0.1:3000'+p),['results',id,'assets',sha]);await vi.advanceTimersByTimeAsync(10001);expect((await pending).status).toBe(503);expect(cancelled).toBe(true);
});
