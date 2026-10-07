import { z } from 'zod';
import { parseContentJSON, validContentString } from '../content/raw-json';
import { readContentBytes } from '../content/bytes';
import { answerSchema, identitySchema, instanceIdentitySchema, learningUUID } from '../learning/schemas';
import { CorrectionRequestError, correctionPolicies, type CorrectionRoute, type CaseInput, type CaseMetadata, type PlanInput, type PlanMetadata, type ResultMetadata, type ResultDetail, type Receipt, type PageQuery } from './types';
export const correctionUUID = learningUUID;
export const uuidSchema = z.string().regex(correctionUUID), versionSchema = z.number().int().min(1).max(2147483647), sequenceSchema = z.number().int().min(1).max(Number.MAX_SAFE_INTEGER), timeSchema = z.iso.datetime({ offset: true });
const text = z.string().refine(validContentString), scalar = text.refine(v => [...v].length >= 1 && [...v].length <= 4000 && v.replace(/^\p{White_Space}+|\p{White_Space}+$/gu, '') !== '');
const sha = z.string().regex(/^[0-9a-f]{64}$/), mathId = text.max(64).regex(/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/);
export const planRefSchema = z.object({ id: uuidSchema, version: versionSchema }).strict();
const withdrawal = z.object({ space: z.enum(['content', 'question']), id: uuidSchema }).strict();
const rule = z.discriminatedUnion('kind', [z.object({ ruleVersion: versionSchema, kind: z.literal('all'), knowledge: z.null() }).strict(), z.object({ ruleVersion: versionSchema, kind: z.literal('knowledge'), knowledge: identitySchema }).strict()]);
export const caseInputSchema = z.discriminatedUnion('kind', [z.object({ kind: z.literal('withdrawal'), withdrawal, rule: z.null() }).strict(), z.object({ kind: z.literal('grading_rule'), withdrawal: z.null(), rule }).strict()]) satisfies z.ZodType<CaseInput>;
const caseFields = { id: uuidSchema, sequence: z.literal(1), createdAt: timeSchema, hasApprovedPlan: z.boolean() };
export const caseMetadataSchema = z.discriminatedUnion('kind', [z.object({ ...caseFields, kind: z.literal('withdrawal'), withdrawal, rule: z.null(), cutoff: z.null() }).strict(), z.object({ ...caseFields, kind: z.literal('grading_rule'), withdrawal: z.null(), rule, cutoff: timeSchema }).strict()]) satisfies z.ZodType<CaseMetadata>;
const published = z.object({ identity: instanceIdentitySchema, publicationId: uuidSchema }).strict();
export const mappingSchema = z.object({ original: instanceIdentitySchema, originalPublicationId: uuidSchema, replacement: published }).strict();
const mappings = z.array(mappingSchema).max(50).refine(v => new Set(v.map(m => JSON.stringify(m.original))).size === v.length);
export const planInputSchema = z.object({ expectedSequence: sequenceSchema.nullable(), parent: planRefSchema.nullable(), algorithmVersion: z.literal(1), mappings, reason: scalar }).strict() satisfies z.ZodType<PlanInput>;
const createPlanSchema = planInputSchema.refine(v => v.expectedSequence === null), updatePlanSchema = planInputSchema.refine(v => v.expectedSequence !== null);
export const submitInputSchema = z.object({ expectedSequence: sequenceSchema }).strict();
export const decisionInputSchema = submitInputSchema.extend({ decision: z.enum(['approve', 'reject']), reason: scalar }).strict();
export const correctionInputSchemas = { createCase: caseInputSchema, createPlan: createPlanSchema, updatePlan: updatePlanSchema, submitPlan: submitInputSchema, decidePlan: decisionInputSchema, retryJob: submitInputSchema };
export const planMetadataSchema = z.object({ ref: planRefSchema, caseId: uuidSchema, status: z.enum(['draft', 'pending', 'approved', 'rejected']), sequence: sequenceSchema, algorithmVersion: z.literal(1), mappingCount: z.number().int().min(0).max(50), createdAt: timeSchema, updatedAt: timeSchema, digest: sha.nullable() }).strict().refine(v => Date.parse(v.updatedAt) >= Date.parse(v.createdAt) && (v.status === 'draft' ? v.digest === null : v.digest !== null)) satisfies z.ZodType<PlanMetadata>;
const planFields = { plan: planMetadataSchema, parent: planRefSchema.nullable() };
export const planMetadataViewSchema = z.object({ ...planFields, mappings: z.array(mappingSchema).length(0), reason: z.null(), decisionReason: z.null() }).strict();
export const planDetailSchema = z.object({ ...planFields, mappings, reason: scalar, decisionReason: scalar.nullable() }).strict().refine(v => v.mappings.length === v.plan.mappingCount);
export const evidenceRefSchema = z.object({ kind: z.enum(['learning-event', 'practice', 'assessment', 'enrollment']), id: uuidSchema }).strict();
const reasons = z.enum(['answer_corrected', 'rule_regraded', 'source_invalid', 'intent_changed', 'coverage_changed', 'knowledge_changed', 'insufficient_items', 'no_approved_basis', 'conflicting_basis', 'path_withdrawn', 'unaffected']);
const resultFields = { id: uuidSchema, caseId: uuidSchema, plan: planRefSchema.nullable(), parentResultId: uuidSchema.nullable(), knowledge: identitySchema.nullable(), reason: reasons, validity: z.enum(['effective', 'restricted']), handledCaseIds: z.array(uuidSchema).refine(v => new Set(v).size === v.length), createdAt: timeSchema };
const assessmentEvidence = z.object({ kind: z.literal('assessment'), id: uuidSchema }).strict(), practiceEvidence = z.object({ kind: z.literal('practice'), id: uuidSchema }).strict();
export const resultMetadataSchema = z.union([
    z.object({ ...resultFields, plan: planRefSchema, knowledge: identitySchema, status: z.literal('corrected_passed'), evidence: assessmentEvidence, score: z.number().int().min(4).max(5), passed: z.literal(true) }).strict(),
    z.object({ ...resultFields, plan: planRefSchema, knowledge: identitySchema, status: z.literal('corrected_passed'), evidence: practiceEvidence, score: z.null(), passed: z.null() }).strict(),
    z.object({ ...resultFields, plan: planRefSchema, knowledge: identitySchema, status: z.literal('corrected_failed'), evidence: assessmentEvidence, score: z.number().int().min(0).max(3), passed: z.literal(false) }).strict(),
    z.object({ ...resultFields, plan: planRefSchema, knowledge: identitySchema, status: z.literal('corrected_failed'), evidence: practiceEvidence, score: z.null(), passed: z.null() }).strict(),
    z.object({ ...resultFields, status: z.literal('retake_required'), evidence: evidenceRefSchema, score: z.null(), passed: z.null() }).strict(),
    z.object({ ...resultFields, status: z.literal('review_material'), evidence: evidenceRefSchema, score: z.null(), passed: z.null() }).strict(),
    z.object({ ...resultFields, status: z.literal('checked_unaffected'), evidence: evidenceRefSchema, score: z.null(), passed: z.null() }).strict(),
    z.object({ ...resultFields, status: z.literal('awaiting_review'), evidence: evidenceRefSchema, score: z.null(), passed: z.null() }).strict()
]) satisfies z.ZodType<ResultMetadata>;
export const correctedItemSchema = z.object({ position: z.number().int().min(1).max(5), original: instanceIdentitySchema, effective: instanceIdentitySchema, originalTemplate: identitySchema.nullable(), effectiveTemplate: identitySchema.nullable(), prompt: text, choices: z.array(z.object({ id: mathId, text }).strict()).max(12), answer: answerSchema, correct: z.boolean(), explanation: text, assets: z.array(z.object({ id: mathId, sha256: sha }).strict()).max(50) }).strict();
export const resultMetadataViewSchema = z.object({ result: resultMetadataSchema, items: z.array(correctedItemSchema).length(0), planReason: z.null() }).strict();
export const resultDetailSchema = z.object({ result: resultMetadataSchema, items: z.array(correctedItemSchema).max(5), planReason: scalar.nullable() }).strict().refine(v => {
    const corrected = v.result.status === 'corrected_passed' || v.result.status === 'corrected_failed';
    const n = corrected ? (v.result.evidence.kind === 'assessment' ? 5 : 1) : 0;
    return v.items.length === n && v.items.every((item, i) => item.position === i + 1) && (!corrected || v.result.evidence.kind !== 'assessment' || v.items.filter(item => item.correct).length === v.result.score);
}) satisfies z.ZodType<ResultDetail>;
export const jobMetadataSchema = z.object({ id: uuidSchema, caseId: uuidSchema, plan: planRefSchema.nullable(), type: z.enum(['withdrawal_impact', 'rule_impact', 'approved_plan', 'attempt_terminal']), state: z.enum(['queued', 'running', 'succeeded', 'retry_wait', 'failed']), sequence: sequenceSchema, epoch: versionSchema, attempt: z.number().int().min(0).max(8), nextRunAt: timeSchema.nullable(), processedCount: z.number().int().min(0).max(Number.MAX_SAFE_INTEGER), errorClass: z.enum(['database', 'deadline', 'source', 'configuration', 'lease', 'internal']).nullable() }).strict();
const receiptFields = { status: z.union([z.literal(200), z.literal(201)]) };
export const receiptSchema = z.union([z.object({ ...receiptFields, case: caseMetadataSchema, plan: z.null(), job: z.null() }).strict(), z.object({ ...receiptFields, case: z.null(), plan: planMetadataSchema, job: z.null() }).strict(), z.object({ ...receiptFields, case: z.null(), plan: z.null(), job: jobMetadataSchema }).strict()]) satisfies z.ZodType<Receipt>;
// A cursor is the exact Go JSON form, including nanoseconds and key order.
const cursorPayload = z.object({ version: z.literal(1), createdAt: timeSchema, id: uuidSchema }).strict();
export function validCorrectionCursor(raw: string): boolean {
    try {
        if (raw.length === 0 || raw.length > 512 || !/^[A-Za-z0-9_-]+$/.test(raw))
            return false;
        const bytes = Uint8Array.from(atob(raw.replaceAll('-', '+').replaceAll('_', '/')), v => v.charCodeAt(0));
        const c = cursorPayload.parse(parseContentJSON(bytes));
        const m = /^(.*?)(?:\.(\d+))?(Z|[+-]\d{2}:\d{2})$/.exec(c.createdAt);
        if (!m || m[2] && m[2].length > 9)
            return false;
        const frac = (m[2] ?? '').replace(/0+$/, '');
        const canonicalTime = m[1] + (frac ? '.' + frac : '') + ((m[3] === '+00:00' || m[3] === '-00:00') ? 'Z' : m[3]);
        const again = btoa(JSON.stringify({ version: 1, createdAt: canonicalTime, id: c.id })).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
        return again === raw;
    }
    catch {
        return false;
    }
}
export const cursorSchema = z.string().refine(validCorrectionCursor).nullable();
export const pageSchema = <T extends z.ZodType>(item: T) => z.object({ items: z.array(item).max(50), nextCursor: cursorSchema }).strict();
export const casePageSchema = pageSchema(caseMetadataSchema), planPageSchema = pageSchema(planMetadataSchema), jobPageSchema = pageSchema(jobMetadataSchema), resultPageSchema = pageSchema(resultMetadataSchema);
export const envelopeSchema = <T extends z.ZodType>(data: T) => z.object({ actorId: uuidSchema, data }).strict();
export function queryPath(path: string, query: PageQuery = {}): string { const q = z.object({ limit: z.number().int().min(1).max(50).optional(), cursor: z.string().refine(validCorrectionCursor).optional() }).strict().safeParse(query); if (!q.success)
    throw new CorrectionRequestError('INVALID_REQUEST'); const params = new URLSearchParams(); if (q.data.limit !== undefined)
    params.set('limit', String(q.data.limit)); if (q.data.cursor !== undefined)
    params.set('cursor', q.data.cursor); return path + (params.size ? '?' + params : ''); }
export function strictPrivatePath(path: string, prefix: string): {
    parts: string[];
    query: URLSearchParams;
} {
    const bad = (): never => { throw new CorrectionRequestError('INVALID_REQUEST'); };
    if (!path.startsWith(prefix) || /[#\\%]/.test(path.split('?')[0]) || path.includes('#') || path.endsWith('?') || path.includes('\\'))
        return bad();
    const [rawPath, rawQuery, ...extra] = path.split('?');
    if (extra.length || rawPath.split('/').some(p => p === '.' || p === '..'))
        return bad();
    const parts = rawPath.slice(prefix.length).split('/').filter((v, i, all) => !(v === '' && all.length === 1));
    if (parts.some(p => !p))
        return bad();
    const query = new URLSearchParams(rawQuery);
    const seen = new Set<string>();
    for (const entry of (rawQuery === undefined ? [] : rawQuery.split('&'))) {
        const pair = entry.split('=');
        if (pair.length !== 2 || !pair[0] || !pair[1] || /[%+]/.test(pair[0]) || /[%+]/.test(pair[1]) || seen.has(pair[0]))
            return bad();
        seen.add(pair[0]);
    }
    return { parts, query };
}
export function validatePageQuery(query: URLSearchParams, list: boolean, evidence = false): void {
    for (const [key, value] of query) {
        if (!value)
            throw new CorrectionRequestError('INVALID_REQUEST');
        switch (key) {
            case 'limit':
                if (!list || !/^[1-9][0-9]*$/.test(value) || Number(value) > 50)
                    throw new CorrectionRequestError('INVALID_REQUEST');
                break;
            case 'cursor':
                if (!list || !validCorrectionCursor(value))
                    throw new CorrectionRequestError('INVALID_REQUEST');
                break;
            case 'kind':
                if (!evidence || !['learning-event', 'practice', 'assessment', 'enrollment'].includes(value))
                    throw new CorrectionRequestError('INVALID_REQUEST');
                break;
            case 'id':
                if (!evidence || !correctionUUID.test(value))
                    throw new CorrectionRequestError('INVALID_REQUEST');
                break;
            default: throw new CorrectionRequestError('INVALID_REQUEST');
        }
    }
    if (evidence && (!query.has('kind') || !query.has('id')))
        throw new CorrectionRequestError('INVALID_REQUEST');
}
export function resolveCorrectionRoute(path: string, method = 'GET'): CorrectionRoute {
    const { parts: p, query } = strictPrivatePath(path, '/api/v1/corrections/');
    let action: CorrectionRoute['action'], wanted: CorrectionRoute['method'] = 'GET';
    const bad = (): never => { throw new CorrectionRequestError('INVALID_REQUEST'); };
    if (p[0] === 'cases' && p.length === 1) {
        action = method === 'POST' ? 'createCase' : 'listCases';
        wanted = method === 'POST' ? 'POST' : 'GET';
    }
    else if (p[0] === 'cases' && correctionUUID.test(p[1] ?? '') && p.length === 2)
        action = 'readCase';
    else if (p[0] === 'cases' && correctionUUID.test(p[1] ?? '') && p.length === 3 && p[2] === 'plans') {
        action = method === 'POST' ? 'createPlan' : 'listPlans';
        wanted = method === 'POST' ? 'POST' : 'GET';
    }
    else if (p[0] === 'cases' && correctionUUID.test(p[1] ?? '') && p.length === 3 && p[2] === 'jobs')
        action = 'listJobs';
    else if (p[0] === 'plans' && correctionUUID.test(p[1] ?? '') && p[2] === 'versions' && /^[1-9][0-9]*$/.test(p[3] ?? '') && Number(p[3]) <= 2147483647) {
        if (p.length === 4) {
            action = method === 'PUT' ? 'updatePlan' : 'readPlan';
            wanted = method === 'PUT' ? 'PUT' : 'GET';
        }
        else if (p.length === 5 && p[4] === 'detail')
            action = 'readPlanDetail';
        else if (p.length === 5 && p[4] === 'submit') {
            action = 'submitPlan';
            wanted = 'POST';
        }
        else if (p.length === 5 && p[4] === 'decision') {
            action = 'decidePlan';
            wanted = 'POST';
        }
        else
            return bad();
    }
    else if (p[0] === 'jobs' && correctionUUID.test(p[1] ?? '') && p.length === 3 && p[2] === 'retry') {
        action = 'retryJob';
        wanted = 'POST';
    }
    else if (p[0] === 'evidence' && p.length === 1)
        action = 'listOwn';
    else if (p[0] === 'results' && correctionUUID.test(p[1] ?? '') && p.length === 4 && p[2] === 'assets' && /^[0-9a-f]{64}$/.test(p[3] ?? ''))
        action = 'readOwnAsset';
    else if (p[0] === 'results' && correctionUUID.test(p[1] ?? '') && p.length === 2)
        action = 'readOwn';
    else if (p[0] === 'results' && correctionUUID.test(p[1] ?? '') && p.length === 3 && p[2] === 'detail')
        action = 'readOwnDetail';
    else
        return bad();
    if (method !== wanted)
        throw new CorrectionRequestError('METHOD_NOT_ALLOWED');
    validatePageQuery(query, ['listCases', 'listPlans', 'listJobs', 'listOwn'].includes(action), action === 'listOwn');
    return { path, method: wanted, action };
}
function responseSchema(action: CorrectionRoute['action']) { switch (action) {
    case 'createCase':
    case 'createPlan':
    case 'updatePlan':
    case 'submitPlan':
    case 'decidePlan':
    case 'retryJob': return receiptSchema;
    case 'listCases': return casePageSchema;
    case 'readCase': return caseMetadataSchema;
    case 'listPlans': return planPageSchema;
    case 'readPlan': return planMetadataViewSchema;
    case 'readPlanDetail': return planDetailSchema;
    case 'listJobs': return jobPageSchema;
    case 'listOwn': return resultPageSchema;
    case 'readOwn': return resultMetadataViewSchema;
    case 'readOwnDetail': return resultDetailSchema;
    case 'readOwnAsset': throw new CorrectionRequestError('INVALID_REQUEST');
} }
const errorCodes = ['MODULE_RETIRED','INVALID_REQUEST', 'INVALID_COOKIE', 'AUTHENTICATION_REQUIRED', 'CSRF_FAILED', 'FORBIDDEN', 'NOT_FOUND', 'METHOD_NOT_ALLOWED', 'PASSWORD_CHANGE_REQUIRED', 'RATE_LIMITED', 'IDEMPOTENCY_CONFLICT', 'SERVICE_UNAVAILABLE', 'REAUTHENTICATION_REQUIRED', 'CORRECTION_NOT_CONFIGURED', 'CORRECTION_CONFLICT', 'CORRECTION_SOURCE_STALE', 'CORRECTION_ANSWER_OVERLAP', 'CORRECTION_LEASE_LOST'] as const;
const errorSchema = z.object({ error: z.object({ code: z.enum(errorCodes), message: text, requestId: z.string().regex(/^(?:[0-9a-f]{32}|unavailable)$/), retryAt: timeSchema.optional() }).strict() }).strict();
export async function readPrivateCorrectionJSON(response: Response, signal: AbortSignal): Promise<unknown> {
    try {
        const rid = response.headers.get('X-Request-ID');
        if (response.redirected || response.headers.has('Set-Cookie') || response.headers.get('Cache-Control') !== 'private, no-store' || response.headers.get('X-Content-Type-Options') !== 'nosniff' || !rid || !/^(?:[0-9a-f]{32}|unavailable)$/.test(rid) || !/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(response.headers.get('Content-Type') ?? ''))
            throw new Error();
        const value = parseContentJSON(await readContentBytes(response, 2097152, signal));
        if (signal.aborted)
            throw new Error();
        if (response.status >= 400) {
            const e = errorSchema.parse(value).error;
            if (e.requestId !== rid || correctionPolicies[e.code][0] !== response.status || e.code !== 'RATE_LIMITED' && e.retryAt !== undefined || e.code === 'RATE_LIMITED' && (!e.retryAt || !/^[1-9][0-9]*$/.test(response.headers.get('Retry-After') ?? '')))
                throw new Error();
            throw new CorrectionRequestError(e.code, rid, e.retryAt);
        }
        return value;
    }
    catch (e) {
        throw e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
    }
}
export async function readCorrectionResponse(response: Response, path: string, signal: AbortSignal, method = 'GET') {
    try {
        const route = resolveCorrectionRoute(path, method);
        const value = await readPrivateCorrectionJSON(response, signal);
        const out = envelopeSchema(responseSchema(route.action)).parse(value);
        const expected = ['createCase', 'createPlan'].includes(route.action) ? 201 : 200;
        if (response.status !== expected || 'status' in out.data && out.data.status !== response.status)
            throw new Error();
        if (route.action === 'createCase' && (!('case' in out.data) || out.data.case === null) || ['createPlan', 'updatePlan', 'submitPlan', 'decidePlan'].includes(route.action) && (!('plan' in out.data) || out.data.plan === null) || route.action === 'retryJob' && (!('job' in out.data) || out.data.job === null))
            throw new Error();
        return out;
    }
    catch (e) {
        throw e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
    }
}
