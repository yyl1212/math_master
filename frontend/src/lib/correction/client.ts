import { z } from 'zod';
import { getAuthContext, notifyAuthChanged } from '../auth/client';
import { correctionAwait, withCorrectionDeadline, validateCorrectionBytes } from './bytes';
import { CorrectionRequestError, type CommandAccess, type ReadAccess, type PageQuery, type CaseInput, type PlanInput, type PlanRef, type SubmitInput, type DecisionInput, type RetryInput, type EvidenceRef } from './types';
import { uuidSchema, planRefSchema, envelopeSchema, casePageSchema, caseMetadataSchema, planPageSchema, planMetadataViewSchema, planDetailSchema, jobPageSchema, resultPageSchema, resultMetadataViewSchema, resultDetailSchema, receiptSchema, resolveCorrectionRoute, readCorrectionResponse, queryPath } from './schemas';
export async function currentCorrectionActor(access: ReadAccess, signal: AbortSignal) { if (access.actorId !== undefined && !uuidSchema.safeParse(access.actorId).success)
    throw new CorrectionRequestError('INVALID_REQUEST'); const expectedActor = access.actorId; const context = await correctionAwait(getAuthContext(true), signal); if (!context.ok)
    throw new CorrectionRequestError(context.code === 'AUTHENTICATION_REQUIRED' ? 'AUTHENTICATION_REQUIRED' : context.code === 'PASSWORD_CHANGE_REQUIRED' ? 'PASSWORD_CHANGE_REQUIRED' : 'SERVICE_UNAVAILABLE'); const user = context.data.user; if (!user || expectedActor !== undefined && user.id !== expectedActor) {
    notifyAuthChanged();
    throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
} if (user.mustChangePassword) {
    notifyAuthChanged();
    throw new CorrectionRequestError('PASSWORD_CHANGE_REQUIRED');
} return { actorId: user.id, csrfToken: context.data.csrfToken }; }
export function assertCorrectionActor(actual: string, expected: string): void { if (actual !== expected) {
    notifyAuthChanged();
    throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
} }
function checkedId(id: string) { if (!uuidSchema.safeParse(id).success)
    throw new CorrectionRequestError('INVALID_REQUEST'); return id; }
function planPath(ref: PlanRef) { const r = planRefSchema.safeParse(ref); if (!r.success)
    throw new CorrectionRequestError('INVALID_REQUEST'); return '/api/v1/corrections/plans/' + r.data.id + '/versions/' + r.data.version; }
function read<T>(path: string, schema: z.ZodType<T>, access: ReadAccess = {}) { return withCorrectionDeadline(access.signal, async (active) => { resolveCorrectionRoute(path); const current = await currentCorrectionActor(access, active); const response = await correctionAwait(fetch(path, { method: 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: { Accept: 'application/json' }, signal: active }), active); const out = envelopeSchema(schema).parse(await correctionAwait(readCorrectionResponse(response, path, active), active)); assertCorrectionActor(out.actorId, current.actorId); return out; }); }
function command(path: string, method: 'POST' | 'PUT', input: unknown, access: CommandAccess) { return withCorrectionDeadline(access.signal, async (active) => { if (!uuidSchema.safeParse(access.actorId).success || !uuidSchema.safeParse(access.key).success)
    throw new CorrectionRequestError('INVALID_REQUEST'); const route = resolveCorrectionRoute(path, method), body = JSON.stringify(input), actorId = access.actorId, key = access.key; validateCorrectionBytes(new TextEncoder().encode(body), route.action); const current = await currentCorrectionActor({ actorId }, active); const response = await correctionAwait(fetch(path, { method, credentials: 'same-origin', cache: 'no-store', redirect: 'error', signal: active, headers: { Accept: 'application/json', 'Content-Type': 'application/json', 'X-CSRF-Token': current.csrfToken, 'Idempotency-Key': key }, body }), active); const out = envelopeSchema(receiptSchema).parse(await correctionAwait(readCorrectionResponse(response, path, active, method), active)); assertCorrectionActor(out.actorId, actorId); return out; }); }
export const correctionClient = {
    listCases: (query: PageQuery = {}, access: ReadAccess = {}) => read(queryPath('/api/v1/corrections/cases', query), casePageSchema, access),
    readCase: (id: string, access: ReadAccess = {}) => read('/api/v1/corrections/cases/' + checkedId(id), caseMetadataSchema, access),
    listPlans: (id: string, query: PageQuery = {}, access: ReadAccess = {}) => read(queryPath('/api/v1/corrections/cases/' + checkedId(id) + '/plans', query), planPageSchema, access),
    readPlan: (ref: PlanRef, access: ReadAccess = {}) => read(planPath(ref), planMetadataViewSchema, access),
    readPlanDetail: (ref: PlanRef, access: ReadAccess = {}) => read(planPath(ref) + '/detail', planDetailSchema, access),
    listJobs: (id: string, query: PageQuery = {}, access: ReadAccess = {}) => read(queryPath('/api/v1/corrections/cases/' + checkedId(id) + '/jobs', query), jobPageSchema, access),
    listOwn: (query: PageQuery & EvidenceRef, access: ReadAccess = {}) => { const { kind, id, ...page } = query; let path = queryPath('/api/v1/corrections/evidence', page); if (kind !== undefined || id !== undefined) {
        if (kind === undefined || id === undefined)
            throw new CorrectionRequestError('INVALID_REQUEST');
        path += (path.includes('?') ? '&' : '?') + new URLSearchParams({ kind, id });
    } return read(path, resultPageSchema, access); },
    readOwn: (id: string, access: ReadAccess = {}) => read('/api/v1/corrections/results/' + checkedId(id), resultMetadataViewSchema, access),
    readOwnDetail: (id: string, access: ReadAccess = {}) => read('/api/v1/corrections/results/' + checkedId(id) + '/detail', resultDetailSchema, access),
    createCase: (input: CaseInput, access: CommandAccess) => command('/api/v1/corrections/cases', 'POST', input, access),
    createPlan: (id: string, input: PlanInput, access: CommandAccess) => command('/api/v1/corrections/cases/' + checkedId(id) + '/plans', 'POST', input, access),
    updatePlan: (ref: PlanRef, input: PlanInput, access: CommandAccess) => command(planPath(ref), 'PUT', input, access),
    submitPlan: (ref: PlanRef, input: SubmitInput, access: CommandAccess) => command(planPath(ref) + '/submit', 'POST', input, access),
    decidePlan: (ref: PlanRef, input: DecisionInput, access: CommandAccess) => command(planPath(ref) + '/decision', 'POST', input, access),
    retryJob: (id: string, input: RetryInput, access: CommandAccess) => command('/api/v1/corrections/jobs/' + checkedId(id) + '/retry', 'POST', input, access)
};
