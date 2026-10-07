import type { components } from '../api/generated';
type S<K extends keyof components['schemas']> = components['schemas'][K];
export type Identity = S<'LearningIdentity'>;
export type CaseInput = S<'CorrectionCaseInput'>;
export type CaseMetadata = S<'CorrectionCaseMetadata'>;
export type PlanRef = S<'CorrectionPlanRef'>;
export type Mapping = S<'CorrectionMapping'>;
export type PlanInput = S<'CorrectionPlanInput'>;
export type SubmitInput = S<'CorrectionSubmitInput'>;
export type DecisionInput = S<'CorrectionDecisionInput'>;
export type RetryInput = S<'CorrectionRetryInput'>;
export type PlanMetadata = S<'CorrectionPlanMetadata'>;
export type PlanMetadataView = S<'CorrectionPlanMetadataView'>;
export type PlanDetail = S<'CorrectionPlanDetail'>;
export type ResultMetadata = S<'CorrectionResultMetadata'>;
export type ResultMetadataView = S<'CorrectionResultMetadataView'>;
export type ResultDetail = S<'CorrectionResultDetail'>;
export type CorrectedItem = S<'CorrectionCorrectedItem'>;
export type EvidenceRef = S<'CorrectionEvidenceRef'>;
export type JobMetadata = S<'CorrectionJobMetadata'>;
export type Receipt = S<'CorrectionReceipt'>;
export type CorrectionErrorCode = S<'CorrectionError'>['error']['code'];
export type Page<T> = {
    items: T[];
    nextCursor: string | null;
};
export type Envelope<T> = {
    actorId: string;
    data: T;
};
export type PageQuery = {
    limit?: number;
    cursor?: string;
};
export type ReadAccess = {
    actorId?: string;
    signal?: AbortSignal;
};
export type CommandAccess = {
    actorId: string;
    key: string;
    signal?: AbortSignal;
};
export type CorrectionAction = 'createCase' | 'createPlan' | 'updatePlan' | 'submitPlan' | 'decidePlan' | 'retryJob' | 'listCases' | 'readCase' | 'listPlans' | 'readPlan' | 'readPlanDetail' | 'listJobs' | 'listOwn' | 'readOwn' | 'readOwnDetail' | 'readOwnAsset';
export type CorrectionRoute = {
    path: string;
    method: 'GET' | 'POST' | 'PUT';
    action: CorrectionAction;
};
export const correctionPolicies = {
 MODULE_RETIRED:[410,"This module has been retired."],
    INVALID_REQUEST: [400, 'Invalid request.'], INVALID_COOKIE: [400, 'Invalid sign-in cookie.'], AUTHENTICATION_REQUIRED: [401, 'Sign in before continuing.'], CSRF_FAILED: [403, 'Refresh your sign-in before continuing.'], FORBIDDEN: [403, 'You do not have permission for this action.'], NOT_FOUND: [404, 'This correction or notification is unavailable.'], METHOD_NOT_ALLOWED: [405, 'This action is unavailable.'], PASSWORD_CHANGE_REQUIRED: [403, 'Change your password before continuing.'], REAUTHENTICATION_REQUIRED: [428, 'Confirm your password before continuing.'], RATE_LIMITED: [429, 'Too many requests. Try again later.'], IDEMPOTENCY_CONFLICT: [409, 'This request key was used for different input.'], SERVICE_UNAVAILABLE: [503, 'Service temporarily unavailable.'], CORRECTION_NOT_CONFIGURED: [503, 'Corrections are temporarily unavailable.'], CORRECTION_CONFLICT: [409, 'This correction changed. Reload before continuing.'], CORRECTION_SOURCE_STALE: [409, 'This source changed. Reload before continuing.'], CORRECTION_ANSWER_OVERLAP: [409, 'Finish or leave the overlapping assessment before reading this correction.'], CORRECTION_LEASE_LOST: [409, 'This job lease changed. Reload before continuing.']
} as const satisfies Record<CorrectionErrorCode, readonly [
    number,
    string
]>;
export class CorrectionRequestError extends Error {
    readonly status: number;
    constructor(public readonly code: CorrectionErrorCode = 'SERVICE_UNAVAILABLE', public readonly requestId = 'unavailable', public readonly retryAt?: string) { super(correctionPolicies[code][1]); this.name = 'CorrectionRequestError'; this.status = correctionPolicies[code][0]; }
}
