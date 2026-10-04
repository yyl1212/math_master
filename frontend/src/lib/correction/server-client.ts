import 'server-only';
import { z } from 'zod';
import { cookies } from 'next/headers';
import { getGoOrigin } from '../api/server-config';
import { getAuthConfig, AuthNotConfiguredError } from '../auth/config';
import { selectAuthCookies } from '../auth/cookies';
import { correctionAwait, withCorrectionDeadline } from './bytes';
import { CorrectionRequestError, type PlanRef, type PageQuery, type EvidenceRef, type ReadAccess } from './types';
import { uuidSchema, planRefSchema, envelopeSchema, casePageSchema, caseMetadataSchema, planPageSchema, planMetadataViewSchema, planDetailSchema, jobPageSchema, resultPageSchema, resultMetadataViewSchema, resultDetailSchema, resolveCorrectionRoute, readCorrectionResponse, queryPath } from './schemas';
export function correctionServerRead<T>(path: string, schema: z.ZodType<T>, access: ReadAccess = {}, reader: (response: Response, path: string, signal: AbortSignal) => Promise<unknown> = readCorrectionResponse) { return withCorrectionDeadline(access.signal, async (active) => { const expectedActor = access.actorId; let config; try {
    config = getAuthConfig();
}
catch (e) {
    if (e instanceof AuthNotConfiguredError)
        throw new CorrectionRequestError('CORRECTION_NOT_CONFIGURED');
    throw e;
} const origin = getGoOrigin(), jar = await correctionAwait(cookies(), active), cookie = selectAuthCookies(jar.toString(), config.production, true); const response = await correctionAwait(fetch(origin + path, { method: 'GET', cache: 'no-store', redirect: 'error', signal: active, headers: { Accept: 'application/json', ...(cookie ? { Cookie: cookie } : {}) } }), active); const out = envelopeSchema(schema).parse(await correctionAwait(reader(response, path, active), active)); if (expectedActor !== undefined && out.actorId !== expectedActor)
    throw new CorrectionRequestError('AUTHENTICATION_REQUIRED'); return out; }); }
function checkedId(id: string) { if (!uuidSchema.safeParse(id).success)
    throw new CorrectionRequestError('INVALID_REQUEST'); return id; }
function planPath(ref: PlanRef) { const r = planRefSchema.safeParse(ref); if (!r.success)
    throw new CorrectionRequestError('INVALID_REQUEST'); return '/api/v1/corrections/plans/' + r.data.id + '/versions/' + r.data.version; }
function read<T>(path: string, schema: z.ZodType<T>, access: ReadAccess = {}) { resolveCorrectionRoute(path); return correctionServerRead(path, schema, access); }
export const correctionServer = {
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
    readOwnDetail: (id: string, access: ReadAccess = {}) => read('/api/v1/corrections/results/' + checkedId(id) + '/detail', resultDetailSchema, access)
};
