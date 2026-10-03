import 'server-only';
import {validateContentSVG} from '../content/svg';
import {readPrivateCorrectionJSON} from '../correction/schemas';
import { getGoOrigin } from './server-config';
import { getAuthConfig, AuthNotConfiguredError } from '../auth/config';
import { selectAuthCookies } from '../auth/cookies';
import { secretPattern } from '../auth/schemas';
import { correctionAwait, withCorrectionDeadline, readCorrectionBytes, validateCorrectionBytes } from '../correction/bytes';
import { correctionUUID, resolveCorrectionRoute, readCorrectionResponse } from '../correction/schemas';
import { CorrectionRequestError } from '../correction/types';
export function correctionPrivateHeaders(requestId = 'unavailable') { return new Headers({ 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff', 'X-Request-ID': requestId }); }
export function correctionProxyError(e = new CorrectionRequestError()) { const headers = correctionPrivateHeaders(e.requestId); if (e.retryAt)
    headers.set('Retry-After', String(Math.max(1, Math.ceil((Date.parse(e.retryAt) - Date.now()) / 1000)))); return Response.json({ error: { code: e.code, message: e.message, requestId: e.requestId, ...(e.retryAt ? { retryAt: e.retryAt } : {}) } }, { status: e.status, headers }); }
type Route = {
    method: 'GET' | 'POST' | 'PUT';
    action: string;
};
export async function proxyPrivateCorrection(request: Request, segments: readonly string[] | Promise<readonly string[]>, prefix: string, routeFor: (path: string, method: string) => Route, validate: (raw: Uint8Array, action: string) => unknown, read: (response: Response, path: string, signal: AbortSignal, method: string) => Promise<unknown>): Promise<Response> {
    try {
        return await withCorrectionDeadline(request.signal, async (active) => { const selected = await correctionAwait(Promise.resolve(segments), active), url = new URL(request.url), path = url.pathname + url.search; const route = routeFor(path, request.method); if (url.pathname !== prefix + (selected.length ? '/' + selected.join('/') : ''))
            throw new CorrectionRequestError('INVALID_REQUEST'); let config; try {
            config = getAuthConfig();
        }
        catch (e) {
            if (e instanceof AuthNotConfiguredError)
                throw new CorrectionRequestError('CORRECTION_NOT_CONFIGURED');
            throw e;
        } const origin = getGoOrigin(), write = route.method !== 'GET'; if (request.headers.get('Sec-Fetch-Site') === 'cross-site' || (write || request.headers.has('Origin')) && request.headers.get('Origin') !== config.publicOrigin || request.headers.get('X-CSRF-Token')?.includes(',') || write && !secretPattern.test(request.headers.get('X-CSRF-Token') ?? ''))
            throw new CorrectionRequestError('CSRF_FAILED'); if (request.headers.get('Idempotency-Key')?.includes(',') || write && !correctionUUID.test(request.headers.get('Idempotency-Key') ?? ''))
            throw new CorrectionRequestError('INVALID_REQUEST'); const asset = route.action === 'readOwnAsset'; const headers = new Headers({ Accept: asset ? 'image/svg+xml' : 'application/json' }), cookie = selectAuthCookies(request.headers.get('Cookie') ?? '', config.production, true); if (cookie)
            headers.set('Cookie', cookie); for (const name of ['Origin', 'X-CSRF-Token', 'Sec-Fetch-Site']) {
            const v = request.headers.get(name);
            if (v !== null)
                headers.set(name, v);
        } let body: ArrayBuffer | undefined; if (write) {
            if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(request.headers.get('Content-Type') ?? ''))
                throw new CorrectionRequestError('INVALID_REQUEST');
            let raw;
            try {
                raw = await readCorrectionBytes(new Response(request.body, { headers: request.headers.has('Content-Length') ? { 'Content-Length': request.headers.get('Content-Length')! } : {} }), 65536, active);
            }
            catch {
                throw new CorrectionRequestError(active.aborted ? 'SERVICE_UNAVAILABLE' : 'INVALID_REQUEST');
            }
            validate(raw, route.action);
            const copy = new Uint8Array(raw.byteLength);
            copy.set(raw);
            body = copy.buffer;
            headers.set('Content-Type', 'application/json');
            headers.set('Idempotency-Key', request.headers.get('Idempotency-Key')!);
        }
        else if (request.body !== null || request.headers.has('Transfer-Encoding') || Number(request.headers.get('Content-Length') ?? '0') !== 0)
            throw new CorrectionRequestError('INVALID_REQUEST'); const response = await correctionAwait(fetch(origin + path, { method: route.method, cache: 'no-store', redirect: 'error', signal: active, headers, ...(body === undefined ? {} : { body }) }), active); if(asset){
            if(response.status>=400){await readPrivateCorrectionJSON(response,active);throw new CorrectionRequestError()}
            const rid=response.headers.get('X-Request-ID'),sha=url.pathname.split('/').at(-1)!;
            if(response.status!==200||response.redirected||response.headers.has('Set-Cookie')||response.headers.get('Cache-Control')!=='private, no-store'||response.headers.get('X-Content-Type-Options')!=='nosniff'||response.headers.get('Content-Security-Policy')!=="sandbox; default-src 'none'"||!rid||!/^(?:[0-9a-f]{32}|unavailable)$/.test(rid)||!/^image\/svg\+xml(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(response.headers.get('Content-Type')??''))throw new CorrectionRequestError();
            const bytes=await readCorrectionBytes(response,1048576,active);if(!validateContentSVG(bytes,sha)||active.aborted)throw new CorrectionRequestError();
            const out=correctionPrivateHeaders(rid);out.set('Content-Type','image/svg+xml');out.set('Content-Security-Policy',"sandbox; default-src 'none'");out.set('Content-Length',String(bytes.byteLength));const copy=new Uint8Array(bytes.byteLength);copy.set(bytes);return new Response(copy.buffer,{status:200,headers:out});
        } const result = await correctionAwait(read(response, path, active, route.method), active); return Response.json(result, { status: response.status, headers: correctionPrivateHeaders(response.headers.get('X-Request-ID')!) }); });
    }
    catch (e) {
        return correctionProxyError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
    }
}
export function proxyCorrection(request: Request, segments: readonly string[] | Promise<readonly string[]>) { return proxyPrivateCorrection(request, segments, '/api/v1/corrections', resolveCorrectionRoute, validateCorrectionBytes, readCorrectionResponse); }
