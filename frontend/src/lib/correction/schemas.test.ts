import { it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { z } from 'zod';
import { validateCorrectionBytes } from './bytes';
import { resolveCorrectionRoute, caseMetadataSchema, planMetadataViewSchema, planDetailSchema, resultMetadataViewSchema, resultDetailSchema, readCorrectionResponse } from './schemas';
import { id, caseMetadata, planMetadata, resultMetadata, resultDetail, json, cursor } from './test-fixtures';
it('all shared Go correction raw boundary cases agree', () => { const corpus = z.object({ version: z.literal(1), requests: z.array(z.object({ name: z.string(), domain: z.string(), action: z.string(), raw: z.string(), valid: z.boolean() })) }).parse(JSON.parse(readFileSync('../api/correction-boundary-cases.json', 'utf8'))); for (const c of corpus.requests.filter(c => c.domain === 'correction')) {
    let accepted = true;
    try {
        validateCorrectionBytes(new TextEncoder().encode(c.raw), c.action);
    }
    catch {
        accepted = false;
    }
    expect(accepted, c.name).toBe(c.valid);
} });
it('requires every nullable key and rejects free or private metadata', () => { const cm = caseMetadata(); expect(caseMetadataSchema.safeParse(cm).success).toBe(true); for (const key of ['withdrawal', 'cutoff', 'rule']) {
    const v: Record<string, unknown> = { ...cm };
    delete v[key];
    expect(caseMetadataSchema.safeParse(v).success, key).toBe(false);
} const p = { plan: planMetadata(), parent: null, mappings: [], reason: null, decisionReason: null }; expect(planMetadataViewSchema.safeParse(p).success).toBe(true); expect(planMetadataViewSchema.safeParse({ ...p, reason: 'secret' }).success).toBe(false); expect(planDetailSchema.safeParse({ ...p, reason: 'Approved reason.' }).success).toBe(true); const r = { result: resultMetadata(), items: [], planReason: null }; expect(resultMetadataViewSchema.safeParse(r).success).toBe(true); expect(resultMetadataViewSchema.safeParse({ ...r, items: resultDetail().items }).success).toBe(false); expect(caseMetadataSchema.safeParse({ ...cm, actorId: id }).success).toBe(false); });
it('validates corrected scores, complete positions, and retake redaction', () => { const d = resultDetail(); expect(resultDetailSchema.safeParse(d).success).toBe(true); expect(resultDetailSchema.safeParse({ ...d, result: { ...d.result, score: 3 } }).success).toBe(false); expect(resultDetailSchema.safeParse({ ...d, items: d.items.slice(0, 4) }).success).toBe(false); expect(resultDetailSchema.safeParse({ ...d, items: d.items.map(v => ({ ...v, position: 1 })) }).success).toBe(false); const retake = { ...d, result: { ...d.result, status: 'retake_required', score: null, passed: null } }; expect(resultDetailSchema.safeParse(retake).success).toBe(false); expect(resultDetailSchema.safeParse({ ...retake, items: [] }).success).toBe(true); });
it('rejects invalid versions, cursor encodings, duplicate and encoded queries', () => { expect(resolveCorrectionRoute('/api/v1/corrections/cases?limit=20&cursor=' + cursor()).action).toBe('listCases'); for (const path of ['/api/v1/corrections/cases?limit=0', '/api/v1/corrections/cases?limit=51', '/api/v1/corrections/cases?limit=1&limit=2', '/api/v1/corrections/cases?%6cimit=2', '/api/v1/corrections/cases?cursor=e30', '/api/v1/corrections/cases?cursor=' + cursor() + '=', '/api/v1/corrections/cases?unknown=x', '/api/v1/corrections/plans/' + id + '/versions/01', '/api/v1/corrections/plans/' + id + '/versions/2147483648', '/api/v1/corrections/cases/' + id + '?limit=1', '/api/v1/corrections/cases/../cases', '/api/v1/corrections/cases#x'])
    expect(() => resolveCorrectionRoute(path), path).toThrow(); });
it('bounds complete bytes and validates error headers before displaying static errors', async () => { const signal = new AbortController().signal; for (const r of [json(caseMetadata(), 200, id, { 'Content-Length': '2097153' }), json({ ...caseMetadata(), reason: 'private' }), json(caseMetadata(), 200, id, { 'Set-Cookie': 'private' }), json(caseMetadata(), 200, id, { 'Content-Length': '1' })])
    await expect(readCorrectionResponse(r, '/api/v1/corrections/cases/' + id, signal)).rejects.toMatchObject({ code: 'SERVICE_UNAVAILABLE' }); const response = Response.json({ error: { code: 'CORRECTION_CONFLICT', message: 'private sentinel', requestId: 'a'.repeat(32) } }, { status: 409, headers: { 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff', 'X-Request-ID': 'a'.repeat(32) } }); await expect(readCorrectionResponse(response, '/api/v1/corrections/cases/' + id, signal)).rejects.toMatchObject({ code: 'CORRECTION_CONFLICT', message: 'This correction changed. Reload before continuing.' }); });
it('uses the existing authentication error codes and statuses for recovery', async () => { for (const [code, status] of [['REAUTHENTICATION_REQUIRED', 428], ['PASSWORD_CHANGE_REQUIRED', 403]] as const) {
    const response = Response.json({ error: { code, message: 'private source text', requestId: 'a'.repeat(32) } }, { status, headers: { 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff', 'X-Request-ID': 'a'.repeat(32) } });
    await expect(readCorrectionResponse(response, '/api/v1/corrections/cases/' + id, new AbortController().signal)).rejects.toMatchObject({ code, status });
} });
it('corrected outcomes require an approved plan and accurate knowledge identity', () => { const d = resultDetail(); for (const field of ['plan', 'knowledge'])
    expect(resultDetailSchema.safeParse({ ...d, result: { ...d.result, [field]: null } }).success).toBe(false); });
it('matches Go required evidence filters and exact raw query pairs', () => { for (const suffix of ['evidence', 'evidence?kind=assessment', 'cases?limit=1&', 'cases?limit=1&&cursor=e30', 'cases?cursor=%'])
    expect(() => resolveCorrectionRoute('/api/v1/corrections/' + suffix)).toThrow(); });
it('measures actual response bytes without trusting Content-Length', async () => { const raw = JSON.stringify({ actorId: id, data: caseMetadata() }), headers = { 'Content-Type': 'application/json', 'Cache-Control': 'private, no-store', 'X-Content-Type-Options': 'nosniff', 'X-Request-ID': 'a'.repeat(32) }; const n = new TextEncoder().encode(raw).byteLength; for (const size of [2097152, 2097153]) {
    const pending = readCorrectionResponse(new Response(raw + ' '.repeat(size - n), { headers }), '/api/v1/corrections/cases/' + id, new AbortController().signal);
    if (size === 2097152)
        expect((await pending).data).toEqual(caseMetadata());
    else
        await expect(pending).rejects.toMatchObject({ code: 'SERVICE_UNAVAILABLE' });
} });
