import { z } from "zod";
import { parseContentJSON, validContentString } from "../content/raw-json";
import { readContentBytes } from "../content/bytes";
import { validSVGMarkup } from "../content/svg-markup";
import type { LearningAction, LearningRoute, LearningResult, LearningErrorCode, Identity } from "./types";
export const learningUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
export const learningSHA = /^[0-9a-f]{64}$/;
const text = z.string().refine(validContentString), id = text.max(64).regex(/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/), uuid = text.regex(learningUUID), sha = text.regex(learningSHA), count = z.number().int().min(0).max(Number.MAX_SAFE_INTEGER), version = count.min(1).max(2147483647), time = z.iso.datetime({ offset: true });
export const identitySchema = z.object({ id, version, sha256: sha }).strict();
export const instanceIdentitySchema = identitySchema.extend({ id: z.union([id, text.regex(/^qi-[0-9a-f]{64}$/)]) }).strict();
const validity = z.enum(["effective", "restricted", "stale"]), mode = z.enum(["node", "diagnostic", "review"]), reason = z.enum(["knowledge-updated", "knowledge-withdrawn", "unit-withdrawn", "asset-withdrawn", "template-withdrawn", "instance-withdrawn", "blueprint-withdrawn", "exposed-after-creation", "grading-issue"]), reasons = z.array(reason).max(9);
const same = (a: Identity, b: Identity) => a.id === b.id && a.version === b.version && a.sha256 === b.sha256;
const uniqueIdentities = (values: Identity[]) => new Set(values.map(v => `${v.id}/${v.version}/${v.sha256}`)).size === values.length;
const qualification = z.object({ knowledge: identitySchema, kind: z.enum(["normal", "diagnostic"]), evidenceAttemptId: uuid, completedEventId: uuid.nullable(), correctionId: uuid.nullable(), validity: z.literal("effective") }).strict();
export const knowledgeStateSchema = z.object({ knowledge: identitySchema, title: text, titleZh: text, state: z.enum(["unlearned", "learning", "learned", "needs-review", "mastered"]), canEnter: z.boolean(), everUnlocked: z.boolean(), completionValid: z.boolean(), startedAt: time.nullable(), completedAt: time.nullable(), qualification: qualification.nullable(), prerequisites: z.array(z.object({ knowledge: identitySchema, qualified: z.boolean(), reasons }).strict()).max(1000) }).strict().refine(v => v.qualification === null || same(v.knowledge, v.qualification.knowledge));
const blueprint = z.object({ blueprint: identitySchema, coreObjectiveIndices: z.array(count).min(1).max(8).refine(v => new Set(v).size === v.length), ready: z.boolean(), reasons: z.array(z.enum(["no-blueprint", "no-instances", "insufficient-coverage", "exposure-cooldown", "recent-assessment-exclusion", "grading-issue"])).max(6), retryAt: time.nullable() }).strict().refine(v => !v.ready || v.reasons.length === 0 && v.retryAt === null);
const summaryFields = { id: uuid, knowledge: identitySchema, createdAt: time, expiresAt: time, submittedAt: time.nullable() };
export const attemptSummarySchema = z.discriminatedUnion("kind", [z.object({ ...summaryFields, kind: z.literal("practice"), mode: z.null(), state: z.enum(["active", "answered", "revealed", "abandoned", "expired"]) }).strict(), z.object({ ...summaryFields, kind: z.literal("assessment"), mode, state: z.enum(["active", "submitted", "abandoned", "expired"]) }).strict()]).refine(v => Date.parse(v.expiresAt) > Date.parse(v.createdAt));
export const knowledgeDetailSchema = z.object({ knowledgeHead: uuid, questionHead: uuid.nullable(), state: knowledgeStateSchema, objectives: z.array(text), blueprints: z.array(blueprint).max(1000), activeAssessment: attemptSummarySchema.nullable() }).strict();
export const pathSummarySchema = z.object({ id: uuid, path: identitySchema, title: text, titleZh: text, knowledgePublicationId: uuid, totalNodes: count.min(1).max(1000), completedNodes: count.max(1000), passedNodes: count.max(1000), unlockedNodes: count.max(1000), newVersionAvailable: z.boolean(), createdAt: time }).strict().refine(v => Math.max(v.completedNodes, v.passedNodes, v.unlockedNodes) <= v.totalNodes);
export const pathNodeSchema = z.object({ position: count.max(999), title: text, titleZh: text, state: knowledgeStateSchema, available: z.boolean(), reasons }).strict();
export const pathViewSchema = z.object({ summary: pathSummarySchema }).strict();
export const historyEntrySchema = z.object({ id: uuid, kind: z.enum(["started", "completed", "practice", "assessment", "enrollment"]), path: identitySchema.nullable(), knowledge: identitySchema, occurredAt: time, state: z.enum(["started", "completed", "active", "submitted", "answered", "revealed", "abandoned", "expired", "enrolled"]), validity, attemptId: uuid.nullable() }).strict();
export const overviewSchema = z.object({ knowledgeHead: uuid.nullable(), questionHead: uuid.nullable(), availablePaths: z.array(z.object({ path: identitySchema, title: text, titleZh: text, totalNodes: count.min(1).max(1000) }).strict()).max(200), startedCount: count, completedCount: count, effectivePassedCount: count, historicalUnlockedCount: count, activePractice: attemptSummarySchema.nullable(), activeAssessment: attemptSummarySchema.nullable(), recent: z.array(historyEntrySchema).max(20) }).strict();
const page = <T extends z.ZodType>(item: T) => z.object({ items: z.array(item).max(100), total: count, limit: count.min(1).max(100), offset: count.max(100000) }).strict().refine(v => v.items.length <= v.limit && v.items.length <= Math.max(0, v.total - v.offset));
const choice = z.object({ id, text }).strict(), asset = z.object({ id, sha256: sha }).strict();
const questionFields = { position: count.min(1).max(5), instance: instanceIdentitySchema, knowledge: identitySchema, prompt: text, assets: z.array(asset).max(8) };
export const safeQuestionSchema = z.discriminatedUnion("type", [z.object({ ...questionFields, type: z.literal("single_choice"), choices: z.array(choice).min(2).max(6), answerFormat: z.null() }).strict(), z.object({ ...questionFields, type: z.literal("numeric"), choices: z.array(choice).length(0), answerFormat: z.enum(["rational", "percentage"]) }).strict()]);
const numericAnswer = z.object({ kind: z.literal("numeric"), raw: text.refine(v => [...v].length <= 128) }).strict();
const choiceAnswer = z.object({ kind: z.literal("choice"), choiceId: id }).strict();
export const answerSchema = z.discriminatedUnion("kind", [choiceAnswer, numericAnswer, z.object({ kind: z.literal("skipped") }).strict()]);
export const practiceAnswerSchema = z.discriminatedUnion("kind", [choiceAnswer, numericAnswer]);
const rational = z.object({ numerator: text.regex(/^(0|-?[1-9][0-9]{0,255})$/), denominator: text.regex(/^[1-9][0-9]{0,255}$/) }).strict();
export const resultItemSchema = z.object({ question: safeQuestionSchema, answer: answerSchema.nullable(), correct: z.boolean().nullable(), correctChoiceId: id.nullable(), correctNumeric: rational.nullable(), explanation: text.nullable(), validity, reasons }).strict().refine(v => v.validity !== "restricted" || v.correct === null && v.correctChoiceId === null && v.correctNumeric === null && v.explanation === null);
export const attemptViewSchema = z.object({ summary: attemptSummarySchema, questions: z.array(safeQuestionSchema).length(5) }).strict().refine(v => v.summary.kind === "assessment" && uniqueIdentities(v.questions.map(q => q.instance)) && v.questions.every((q, i) => q.position === i + 1 && same(q.knowledge, v.summary.knowledge)));
export const practiceViewSchema = z.object({ summary: attemptSummarySchema, question: safeQuestionSchema, result: z.object({ outcome: z.enum(["answered", "revealed"]), item: resultItemSchema }).strict().nullable() }).strict().refine(v => v.summary.kind === "practice" && v.question.position === 1 && same(v.summary.knowledge, v.question.knowledge) && (["answered", "revealed"].includes(v.summary.state) ? v.result !== null && v.result.outcome === v.summary.state && same(v.result.item.question.instance, v.question.instance) : v.result === null));
export const resultViewSchema = z.object({ summary: attemptSummarySchema, ruleVersion: z.literal(1), score: count.max(5).nullable(), passed: z.boolean().nullable(), outcome: z.enum(["passed", "failed", "affected"]), validity, reasons, items: z.array(resultItemSchema).length(5), progress: z.object({ knowledge: identitySchema, qualificationGranted: z.boolean(), newlyUnlocked: z.array(identitySchema).max(1000) }).strict() }).strict().refine(v => v.summary.kind === "assessment" && v.summary.state === "submitted" && same(v.progress.knowledge, v.summary.knowledge) && v.items.every((q, i) => q.question.position === i + 1 && same(q.question.knowledge, v.summary.knowledge) && q.answer !== null) && (v.outcome === "affected" ? v.score === null && v.passed === null && v.items.every(i => i.correct === null) : v.score !== null && v.passed !== null && (v.outcome === "passed" ? v.score >= 4 && v.passed : v.score < 4 && !v.passed)));
const startInput = z.object({ knowledge: identitySchema, expectedKnowledgeHead: uuid }).strict();
export const submitInputSchema = z.object({ answers: z.array(z.object({ position: count.min(1).max(5), instance: instanceIdentitySchema, answer: answerSchema }).strict()).length(5) }).strict().refine(v => new Set(v.answers.map(a => a.position)).size === 5 && uniqueIdentities(v.answers.map(a => a.instance)));
const inputs = { startKnowledge: startInput, completeKnowledge: startInput, enrollPath: z.object({ path: identitySchema, expectedKnowledgeHead: uuid }).strict(), createPractice: startInput.extend({ expectedQuestionHead: uuid }).strict(), answerPractice: practiceAnswerSchema, revealPractice: z.object({}).strict(), abandonPractice: z.object({}).strict(), createAssessment: startInput.extend({ blueprint: identitySchema, mode, expectedQuestionHead: uuid }).strict(), submitAssessment: submitInputSchema, abandonAssessment: z.object({}).strict() };
function answerInputError(value: unknown, skipped: boolean): LearningErrorCode | null {
    const numeric = z.object({ kind: z.literal("numeric"), raw: text }).strict().safeParse(value);
    if (numeric.success)
        return [...numeric.data.raw].length > 128 ? "ANSWER_FORMAT_INVALID" : null;
    if (choiceAnswer.safeParse(value).success || skipped && z.object({ kind: z.literal("skipped") }).strict().safeParse(value).success)
        return null;
    return "INVALID_REQUEST";
}
export function learningInputError(action: LearningAction, value: unknown): LearningErrorCode | null {
    if (!Object.hasOwn(inputs, action))
        return "INVALID_REQUEST";
    if (action === "answerPractice")
        return answerInputError(value, false);
    if (action === "submitAssessment") {
        const outer = z.object({ answers: z.array(z.unknown()).length(5) }).strict().safeParse(value);
        if (!outer.success)
            return "INVALID_REQUEST";
        const positions = new Set<number>(), identities = new Set<string>();
        for (const raw of outer.data.answers) {
            const parsed = z.object({ position: count.min(1).max(5), instance: instanceIdentitySchema, answer: z.unknown() }).strict().safeParse(raw);
            if (!parsed.success)
                return "INVALID_REQUEST";
            const a = parsed.data, k = `${a.instance.id}/${a.instance.version}/${a.instance.sha256}`;
            if (positions.has(a.position) || identities.has(k))
                return "INVALID_REQUEST";
            const error = answerInputError(a.answer, true);
            if (error)
                return error;
            positions.add(a.position);
            identities.add(k);
        }
        return null;
    }
    return inputs[action as keyof typeof inputs].safeParse(value).success ? null : "INVALID_REQUEST";
}
const query = z.object({ limit: count.min(1).max(100).optional(), offset: count.max(100000).optional() }).strict();
const routeSchema = z.union([z.object({ kind: z.enum(["readOverview", "createPractice", "createAssessment"]) }).strict(), z.object({ kind: z.enum(["listKnowledge", "listPaths", "listHistory"]), query: query.optional() }).strict(), z.object({ kind: z.literal("readKnowledge"), id, version }).strict(), z.object({ kind: z.enum(["startKnowledge", "completeKnowledge", "enrollPath"]), id }).strict(), z.object({ kind: z.enum(["readPath", "readPractice", "answerPractice", "revealPractice", "abandonPractice", "readAssessment", "submitAssessment", "abandonAssessment", "readAssessmentResult"]), id: uuid }).strict(), z.object({ kind: z.literal("listPathNodes"), id: uuid, query: query.optional() }).strict(), z.object({ kind: z.literal("readAsset"), id: uuid, sha }).strict()]);
export function learningRouteRequest(value: LearningRoute): {
    path: string;
    method: "GET" | "POST";
} | null {
    const parsed = routeSchema.safeParse(value);
    if (!parsed.success)
        return null;
    const r = parsed.data;
    let path: string;
    switch (r.kind) {
        case "readOverview":
            path = "overview";
            break;
        case "listKnowledge":
            path = "knowledge";
            break;
        case "listPaths":
            path = "paths";
            break;
        case "listHistory":
            path = "history";
            break;
        case "readKnowledge":
            path = `knowledge/${r.id}?version=${r.version}`;
            break;
        case "startKnowledge":
            path = `knowledge/${r.id}/start`;
            break;
        case "completeKnowledge":
            path = `knowledge/${r.id}/complete`;
            break;
        case "enrollPath":
            path = `paths/${r.id}/enroll`;
            break;
        case "readPath":
            path = `paths/${r.id}`;
            break;
        case "listPathNodes":
            path = `paths/${r.id}/nodes`;
            break;
        case "createPractice":
            path = "practice";
            break;
        case "readPractice":
            path = `practice/${r.id}`;
            break;
        case "answerPractice":
            path = `practice/${r.id}/answer`;
            break;
        case "revealPractice":
            path = `practice/${r.id}/reveal`;
            break;
        case "abandonPractice":
            path = `practice/${r.id}/abandon`;
            break;
        case "createAssessment":
            path = "assessments";
            break;
        case "readAssessment":
            path = `assessments/${r.id}`;
            break;
        case "submitAssessment":
            path = `assessments/${r.id}/submit`;
            break;
        case "abandonAssessment":
            path = `assessments/${r.id}/abandon`;
            break;
        case "readAssessmentResult":
            path = `assessments/${r.id}/result`;
            break;
        case "readAsset":
            path = `assets/${r.id}/${r.sha}`;
            break;
    }
    if ("query" in r && r.query) {
        const params = new URLSearchParams();
        for (const k of ["limit", "offset"] as const) {
            const v = r.query[k];
            if (v !== undefined)
                params.set(k, String(v));
        }
        ;
        if (params.size)
            path += "?" + params.toString();
    }
    ;
    return { path: "/api/v1/learning/" + path, method: Object.hasOwn(inputs, r.kind) ? "POST" : "GET" };
}
export const learningSuccessStatus = (a: LearningAction) => ["startKnowledge", "enrollPath", "createPractice", "createAssessment"].includes(a) ? 201 : 200;
export const learningPolicies: Record<LearningErrorCode, {
    status: number;
    message: string;
}> = {
    LEARNING_NOT_CONFIGURED: { status: 503, message: "Learning is temporarily unavailable." }, LEARNING_VERSION_STALE: { status: 409, message: "Published learning content changed. Reload before continuing." }, LEARNING_PREREQUISITES_UNMET: { status: 409, message: "Complete the prerequisites or take a diagnostic assessment." }, ASSESSMENT_NOT_READY: { status: 409, message: "Five eligible questions are not available yet." }, ASSESSMENT_ACTIVE: { status: 409, message: "Continue or abandon the existing attempt." }, ASSESSMENT_EXPIRED: { status: 409, message: "This attempt has expired." }, ASSESSMENT_STATE_CONFLICT: { status: 409, message: "This attempt cannot accept the action." }, ANSWER_FORMAT_INVALID: { status: 400, message: "Check the requested answer format." }, INVALID_REQUEST: { status: 400, message: "Invalid request." }, INVALID_COOKIE: { status: 400, message: "Invalid sign-in cookie." }, INVALID_CREDENTIALS: { status: 401, message: "Invalid username or password." }, AUTHENTICATION_REQUIRED: { status: 401, message: "Please sign in to continue." }, CSRF_FAILED: { status: 403, message: "Request verification failed." }, FORBIDDEN: { status: 403, message: "You do not have permission." }, PASSWORD_CHANGE_REQUIRED: { status: 403, message: "Change your password to continue." }, NOT_FOUND: { status: 404, message: "Resource not found." }, METHOD_NOT_ALLOWED: { status: 405, message: "Method not allowed." }, IDEMPOTENCY_CONFLICT: { status: 409, message: "This request key was used for different input." }, PAYLOAD_TOO_LARGE: { status: 413, message: "This request exceeds the size limit." }, RATE_LIMITED: { status: 429, message: "Too many requests. Try again later." }, SERVICE_UNAVAILABLE: { status: 503, message: "Service temporarily unavailable." }, AUTH_NOT_CONFIGURED: { status: 503, message: "Accounts are temporarily unavailable." }, USERNAME_UNAVAILABLE: { status: 409, message: "This username is unavailable." }, ALREADY_AUTHENTICATED: { status: 409, message: "Sign out before using another account." }, LAST_ADMIN_REQUIRED: { status: 409, message: "At least one administrator is required." }, REAUTHENTICATION_REQUIRED: { status: 428, message: "Verify your password before continuing." }
};
export const learningFailure = (code: LearningErrorCode = "SERVICE_UNAVAILABLE", requestId = "unavailable"): Extract<LearningResult<never>, {
    ok: false;
}> => ({ ok: false, code, ...learningPolicies[code], requestId });
const errorFields = { message: text.max(500), requestId: text.regex(/^(?:[0-9a-f]{32}|unavailable)$/) };
export const learningErrorSchema = z.object({ error: z.union([z.object({ ...errorFields, code: z.literal("ASSESSMENT_NOT_READY"), retryAt: time.optional() }).strict(), z.object({ ...errorFields, code: z.literal("ASSESSMENT_ACTIVE"), activeAttempt: attemptSummarySchema.refine(v => v.state === "active").optional() }).strict(), z.object({ ...errorFields, code: z.literal("ANSWER_FORMAT_INVALID"), formatCode: z.enum(["INVALID_SYNTAX", "INPUT_TOO_LONG", "ZERO_DENOMINATOR", "PERCENT_REQUIRED", "RESULT_TOO_LARGE", "INVALID_MODE"]).optional() }).strict(), z.object({ ...errorFields, code: z.enum(Object.keys(learningPolicies).filter(c => !["ASSESSMENT_NOT_READY", "ASSESSMENT_ACTIVE", "ANSWER_FORMAT_INVALID"].includes(c)) as [
                LearningErrorCode,
                ...LearningErrorCode[]
            ]) }).strict()]) }).strict();
const outputs = { readOverview: overviewSchema, listKnowledge: page(knowledgeStateSchema), readKnowledge: knowledgeDetailSchema, startKnowledge: knowledgeStateSchema, completeKnowledge: knowledgeStateSchema, enrollPath: pathViewSchema, listPaths: page(pathSummarySchema), readPath: pathViewSchema, listPathNodes: page(pathNodeSchema), createPractice: practiceViewSchema, readPractice: practiceViewSchema, answerPractice: practiceViewSchema, revealPractice: practiceViewSchema, abandonPractice: practiceViewSchema, createAssessment: attemptViewSchema, readAssessment: attemptViewSchema, submitAssessment: resultViewSchema, abandonAssessment: attemptViewSchema, readAssessmentResult: resultViewSchema, listHistory: page(historyEntrySchema) };
export async function readLearningResponse(response: Response, route: LearningRoute, signal: AbortSignal, validateAsset?: (bytes: Uint8Array, sha: string) => boolean | Promise<boolean>): Promise<LearningResult<unknown>> {
    try {
        const requestId = response.headers.get("X-Request-ID");
        if (response.headers.has("Set-Cookie") || response.redirected || response.status >= 300 && response.status < 400 || !requestId || !/^(?:[0-9a-f]{32}|unavailable)$/.test(requestId) || response.headers.get("Cache-Control") !== "private, no-store" || response.headers.get("X-Content-Type-Options") !== "nosniff")
            throw new Error("Invalid private learning response.");
        if (route.kind === "readAsset" && response.status === 200) {
            if (response.headers.get("Content-Type") !== "image/svg+xml" || response.headers.get("Content-Security-Policy") !== "sandbox; default-src 'none'")
                throw new Error("Invalid SVG headers.");
            const bytes = await readContentBytes(response, 1 << 20, signal);
            let valid: boolean;
            if (validateAsset)
                valid = await validateAsset(bytes, route.sha);
            else {
                const hash = await crypto.subtle.digest("SHA-256", bytes.slice().buffer as ArrayBuffer);
                valid = [...new Uint8Array(hash)].map(b => b.toString(16).padStart(2, "0")).join("") === route.sha && validSVGMarkup(bytes);
            }
            ;
            if (!valid || signal.aborted)
                throw new Error("Invalid SVG bytes.");
            return { ok: true, data: bytes };
        }
        if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(response.headers.get("Content-Type") ?? ""))
            throw new Error("Invalid JSON headers.");
        const value = parseContentJSON(await readContentBytes(response, 4 << 20, signal));
        if (!response.ok) {
            const e = learningErrorSchema.parse(value).error;
            if (learningPolicies[e.code].status !== response.status || e.requestId !== requestId)
                throw new Error("Invalid learning error.");
            const result = learningFailure(e.code, requestId);
            if ("retryAt" in e && e.retryAt !== undefined)
                result.retryAt = e.retryAt;
            if ("activeAttempt" in e && e.activeAttempt !== undefined)
                result.activeAttempt = e.activeAttempt;
            if ("formatCode" in e && e.formatCode !== undefined)
                result.formatCode = e.formatCode;
            if (e.code === "RATE_LIMITED") {
                const retry = response.headers.get("Retry-After");
                if (!retry || !/^[1-9][0-9]*$/.test(retry) || !Number.isSafeInteger(Number(retry)))
                    throw new Error("Invalid retry time.");
                result.retryAfter = Number(retry);
            }
            ;
            return result;
        }
        ;
        if (route.kind === "readAsset" || response.status !== learningSuccessStatus(route.kind))
            throw new Error("Invalid success status.");
        return { ok: true, data: outputs[route.kind].parse(value) };
    }
    catch {
        return learningFailure();
    }
}
