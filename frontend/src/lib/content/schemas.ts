import { z } from "zod";
import { readContentBytes } from "./bytes";
import { parseContentJSON, validContentString } from "./raw-json";
import type { ContentEndpoint, ContentRoute, ContentResult, ContentErrorCode, ContentListQuery } from "./types";
export { parseContentJSON } from "./raw-json";
export const contentUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
export const contentSHA = /^[0-9a-f]{64}$/;
const mathID = /^[a-z][a-z0-9-]{0,63}$/;
const text = z.string().refine(validContentString), uuid = text.regex(contentUUID), id = text.regex(mathID), sha = text.regex(contentSHA);
const count = z.number().int().min(0).max(Number.MAX_SAFE_INTEGER), positive = count.min(1), version = positive.max(2147483647), time = z.iso.datetime({ offset: true });
const note = text.refine(v => [...v].length <= 1000 && new TextEncoder().encode(v).byteLength <= 3000);
const reason = note.refine(v => [...v].length >= 10 && v.trim() !== "");
const unique = <T>(a: T[]) => new Set(a).size === a.length;
const authors = z.array(uuid).min(1).refine(a => unique(a) && a.every((v, i) => i === 0 || a[i - 1] < v));
const ref = z.object({ id, version }).strict();
const relation = z.object({ kind: z.enum(["prerequisite", "derivation", "related"]), target: ref }).strict();
const source = z.object({ kind: z.enum(["original", "external"]), author: text, title: text, url: text, accessedAt: text, license: text, attribution: text }).strict();
export const knowledgeSchema = z.object({ id, version, domainIds: z.array(id), topicIds: z.array(id), type: z.enum(["concept", "definition", "axiom", "theorem", "corollary", "method", "mathematical-thinking"]), title: text, titleZh: text, statement: text, scope: text, objectives: z.array(text), conditions: z.array(text), system: text, proof: text, sources: z.array(source), relations: z.array(relation) }).strict();
export const unitSchema = z.object({ id, version, knowledge: ref, angles: z.array(z.object({ kind: text, body: text }).strict()), examples: z.array(text), counterexamples: z.array(text), assetIds: z.array(id) }).strict();
export const pathSchema = z.object({ id, version, domainIds: z.array(id), title: text, titleZh: text, nodes: z.array(ref) }).strict();
export const assetSchema = z.object({ id, path: text, sha256: sha, author: text, license: text, attribution: text, knowledge: ref }).strict();
const assetView = assetSchema.omit({ path: true }).strict();
export const packageSchema = z.object({ schemaVersion: z.literal(1), id, version, knowledge: z.array(knowledgeSchema).max(100), units: z.array(unitSchema).max(200), paths: z.array(pathSchema).max(20), assets: z.array(assetSchema).max(16) }).strict().refine(p => [p.knowledge, p.units, p.paths, p.assets].every(items => unique(items.map(v => v.id))));
const relativePath = (v: string) => v !== "" && !v.startsWith("/") && !/^[A-Za-z]:/.test(v) && !v.includes("\\") && v.split("/").every(s => s.trim() !== "" && s !== "." && s !== "..");
const sourceLink = z.object({ knowledge: ref, batchSha256: sha, relativePath: text.refine(relativePath), sha256: sha, legacyId: text, note: text }).strict();
const sourceMap = z.array(sourceLink);
const issue = z.object({ code: text, path: text, message: text }).strict();
export const gateSchema = z.object({ structuralErrors: z.array(issue).max(100), completenessErrors: z.array(issue).max(100), humanReviewRequirements: z.array(issue).max(100), structuralTotal: count, completenessTotal: count, humanReviewTotal: count, truncated: z.boolean(), readyToSubmit: z.boolean(), digest: sha }).strict().refine(g => {
    const shown = g.structuralErrors.length + g.completenessErrors.length + g.humanReviewRequirements.length;
    return shown <= 100 && g.structuralTotal >= g.structuralErrors.length && g.completenessTotal >= g.completenessErrors.length && g.humanReviewTotal >= g.humanReviewRequirements.length && g.readyToSubmit === (g.structuralTotal === 0 && g.completenessTotal === 0) && g.truncated === (shown < g.structuralTotal + g.completenessTotal + g.humanReviewTotal);
});
const assetMetadataMatches = (p: z.infer<typeof packageSchema>, assets: z.infer<typeof assetView>[]) => assets.length === p.assets.length && assets.every(a => { const expected = p.assets.find(v => v.id === a.id); return expected && a.sha256 === expected.sha256 && a.author === expected.author && a.license === expected.license && a.attribution === expected.attribution && a.knowledge.id === expected.knowledge.id && a.knowledge.version === expected.knowledge.version; }) && unique(assets.map(a => a.id));
const draftShape = { id: uuid, ownerId: uuid, catalogueSha256: sha, catalogueVersion: version, revision: positive, status: z.enum(["editing", "submitted"]), package: packageSchema, sourceMap, authorIds: authors, legacyUnattributed: z.boolean(), assets: z.array(assetView).max(16), gate: gateSchema, createdAt: time, updatedAt: time };
export const draftViewSchema = z.object(draftShape).strict().refine(d => d.authorIds.includes(d.ownerId) && assetMetadataMatches(d.package, d.assets));
const frozen = z.object({ catalogueVersion: version, catalogueSha256: sha, package: packageSchema, sourceMap, authorIds: authors, legacyUnattributed: z.boolean(), assets: z.array(assetView).max(16), frozenDigest: sha }).strict().refine(f => assetMetadataMatches(f.package, f.assets));
export const reviewChecksSchema = z.object({ mathematics: z.boolean(), explanations: z.boolean(), relationships: z.boolean(), sources: z.boolean(), illustrations: z.boolean() }).strict();
const reviewDecision = z.object({ id: uuid, submissionId: uuid, reviewerId: uuid, frozenDigest: sha, decision: z.enum(["approve", "return"]), checks: reviewChecksSchema, independenceNote: note, note: reason, createdAt: time }).strict();
export const submissionViewSchema = z.object({ id: uuid, workspaceId: uuid, ownerId: uuid, status: z.enum(["pending", "approved", "returned"]), revision: positive, frozen, gate: gateSchema, review: reviewDecision.nullable(), createdAt: time }).strict().refine(s => {
    if (!s.frozen.authorIds.includes(s.ownerId))
        return false;
    if (s.status === "pending")
        return s.review === null;
    if (!s.review || s.review.submissionId !== s.id || s.review.frozenDigest !== s.frozen.frozenDigest || s.review.decision !== (s.status === "approved" ? "approve" : "return"))
        return false;
    return s.status !== "approved" || Object.values(s.review.checks).every(Boolean) && reason.safeParse(s.review.independenceNote).success && !s.frozen.authorIds.includes(s.review.reviewerId);
});
const identity = z.object({ kind: z.enum(["knowledge", "unit", "path", "asset"]), id, packageId: id, sha256: sha, version, packageVersion: version }).strict().refine(m => m.kind !== "asset" || m.version === 1);
const evidence = z.object({ submissionId: uuid, decisionId: uuid, frozenDigest: sha, inheritedFrom: uuid.nullable() }).strict();
const member = z.object({ identity, evidence }).strict();
const binding = z.object({ unit: ref, assetId: id, sha256: sha }).strict();
const compareString = (a: string, b: string) => a < b ? -1 : a > b ? 1 : 0;
const compareMembers = (a: z.infer<typeof identity>, b: z.infer<typeof identity>) => compareString(a.kind, b.kind) || compareString(a.id, b.id) || a.version - b.version || compareString(a.packageId, b.packageId) || a.packageVersion - b.packageVersion || compareString(a.sha256, b.sha256);
export const manifestSchema = z.object({ catalogueVersion: version, catalogueSha256: sha, baseHead: uuid.nullable(), members: z.array(member).max(6200), bindings: z.array(binding).max(64000) }).strict().refine(m => {
    if (!unique(m.members.map(v => v.identity.kind + "/" + v.identity.id)) || m.members.some((v, i) => i > 0 && compareMembers(m.members[i - 1].identity, v.identity) >= 0) || m.members.some(v => v.evidence.inheritedFrom !== null && v.evidence.inheritedFrom !== m.baseHead))
        return false;
    const requiredUnits = new Map(m.members.filter(v => v.identity.kind === "unit").map(v => [v.identity.id, v.identity.version]));
    const assets = new Map(m.members.filter(v => v.identity.kind === "asset").map(v => [v.identity.id, v.identity.sha256]));
    return unique(m.bindings.map(b => b.unit.id + ":" + b.unit.version + ":" + b.assetId)) && m.bindings.every((b, i) => requiredUnits.get(b.unit.id) === b.unit.version && assets.get(b.assetId) === b.sha256 && (i === 0 || compareString(m.bindings[i - 1].unit.id, b.unit.id) < 0 || m.bindings[i - 1].unit.id === b.unit.id && (m.bindings[i - 1].unit.version < b.unit.version || m.bindings[i - 1].unit.version === b.unit.version && m.bindings[i - 1].assetId < b.assetId)));
});
const change = z.object({ kind: z.enum(["knowledge", "unit", "path", "asset"]), id, reason: text, before: identity.nullable(), after: identity.nullable() }).strict().refine(c => !!(c.before || c.after) && [c.before, c.after].every(m => !m || m.id === c.id && m.kind === c.kind));
export const diffSchema = z.object({ added: count, replaced: count, removed: count, changes: z.array(change).max(12400) }).strict().refine(d => d.added === d.changes.filter(c => c.before === null && c.after !== null).length && d.removed === d.changes.filter(c => c.before !== null && c.after === null).length && d.replaced === d.changes.filter(c => c.before !== null && c.after !== null).length && unique(d.changes.map(c => c.kind + "/" + c.id)));
export const publicationViewSchema = z.object({ id: uuid, status: z.enum(["draft", "published"]), manifestSha: sha, createdAt: time, manifest: manifestSchema, diff: diffSchema }).strict();
const draftSummary = z.object({ id: uuid, ownerId: uuid, packageId: id, status: z.enum(["editing", "submitted"]), createdAt: time, updatedAt: time, packageVersion: version, catalogueVersion: version, structuralTotal: count, completenessTotal: count, revision: positive }).strict();
const submissionSummary = z.object({ id: uuid, workspaceId: uuid, ownerId: uuid, packageId: id, status: z.enum(["pending", "approved", "returned"]), frozenDigest: sha, createdAt: time, packageVersion: version, catalogueVersion: version, revision: positive }).strict();
const page = <T extends z.ZodType>(item: T) => z.object({ items: z.array(item).max(100), total: count, limit: positive.max(100), offset: count.max(100000) }).strict().refine(p => p.items.length <= p.limit && p.items.length <= p.total);
export const draftPageSchema = page(draftSummary), submissionPageSchema = page(submissionSummary);
export const publicationPageSchema = z.object({ items: z.array(publicationViewSchema).max(100), total: count, limit: positive.max(100), offset: count.max(100000), head: uuid.nullable() }).strict().refine(p => p.items.length <= p.limit && p.items.length <= p.total);
export const withdrawalTargetSchema = z.union([z.object({ kind: z.enum(["knowledge", "unit", "path"]), id, version }).strict(), z.object({ kind: z.literal("asset"), sha256: sha }).strict()]);
export const withdrawalPreviewSchema = z.object({ currentHead: uuid.nullable(), target: withdrawalTargetSchema, diff: diffSchema }).strict();
export const withdrawalResultSchema = z.object({ eventId: uuid, previousHead: uuid.nullable(), publication: publicationViewSchema }).strict().refine(v => v.publication.status === "published" && v.publication.manifest.baseHead === v.previousHead);
const assetInput = z.object({ id, base64: text.max(1398104).regex(/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/) }).strict();
const draftInputFields = { catalogueVersion: version, package: packageSchema, assetBytes: z.array(assetInput).max(16), sourceMap };
export const draftInputSchema = z.object(draftInputFields).strict();
export const saveDraftInputSchema = z.object({ ...draftInputFields, expectedRevision: positive }).strict();
const inputSchemas = { createDraft: draftInputSchema, saveDraft: saveDraftInputSchema, adoptDraft: z.object({ packageId: id, packageVersion: version, reason }).strict(), validateDraft: z.object({ expectedRevision: positive }).strict(), submitDraft: z.object({ expectedRevision: positive, expectedDigest: sha }).strict(), reviseSubmission: z.object({}).strict(), decideReview: z.object({ decision: z.enum(["approve", "return"]), checks: reviewChecksSchema, independenceNote: note, note: reason }).strict(), prepareRelease: z.object({ submissionIds: z.array(uuid).min(1).max(20).refine(unique), expectedHead: uuid.nullable(), reason }).strict(), activateRelease: z.object({ expectedHead: uuid.nullable(), expectedManifestSha: sha, reason }).strict(), previewWithdrawal: z.object({ target: withdrawalTargetSchema }).strict(), withdrawVersion: z.object({ target: withdrawalTargetSchema, expectedHead: uuid.nullable(), reason }).strict() };
export const contentInputLimit = (kind: ContentEndpoint) => kind === "createDraft" || kind === "saveDraft" ? 8 << 20 : 8192;
export const contentIdempotent = (kind: ContentEndpoint) => Object.hasOwn(inputSchemas, kind) && kind !== "validateDraft" && kind !== "previewWithdrawal";
// Go json.Marshal escapes HTML and JavaScript separator code points.
export const contentCanonicalJSON = (value: unknown) => JSON.stringify(value).replace(/[<>&\u2028\u2029]/g, c => "\\u" + c.charCodeAt(0).toString(16).padStart(4, "0"));
const canonicalSize = (value: unknown) => new TextEncoder().encode(contentCanonicalJSON(value)).byteLength;
export function contentInputError(kind: ContentEndpoint, value: unknown): ContentErrorCode | null {
    if (!Object.hasOwn(inputSchemas, kind))
        return "INVALID_REQUEST";
    // Count limits are business limits, independently of DTO shape.
    if ((kind === "createDraft" || kind === "saveDraft") && value && typeof value === "object") {
        const v = value as Record<string, unknown>, p = v.package as Record<string, unknown> | undefined;
        if (p && typeof p === "object" && [[p.knowledge, 100], [p.units, 200], [p.paths, 20], [p.assets, 16], [v.assetBytes, 16]].some(([items, max]) => Array.isArray(items) && items.length > Number(max)))
            return "CONTENT_LIMIT_EXCEEDED";
        if (Array.isArray(v.assetBytes) && v.assetBytes.some(a => a && typeof a === "object" && typeof a.base64 === "string" && a.base64.length > 1398104))
            return "CONTENT_LIMIT_EXCEEDED";
    }
    if (!inputSchemas[kind as keyof typeof inputSchemas].safeParse(value).success)
        return "INVALID_REQUEST";
    if (kind === "createDraft" || kind === "saveDraft") {
        const v = value as z.infer<typeof draftInputSchema>;
        if (canonicalSize(v.package) > 2 << 20 || canonicalSize(v.sourceMap) > 256 << 10 || canonicalSize(value) > 8 << 20)
            return "CONTENT_LIMIT_EXCEEDED";
        if (!unique(v.assetBytes.map(a => a.id)))
            return "INVALID_REQUEST";
        if (v.assetBytes.length !== v.package.assets.length || v.assetBytes.some(a => !v.package.assets.some(m => m.id === a.id)))
            return "CONTENT_INVALID";
        if (v.sourceMap.some(link => !v.package.knowledge.some(k => k.id === link.knowledge.id && k.version === link.knowledge.version)))
            return "INVALID_REQUEST";
        let total = 0;
        for (const a of v.assetBytes) {
            try {
                const bytes = atob(a.base64);
                if (btoa(bytes) !== a.base64)
                    return "INVALID_REQUEST";
                if (bytes.length > 1 << 20)
                    return "CONTENT_LIMIT_EXCEEDED";
                total += bytes.length;
            }
            catch {
                return "INVALID_REQUEST";
            }
        }
        if (total > 4 << 20)
            return "CONTENT_LIMIT_EXCEEDED";
    }
    else if (new TextEncoder().encode(JSON.stringify(value)).byteLength > contentInputLimit(kind))
        return "PAYLOAD_TOO_LARGE";
    return null;
}
export const validContentInput = (kind: ContentEndpoint, value: unknown) => contentInputError(kind, value) === null;
export const contentPolicies: Record<ContentErrorCode, {
    status: number;
    message: string;
}> = {
    INVALID_REQUEST: { status: 400, message: "Invalid request." }, INVALID_COOKIE: { status: 400, message: "Invalid sign-in cookie." }, AUTHENTICATION_REQUIRED: { status: 401, message: "Please sign in to continue." }, CSRF_FAILED: { status: 403, message: "Request verification failed." }, FORBIDDEN: { status: 403, message: "You do not have permission." }, PASSWORD_CHANGE_REQUIRED: { status: 403, message: "Change your password to continue." }, NOT_FOUND: { status: 404, message: "Resource not found." }, METHOD_NOT_ALLOWED: { status: 405, message: "Method not allowed." }, DRAFT_CONFLICT: { status: 409, message: "This draft has changed. Reload it before continuing." }, REVIEW_CONFLICT: { status: 409, message: "This submission already has a review decision." }, IMMUTABLE_CONFLICT: { status: 409, message: "This version conflicts with saved content." }, VERSION_CONFLICT: { status: 409, message: "This version conflicts with saved content." }, IDEMPOTENCY_CONFLICT: { status: 409, message: "This request key was already used for different input." }, PUBLICATION_STALE: { status: 409, message: "Published content has changed. Prepare a new snapshot." }, PAYLOAD_TOO_LARGE: { status: 413, message: "This content exceeds the request size limit." }, CONTENT_NOT_READY: { status: 422, message: "Complete the required content before submitting." }, CONTENT_INVALID: { status: 422, message: "Content validation failed." }, REVIEW_REQUIRED: { status: 422, message: "Independent review is required before publication." }, CONTENT_LIMIT_EXCEEDED: { status: 422, message: "Split this content into smaller reviewed batches." }, REAUTHENTICATION_REQUIRED: { status: 428, message: "Verify your password before continuing." }, RATE_LIMITED: { status: 429, message: "Too many requests. Try again later." }, SERVICE_UNAVAILABLE: { status: 503, message: "Service temporarily unavailable." }, AUTH_NOT_CONFIGURED: { status: 503, message: "Accounts are temporarily unavailable." }, CONTENT_NOT_CONFIGURED: { status: 503, message: "Content management is temporarily unavailable." }
};
export const contentFailure = (code: ContentErrorCode = "SERVICE_UNAVAILABLE", requestId = "unavailable"): Extract<ContentResult<never>, {
    ok: false;
}> => ({ ok: false, code, ...contentPolicies[code], requestId });
const requestIdPattern = /^(?:[0-9a-f]{32}|unavailable)$/;
const errorSchema = z.object({ error: z.object({ code: z.enum(Object.keys(contentPolicies) as [
            ContentErrorCode,
            ...ContentErrorCode[]
        ]), message: text, requestId: text.regex(requestIdPattern) }).strict() }).strict().refine(v => v.error.message === contentPolicies[v.error.code].message);
const outputSchemas = { listDrafts: draftPageSchema, createDraft: draftViewSchema, readDraft: draftViewSchema, saveDraft: draftViewSchema, adoptDraft: draftViewSchema, validateDraft: gateSchema, submitDraft: submissionViewSchema, listSubmissions: submissionPageSchema, readSubmission: submissionViewSchema, reviseSubmission: draftViewSchema, decideReview: submissionViewSchema, listPublications: publicationPageSchema, readPublication: publicationViewSchema, prepareRelease: publicationViewSchema, activateRelease: publicationViewSchema, previewWithdrawal: withdrawalPreviewSchema, withdrawVersion: withdrawalResultSchema };
export const contentSuccessStatus = (kind: ContentEndpoint) => ["createDraft", "adoptDraft", "submitDraft", "reviseSubmission", "prepareRelease", "withdrawVersion"].includes(kind) ? 201 : 200;
export function contentResponseHeaders(response: Response): string {
    if (response.headers.has("Set-Cookie") || response.redirected || response.status >= 300 && response.status < 400)
        throw new Error("Invalid content response.");
    const id = response.headers.get("X-Request-ID");
    if (id === null || !requestIdPattern.test(id))
        throw new Error("Invalid content response.");
    return id;
}
export async function readContentResponse(response: Response, kind: ContentEndpoint, signal: AbortSignal): Promise<{
    result: ContentResult<unknown>;
    payload: unknown;
}> {
    const requestId = contentResponseHeaders(response);
    if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(response.headers.get("Content-Type") ?? "")) {
        void response.body?.cancel().catch(() => { });
        throw new Error("Invalid content response.");
    }
    const bytes = await readContentBytes(response, 4 << 20, signal), raw = parseContentJSON(bytes);
    if (!response.ok) {
        const error = errorSchema.parse(raw).error;
        if (contentPolicies[error.code].status !== response.status || error.requestId !== requestId)
            throw new Error("Invalid content response.");
        const result = contentFailure(error.code, requestId);
        if (error.code === "RATE_LIMITED") {
            const retry = response.headers.get("Retry-After");
            if (!retry || !/^(?:[1-9]|[1-5][0-9]|60)$/.test(retry))
                throw new Error("Invalid content response.");
            result.retryAfter = Number(retry);
        }
        else if (response.headers.has("Retry-After"))
            throw new Error("Invalid content response.");
        return { result, payload: raw };
    }
    if (response.status !== contentSuccessStatus(kind) || response.headers.has("Retry-After") || !(Object.hasOwn(outputSchemas, kind)))
        throw new Error("Invalid content response.");
    const parsed = outputSchemas[kind as keyof typeof outputSchemas].parse(raw);
    return { result: { ok: true, data: parsed }, payload: parsed };
}
export function normalizeContentQuery(kind: ContentEndpoint, query: ContentListQuery = {}): ContentListQuery | null {
    if (query === null || typeof query !== "object" || Object.keys(query).some(k => !["scope", "status", "limit", "offset"].includes(k)))
        return null;
    const scopes: Record<string, string[]> = { listDrafts: ["mine", "all"], listSubmissions: ["mine", "review", "all"], listPublications: ["all"] };
    const statuses: Record<string, string[]> = { listDrafts: ["editing", "submitted"], listSubmissions: ["pending", "approved", "returned"], listPublications: ["draft", "published"] };
    if (!scopes[kind] || query.scope !== undefined && !scopes[kind].includes(query.scope) || query.status !== undefined && !statuses[kind].includes(query.status) || query.limit !== undefined && (!Number.isSafeInteger(query.limit) || query.limit < 1 || query.limit > 100) || query.offset !== undefined && (!Number.isSafeInteger(query.offset) || query.offset < 0 || query.offset > 100000))
        return null;
    return { ...query };
}
export function contentRouteRequest(route: ContentRoute): {
    path: string;
    method: string;
    kind: ContentEndpoint;
} | null {
    if (!route || typeof route !== "object")
        return null;
    const list = { listDrafts: "drafts", listSubmissions: "submissions", listPublications: "publications" };
    const plain = { createDraft: ["drafts", "POST"], adoptDraft: ["drafts/adopt", "POST"], prepareRelease: ["publications/prepare", "POST"], previewWithdrawal: ["withdrawals/preview", "POST"], withdrawVersion: ["withdrawals", "POST"] };
    const identified = { readDraft: ["drafts", "", "GET"], saveDraft: ["drafts", "", "PUT"], validateDraft: ["drafts", "/validate", "POST"], submitDraft: ["drafts", "/submit", "POST"], readSubmission: ["submissions", "", "GET"], reviseSubmission: ["submissions", "/revision", "POST"], decideReview: ["submissions", "/decision", "POST"], readPublication: ["publications", "", "GET"], activateRelease: ["publications", "/activate", "POST"] };
    const keys = Object.keys(route), kind = route.kind;
    const allowed = (names: string[]) => keys.every(k => names.includes(k));
    let suffix = "", method = "GET";
    if (Object.hasOwn(list, kind)) {
        if (!allowed(["kind", "query"]))
            return null;
        const q = normalizeContentQuery(kind, "query" in route ? route.query : undefined);
        if (!q)
            return null;
        const params = new URLSearchParams();
        for (const [key, value] of Object.entries(q)) {
            if (value !== undefined)
                params.set(key, String(value));
        }
        ;
        suffix = list[kind as keyof typeof list] + (params.size ? "?" + params : "");
    }
    else if (Object.hasOwn(plain, kind)) {
        if (!allowed(["kind"]))
            return null;
        [suffix, method] = plain[kind as keyof typeof plain];
    }
    else if (Object.hasOwn(identified, kind)) {
        if (!allowed(["kind", "id"]) || !("id" in route) || !contentUUID.test(route.id))
            return null;
        const [group, operation, verb] = identified[kind as keyof typeof identified];
        suffix = group + "/" + route.id + operation;
        method = verb;
    }
    else if (kind === "readDraftAsset" || kind === "readSubmissionAsset") {
        if (!allowed(["kind", "id", "sha"]) || !contentUUID.test(route.id) || !contentSHA.test(route.sha))
            return null;
        suffix = (kind === "readDraftAsset" ? "drafts" : "submissions") + "/" + route.id + "/assets/" + route.sha;
    }
    else
        return null;
    return { path: "/api/v1/content/" + suffix, method, kind };
}
