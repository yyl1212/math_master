import { z } from "zod";
import { readContentBytes } from "../content/bytes";
import { parseContentJSON, validContentString } from "../content/raw-json";
import type { QuestionRoute, QuestionAction, QuestionResult, QuestionErrorCode, QuestionListQuery } from "./types";
export { parseContentJSON as parseQuestionJSON };
export const questionUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
export const questionSHA = /^[0-9a-f]{64}$/;
const mathID = /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/;
const text = z.string().refine(validContentString), id = text.max(64).regex(mathID), uuid = text.regex(questionUUID), sha = text.regex(questionSHA), nullableSha = z.union([sha, z.literal("")]);
const instanceID = z.union([id, text.regex(/^qi-[0-9a-f]{64}$/)]);
const count = z.number().int().min(0).max(Number.MAX_SAFE_INTEGER), positive = count.min(1), version = positive.max(2147483647), time = z.iso.datetime({ offset: true });
const unique = <T>(a: T[]) => new Set(a).size === a.length;
const bytes = (s: string) => new TextEncoder().encode(s).byteLength;
export const questionCanonicalJSON = (v: unknown) => JSON.stringify(v).replace(/[<>&\u2028\u2029]/g, c => "\\u" + c.charCodeAt(0).toString(16).padStart(4, "0"));
const size = (v: unknown) => bytes(questionCanonicalJSON(v));
const note = text.refine(s => [...s].length <= 1000 && bytes(s) <= 3000);
const reason = note.refine(s => [...s].length >= 10 && s.trim() !== "");
const adoptReason = text.refine(s => [...s].length >= 10 && [...s].length <= 2000 && bytes(s) <= 6000 && s.trim() !== "");
const ref = z.object({ id, version }).strict();
const identity = z.object({ id: instanceID, version, sha256: sha }).strict();
const reportIdentity = identity.extend({ sha256: nullableSha }).strict();
const source = z.object({ kind: text, author: text, title: text, url: text, accessedAt: text, license: text, attribution: text }).strict();
const sourceLink = z.object({ knowledge: ref, batchSha256: text, relativePath: text, sha256: text, legacyId: text, note: text }).strict();
const sourceMap = z.array(sourceLink);
const rational = z.object({ numerator: text, denominator: text }).strict();
const parameterValue = z.object({ name: text, value: text }).strict();
const coverage = z.object({ knowledge: ref, objectiveIndices: z.array(count.max(2147483647)) }).strict();
const asset = z.object({ id, sha256: sha }).strict();
const engine = z.object({ family: z.enum(["rational_arithmetic", "rational_comparison", "missing_operand"]), operation: z.enum(["add", "subtract", "multiply", "divide", "compare"]), unknownSide: z.enum(["left", "right"]).nullable(), generatorVersion: version, verifierVersion: version }).strict().refine(e => e.family === "rational_comparison" ? e.operation === "compare" && e.unknownSide === null : e.operation !== "compare" && (e.family === "missing_operand" ? e.unknownSide !== null : e.unknownSide === null));
const witness = z.object({ engine, parameters: z.array(parameterValue) }).strict();
export const questionBodySchema = z.object({ type: z.enum(["single_choice", "numeric"]), knowledge: ref, coverage: z.array(coverage), units: z.array(ref), prompt: text, explanation: text, answerFormat: z.enum(["rational", "percentage"]).nullable(), choices: z.array(z.object({ id, text }).strict()), correctChoiceId: text.nullable(), correctNumeric: rational.nullable(), witness: witness.nullable(), assets: z.array(asset), sources: z.array(source) }).strict().refine(b => b.type === "numeric" ? b.answerFormat !== null && b.choices.length === 0 && b.correctChoiceId === null && b.correctNumeric !== null : b.answerFormat === null && b.correctNumeric === null && b.choices.length >= 1 && b.correctChoiceId !== null && mathID.test(b.correctChoiceId) && b.correctChoiceId.length <= 64);
export const templateSchema = z.object({ id, version, knowledge: ref, coverage: z.array(coverage), units: z.array(ref), type: z.enum(["single_choice", "numeric"]), answerFormat: z.enum(["rational", "percentage"]).nullable(), promptTemplate: text, explanationTemplate: text, engine, parameters: z.array(z.object({ name: text, values: z.array(text) }).strict()), constraints: z.array(z.enum(["nonzero_divisor", "nonnegative_result", "distinct_operands"])), distractors: z.array(z.enum(["negate", "plus_one", "minus_one", "reciprocal"])), assets: z.array(asset), sources: z.array(source) }).strict().refine(t => {
    if (t.type === "numeric" ? t.answerFormat === null : t.answerFormat !== null)
        return false;
    const names = t.engine.family === "missing_operand" ? ["known", "result"] : ["left", "right"];
    return t.parameters.every(p => names.includes(p.name)) && (t.engine.family !== "rational_comparison" || t.type === "single_choice" && t.distractors.length === 0);
});
export const fixedQuestionSchema = z.object({ id, version, body: questionBodySchema }).strict();
export const blueprintSchema = z.object({ id, version, knowledge: ref, coreObjectiveIndices: z.array(count.max(2147483647)), sources: z.array(z.object({ kind: z.enum(["template", "instance"]), ref }).strict()), coverageNote: text, ruleVersion: z.literal(1), questionCount: z.literal(5), passCount: z.literal(4) }).strict();
export const questionPackageSchema = z.object({ kind: z.literal("question-bank"), schemaVersion: z.literal(1), id, version, templates: z.array(templateSchema), fixedQuestions: z.array(fixedQuestionSchema), blueprints: z.array(blueprintSchema) }).strict();
const issue = z.object({ code: text, path: text, message: text }).strict();
const generation = z.object({ template: reportIdentity, rawCombinations: count, excludedCombinations: count, validInstances: count, generatorVersion: version, verifierVersion: version, constraintCounts: z.array(z.object({ constraint: z.enum(["nonzero_divisor", "nonnegative_result", "distinct_operands"]), count }).strict()) }).strict().refine(g => g.rawCombinations >= g.excludedCombinations + g.validInstances);
export const coverageNodeSchema = z.object({ knowledge: reportIdentity, blueprint: identity.nullable(), effectiveInstances: count, fixedInstances: count, generatedInstances: count, assessmentInstances: count, duplicateInstances: count, coreObjectiveIndices: z.array(count), coveredObjectiveIndices: z.array(count), fiveQuestionFeasible: z.boolean(), ready: z.boolean(), reasons: z.array(issue), supplementaryObjectiveIndices: z.array(count) }).strict().refine(n => n.effectiveInstances === n.fixedInstances + n.generatedInstances && (!n.ready || n.blueprint !== null && n.fiveQuestionFeasible));
export const validationSchema = z.object({ structuralErrors: z.array(issue).max(100), completenessErrors: z.array(issue).max(100), humanReviewRequirements: z.array(issue).max(100), structuralTotal: count, completenessTotal: count, humanReviewTotal: count, truncated: z.boolean(), readyToSubmit: z.boolean(), digest: nullableSha, generation: z.array(generation), coverage: z.array(coverageNodeSchema), packageBytes: count, frozenBytes: count }).strict().refine(g => {
    const shown = g.structuralErrors.length + g.completenessErrors.length + g.humanReviewRequirements.length, total = g.structuralTotal + g.completenessTotal + g.humanReviewTotal;
    return shown <= 100 && g.structuralTotal >= g.structuralErrors.length && g.completenessTotal >= g.completenessErrors.length && g.humanReviewTotal >= g.humanReviewRequirements.length && g.truncated === (shown < total) && (!g.readyToSubmit || g.structuralTotal === 0 && g.completenessTotal === 0 && questionSHA.test(g.digest));
});
const authors = z.array(uuid).min(1).refine(a => unique(a) && a.every((v, i) => i === 0 || a[i - 1] < v));
const inputFields = { catalogueVersion: version, questionPackage: questionPackageSchema, sourceMap };
export const draftInputSchema = z.object(inputFields).strict();
export const saveDraftInputSchema = z.object({ ...inputFields, expectedRevision: positive }).strict();
export const draftViewSchema = z.object({ ...inputFields, id: uuid, ownerId: uuid, catalogueSha256: sha, revision: positive, status: z.enum(["editing", "submitted"]), createdAt: time, updatedAt: time, authorIds: authors, legacyUnattributed: z.boolean(), gate: validationSchema }).strict().refine(d => d.authorIds.includes(d.ownerId));
const draftSummary = z.object({ id: uuid, ownerId: uuid, packageId: id, status: z.enum(["editing", "submitted"]), createdAt: time, updatedAt: time, packageVersion: version, catalogueVersion: version, structuralTotal: count, completenessTotal: count, revision: positive }).strict();
const known = z.object({ id, version: count.max(2147483647), sha256: sha, kind: z.enum(["knowledge", "unit", "asset"]) }).strict().refine(v => v.kind === "asset" ? v.version === 0 : v.version > 0);
export const frozenSchema = z.object({ catalogueVersion: version, catalogueSha256: sha, questionPackage: questionPackageSchema, sourceMap, authorIds: authors, legacyUnattributed: z.boolean(), resolved: z.array(known), objectives: z.array(z.object({ knowledge: identity, objectiveIndex: count, text }).strict()), generation: z.array(generation.refine(g => questionSHA.test(g.template.sha256) && g.rawCombinations === g.excludedCombinations + g.validInstances)), instanceIdentities: z.array(identity), coverage: z.array(coverageNodeSchema), generatorVersions: z.array(version), verifierVersions: z.array(version), frozenDigest: sha }).strict();
export const reviewChecksSchema = z.object({ mathematics: z.boolean(), explanations: z.boolean(), objectives: z.boolean(), sources: z.boolean(), illustrations: z.boolean(), generation: z.boolean() }).strict();
export const reviewInputSchema = z.object({ decision: z.enum(["approve", "return"]), checks: reviewChecksSchema, independenceNote: note, generationNote: note, note: reason }).strict().refine(v => v.decision !== "approve" || Object.values(v.checks).every(Boolean) && reason.safeParse(v.independenceNote).success && reason.safeParse(v.generationNote).success);
const review = reviewInputSchema.safeExtend({ id: uuid, submissionId: uuid, reviewerId: uuid, frozenDigest: sha, createdAt: time });
export const submissionViewSchema = z.object({ id: uuid, workspaceId: uuid, ownerId: uuid, status: z.enum(["pending", "approved", "returned"]), createdAt: time, revision: positive, frozen: frozenSchema, gate: validationSchema, review: review.nullable() }).strict().refine(s => s.frozen.authorIds.includes(s.ownerId) && (s.status === "pending" ? s.review === null : s.review !== null && s.review.submissionId === s.id && s.review.frozenDigest === s.frozen.frozenDigest && (s.status === "approved" ? s.review.decision === "approve" && !s.frozen.authorIds.includes(s.review.reviewerId) : s.review.decision === "return")));
const submissionSummary = z.object({ id: uuid, workspaceId: uuid, ownerId: uuid, packageId: id, status: z.enum(["pending", "approved", "returned"]), frozenDigest: sha, createdAt: time, revision: positive, packageVersion: version, catalogueVersion: version }).strict();
export const instanceSchema = z.object({ identity, origin: z.enum(["fixed", "template"]), template: identity.nullable(), parameters: z.array(parameterValue), generatorVersion: version.nullable(), verifierVersion: version.nullable(), body: questionBodySchema }).strict().refine(i => i.origin === "fixed" ? i.template === null && i.parameters.length === 0 && i.generatorVersion === null && i.verifierVersion === null : i.template !== null && i.generatorVersion !== null && i.verifierVersion !== null);
const memberIdentity = z.object({ kind: z.enum(["template", "instance", "blueprint"]), id: instanceID, packageId: id, sha256: sha, version, packageVersion: version }).strict().refine(m => m.kind === "instance" || mathID.test(m.id) && m.id.length <= 64);
const evidence = z.object({ submissionId: uuid, decisionId: uuid, frozenDigest: sha, inheritedFrom: uuid.nullable() }).strict();
export const manifestMemberSchema = z.object({ identity: memberIdentity, evidence }).strict();
export const changeSchema = z.object({ kind: z.enum(["template", "instance", "blueprint"]), id: instanceID, reason: text, before: memberIdentity.nullable(), after: memberIdentity.nullable() }).strict().refine(c => (c.before !== null || c.after !== null) && [c.before, c.after].every(v => v === null || v.kind === c.kind && v.id === c.id));
const diff = z.object({ added: count, replaced: count, removed: count }).strict();
export const publicationSchema = z.object({ id: uuid, status: z.enum(["prepared", "published"]), manifestSha: sha, createdAt: time, baseKnowledgeHead: uuid.nullable(), baseQuestionHead: uuid.nullable(), catalogueVersion: version, catalogueSha256: sha, templateCount: count.max(200), instanceCount: count.max(10000), blueprintCount: count.max(1000), diff }).strict();
const pageFields = <T extends z.ZodType>(item: T) => ({ items: z.array(item).max(100), total: count, limit: positive.max(100), offset: count.max(100000) });
const validPage = (p: {
    items: unknown[];
    total: number;
    limit: number;
    offset: number;
}) => p.items.length <= p.limit && p.items.length <= Math.max(0, p.total - p.offset);
const page = <T extends z.ZodType>(item: T) => z.object(pageFields(item)).strict().refine(validPage);
export const draftPageSchema = page(draftSummary), submissionPageSchema = page(submissionSummary), instancePageSchema = page(instanceSchema);
export const publicationPageSchema = z.object({ ...pageFields(publicationSchema), head: uuid.nullable() }).strict().refine(validPage);
export const memberPageSchema = z.object({ ...pageFields(manifestMemberSchema), publicationId: uuid, manifestSha: sha }).strict().refine(validPage);
export const changePageSchema = z.object({ ...pageFields(changeSchema), publicationId: uuid, manifestSha: sha }).strict().refine(validPage);
export const withdrawalTargetSchema = z.object({ kind: z.enum(["template", "instance", "blueprint"]), id: instanceID, version }).strict().refine(v => v.kind === "instance" || mathID.test(v.id) && v.id.length <= 64);
export const withdrawalPreviewSchema = z.object({ currentKnowledgeHead: uuid.nullable(), currentQuestionHead: uuid.nullable(), target: withdrawalTargetSchema, affectedTemplates: count, affectedInstances: count, affectedBlueprints: count, diff, changes: page(changeSchema), impactDigest: sha }).strict().refine(v => v.changes.total === v.diff.added + v.diff.replaced + v.diff.removed);
export const withdrawalResultSchema = z.object({ eventId: uuid, previousKnowledgeHead: uuid.nullable(), previousQuestionHead: uuid.nullable(), publication: publicationSchema }).strict().refine(v => v.publication.status === "published" && v.previousKnowledgeHead === v.publication.baseKnowledgeHead && v.previousQuestionHead === v.publication.baseQuestionHead);
export const coverageSchema = z.object({ knowledgeHead: uuid.nullable(), questionHead: uuid.nullable(), publishedKnowledge: count, approvedTemplates: count.max(200), fixedQuestions: count, effectiveInstances: count.max(10000), duplicateInstances: count, nodes: page(coverageNodeSchema) }).strict();
const dualHeads = { expectedKnowledgeHead: uuid.nullable(), expectedQuestionHead: uuid.nullable() };
const inputSchemas = { createDraft: draftInputSchema, saveDraft: saveDraftInputSchema, adoptDraft: z.object({ packageId: id, packageVersion: version, reason: adoptReason }).strict(), validateDraft: z.object({ expectedRevision: positive }).strict(), submitDraft: z.object({ expectedRevision: positive, expectedDigest: sha }).strict(), reviseSubmission: z.object({}).strict(), decideReview: reviewInputSchema, prepareRelease: z.object({ ...dualHeads, submissionIds: z.array(uuid).min(1).max(20).refine(unique), reason }).strict(), activateRelease: z.object({ ...dualHeads, expectedManifestSha: sha, reason }).strict(), previewWithdrawal: z.object({ target: withdrawalTargetSchema }).strict(), withdrawVersion: z.object({ ...dualHeads, target: withdrawalTargetSchema, reason }).strict() };
export const questionInputLimit = (a: QuestionAction) => a === "createDraft" || a === "saveDraft" ? 4194304 : 8192;
export const questionIdempotent = (a: QuestionAction) => Object.hasOwn(inputSchemas, a) && a !== "validateDraft" && a !== "previewWithdrawal";
export function questionInputError(a: QuestionAction, v: unknown): QuestionErrorCode | null {
    if (!Object.hasOwn(inputSchemas, a) || !inputSchemas[a as keyof typeof inputSchemas].safeParse(v).success)
        return "INVALID_REQUEST";
    if (bytes(JSON.stringify(v)) > questionInputLimit(a))
        return "PAYLOAD_TOO_LARGE";
    if (a === "createDraft" || a === "saveDraft") {
        const d = v as z.infer<typeof draftInputSchema>, p = d.questionPackage;
        if (size(v) > 4194304 || size(p) > 2097152 || size(d.sourceMap) > 262144 || p.templates.length > 50 || p.fixedQuestions.length > 200 || p.blueprints.length > 100 || p.templates.some(t => t.parameters.length > 4 || t.parameters.some(p => p.values.length > 32) || t.coverage.length > 4 || t.assets.length > 8 || bytes(t.promptTemplate) + bytes(t.explanationTemplate) > 8192) || p.fixedQuestions.some(q => q.body.coverage.length > 4 || q.body.assets.length > 8 || q.body.choices.length > 6 || bytes(q.body.prompt) + bytes(q.body.explanation) > 8192))
            return "QUESTION_LIMIT_EXCEEDED";
    }
    return null;
}
const listPaths = { listDrafts: "drafts", listSubmissions: "submissions", listPublications: "publications", readCoverage: "coverage" };
const plainPaths = { createDraft: ["drafts", "POST"], adoptDraft: ["drafts/adopt", "POST"], prepareRelease: ["publications/prepare", "POST"], withdrawVersion: ["withdrawals", "POST"], previewWithdrawal: ["withdrawals/preview", "POST"] } as const;
const idPaths = { readDraft: ["drafts", "", "GET"], saveDraft: ["drafts", "", "PUT"], validateDraft: ["drafts", "/validate", "POST"], submitDraft: ["drafts", "/submit", "POST"], readSubmission: ["submissions", "", "GET"], reviseSubmission: ["submissions", "/revision", "POST"], decideReview: ["submissions", "/decision", "POST"], listInstances: ["submissions", "/instances", "GET"], readPublication: ["publications", "", "GET"], activateRelease: ["publications", "/activate", "POST"], listMembers: ["publications", "/members", "GET"], listChanges: ["publications", "/changes", "GET"] } as const;
export const questionHasQuery = (a: QuestionAction) => Object.hasOwn(listPaths, a) || ["listInstances", "listMembers", "listChanges", "previewWithdrawal"].includes(a);
export function normalizeQuestionQuery(a: QuestionAction, q: QuestionListQuery & {
    knowledgeId?: string;
} = {}): Record<string, string | number> | null {
    if (!q || typeof q !== "object" || Array.isArray(q))
        return null;
    const list = Object.hasOwn(listPaths, a) && a !== "readCoverage", allowed = ["limit", "offset", ...(list ? ["scope", "status"] : a === "readCoverage" ? ["knowledgeId"] : [])];
    if (Object.keys(q).some(k => !allowed.includes(k)))
        return null;
    if (q.limit !== undefined && (!Number.isSafeInteger(q.limit) || q.limit < 1 || q.limit > 100) || q.offset !== undefined && (!Number.isSafeInteger(q.offset) || q.offset < 0 || q.offset > 100000))
        return null;
    const scopes: Record<string, string[]> = { listDrafts: ["mine", "all"], listSubmissions: ["mine", "review", "all"], listPublications: ["all"] };
    const statuses: Record<string, string[]> = { listDrafts: ["editing", "submitted"], listSubmissions: ["pending", "approved", "returned"], listPublications: ["prepared", "published"] };
    if (q.scope !== undefined && !scopes[a]?.includes(q.scope) || q.status !== undefined && !statuses[a]?.includes(q.status) || q.knowledgeId !== undefined && !id.safeParse(q.knowledgeId).success)
        return null;
    return Object.fromEntries(Object.entries(q).filter(([, v]) => v !== undefined)) as Record<string, string | number>;
}
export function questionRouteRequest(route: QuestionRoute): {
    method: string;
    path: string;
} | null {
    if (!route || typeof route !== "object" || Array.isArray(route))
        return null;
    const a = route.kind, hasQuery = questionHasQuery(a), hasId = Object.hasOwn(idPaths, a), allowed = ["kind", ...(hasQuery ? ["query"] : []), ...(hasId ? ["id"] : [])];
    if (Object.keys(route).some(k => !allowed.includes(k)))
        return null;
    let suffix = "", method = "GET";
    if (hasId) {
        if (!("id" in route) || !questionUUID.test(route.id))
            return null;
        const [group, op, verb] = idPaths[a as keyof typeof idPaths];
        suffix = group + "/" + route.id + op;
        method = verb;
    }
    else if (Object.hasOwn(plainPaths, a)) {
        [suffix, method] = plainPaths[a as keyof typeof plainPaths];
    }
    else if (Object.hasOwn(listPaths, a)) {
        suffix = listPaths[a as keyof typeof listPaths];
    }
    else
        return null;
    if (hasQuery) {
        const q = normalizeQuestionQuery(a, "query" in route ? route.query : undefined);
        if (!q)
            return null;
        const params = new URLSearchParams();
        for (const [k, v] of Object.entries(q))
            params.set(k, String(v));
        if (params.size)
            suffix += "?" + params.toString();
    }
    return { method, path: "/api/v1/question-bank/" + suffix };
}
export const questionPolicies: Record<QuestionErrorCode, {
    status: number;
    message: string;
}> = {
    "INVALID_REQUEST": {
        "status": 400,
        "message": "Invalid request."
    },
    "INVALID_COOKIE": {
        "status": 400,
        "message": "Invalid sign-in cookie."
    },
    "AUTHENTICATION_REQUIRED": {
        "status": 401,
        "message": "Please sign in to continue."
    },
    "CSRF_FAILED": {
        "status": 403,
        "message": "Request verification failed."
    },
    "FORBIDDEN": {
        "status": 403,
        "message": "You do not have permission."
    },
    "PASSWORD_CHANGE_REQUIRED": {
        "status": 403,
        "message": "Change your password to continue."
    },
    "NOT_FOUND": {
        "status": 404,
        "message": "Resource not found."
    },
    "METHOD_NOT_ALLOWED": {
        "status": 405,
        "message": "Method not allowed."
    },
    "QUESTION_DRAFT_CONFLICT": {
        "status": 409,
        "message": "This draft has changed. Reload before continuing."
    },
    "QUESTION_PUBLICATION_STALE": {
        "status": 409,
        "message": "Published snapshots changed. Prepare again."
    },
    "REVIEW_CONFLICT": {
        "status": 409,
        "message": "This submission already has a final decision."
    },
    "IDEMPOTENCY_CONFLICT": {
        "status": 409,
        "message": "This request key was used for different input."
    },
    "IMMUTABLE_CONFLICT": {
        "status": 409,
        "message": "This fixed version conflicts with saved questions."
    },
    "VERSION_CONFLICT": {
        "status": 409,
        "message": "Create a new version for changed questions."
    },
    "PAYLOAD_TOO_LARGE": {
        "status": 413,
        "message": "This request exceeds the size limit."
    },
    "QUESTION_INVALID": {
        "status": 422,
        "message": "Question validation failed."
    },
    "QUESTION_NOT_READY": {
        "status": 422,
        "message": "Complete the required questions before submitting."
    },
    "QUESTION_LIMIT_EXCEEDED": {
        "status": 422,
        "message": "Split this question bank into smaller reviewed batches."
    },
    "REVIEW_REQUIRED": {
        "status": 422,
        "message": "Independent review is required."
    },
    "REAUTHENTICATION_REQUIRED": {
        "status": 428,
        "message": "Verify your password before continuing."
    },
    "RATE_LIMITED": {
        "status": 429,
        "message": "Too many requests. Try again later."
    },
    "QUESTION_BANK_NOT_CONFIGURED": {
        "status": 503,
        "message": "Question management is temporarily unavailable."
    },
    "AUTH_NOT_CONFIGURED": {
        "status": 503,
        "message": "Accounts are temporarily unavailable."
    },
    "SERVICE_UNAVAILABLE": {
        "status": 503,
        "message": "Service temporarily unavailable."
    }
};
export const questionFailure = (code: QuestionErrorCode = "SERVICE_UNAVAILABLE", requestId = "unavailable"): Extract<QuestionResult<never>, {
    ok: false;
}> => ({ ok: false, code, ...questionPolicies[code], requestId });
const requestIdPattern = /^(?:[0-9a-f]{32}|unavailable)$/;
const errorSchema = z.object({ error: z.object({ code: z.enum(Object.keys(questionPolicies) as [
            QuestionErrorCode,
            ...QuestionErrorCode[]
        ]), message: text, requestId: text.regex(requestIdPattern) }).strict() }).strict().refine(v => v.error.message === questionPolicies[v.error.code].message);
const outputs = { listDrafts: draftPageSchema, createDraft: draftViewSchema, readDraft: draftViewSchema, saveDraft: draftViewSchema, adoptDraft: draftViewSchema, validateDraft: validationSchema, submitDraft: submissionViewSchema, listSubmissions: submissionPageSchema, readSubmission: submissionViewSchema, listInstances: instancePageSchema, reviseSubmission: draftViewSchema, decideReview: submissionViewSchema, listPublications: publicationPageSchema, readPublication: publicationSchema, listMembers: memberPageSchema, listChanges: changePageSchema, prepareRelease: publicationSchema, activateRelease: publicationSchema, previewWithdrawal: withdrawalPreviewSchema, withdrawVersion: withdrawalResultSchema, readCoverage: coverageSchema };
export const questionSuccessStatus = (a: QuestionAction) => ["createDraft", "adoptDraft", "submitDraft", "reviseSubmission", "prepareRelease", "withdrawVersion"].includes(a) ? 201 : 200;
export async function readQuestionResponse(response: Response, a: QuestionAction, signal?: AbortSignal): Promise<QuestionResult<unknown>> {
    const controller = signal ? undefined : new AbortController(), timer = controller ? setTimeout(() => controller.abort(), 10000) : undefined;
    try {
        const requestId = response.headers.get("X-Request-ID");
        if (response.headers.has("Set-Cookie") || response.redirected || response.status >= 300 && response.status < 400 || !requestId || !requestIdPattern.test(requestId) || response.headers.get("Cache-Control") !== "private, no-store" || response.headers.get("X-Content-Type-Options") !== "nosniff" || !/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(response.headers.get("Content-Type") ?? ""))
            throw new Error("Invalid question response.");
        const raw = parseContentJSON(await readContentBytes(response, 4194304, signal ?? controller!.signal));
        if (!response.ok) {
            const e = errorSchema.parse(raw).error;
            if (questionPolicies[e.code].status !== response.status || e.requestId !== requestId)
                throw new Error("Invalid question error.");
            const result = questionFailure(e.code, requestId);
            if (e.code === "RATE_LIMITED") {
                const retry = response.headers.get("Retry-After");
                if (!retry || !/^(?:[1-9]|[1-5][0-9]|60)$/.test(retry))
                    throw new Error("Invalid retry delay.");
                result.retryAfter = Number(retry);
            }
            else if (response.headers.has("Retry-After"))
                throw new Error("Unexpected retry delay.");
            return result;
        }
        if (response.status !== questionSuccessStatus(a) || response.headers.has("Retry-After") || !Object.hasOwn(outputs, a))
            throw new Error("Invalid question status.");
        return { ok: true, data: outputs[a].parse(raw) };
    }
    catch {
        void response.body?.cancel().catch(() => { });
        return questionFailure();
    }
    finally {
        if (timer !== undefined)
            clearTimeout(timer);
    }
}
