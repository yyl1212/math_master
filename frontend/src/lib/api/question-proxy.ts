import "server-only";
import { parseGoOrigin } from "./server-config";
import { parseAuthOrigin } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { secretPattern } from "../auth/schemas";
import { readContentBytes, ContentByteLimitError } from "../content/bytes";
import { questionFailure, questionRouteRequest, questionHasQuery, normalizeQuestionQuery, questionUUID, questionInputLimit, questionIdempotent, questionInputError, parseQuestionJSON, readQuestionResponse } from "../question/schemas";
import type { QuestionRoute, QuestionErrorCode } from "../question/types";
const privateHeaders = () => new Headers({ "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "Content-Type": "application/json", "X-Request-ID": "unavailable" });
export function questionProxyError(code: QuestionErrorCode = "SERVICE_UNAVAILABLE"): Response { const e = questionFailure(code); return Response.json({ error: { code: e.code, message: e.message, requestId: e.requestId } }, { status: e.status, headers: privateHeaders() }); }
function resolve(request: Request, segments: string[]): QuestionRoute | QuestionErrorCode {
    if (!segments.length || segments.some(s => s === ""))
        return "NOT_FOUND";
    const [group, id, op] = segments;
    let route: QuestionRoute;
    if (segments.length === 1) {
        if (group === "drafts")
            route = { kind: request.method === "POST" ? "createDraft" : "listDrafts" };
        else if (group === "submissions")
            route = { kind: "listSubmissions" };
        else if (group === "publications")
            route = { kind: "listPublications" };
        else if (group === "withdrawals")
            route = { kind: "withdrawVersion" };
        else if (group === "coverage")
            route = { kind: "readCoverage" };
        else
            return "NOT_FOUND";
    }
    else if (segments.length === 2 && group === "drafts" && id === "adopt")
        route = { kind: "adoptDraft" };
    else if (segments.length === 2 && group === "publications" && id === "prepare")
        route = { kind: "prepareRelease" };
    else if (segments.length === 2 && group === "withdrawals" && id === "preview")
        route = { kind: "previewWithdrawal" };
    else if (segments.length === 2 && ["drafts", "submissions", "publications"].includes(group)) {
        if (!questionUUID.test(id))
            return "INVALID_REQUEST";
        route = { kind: group === "drafts" ? (request.method === "PUT" ? "saveDraft" : "readDraft") : group === "submissions" ? "readSubmission" : "readPublication", id };
    }
    else if (segments.length === 3 && group === "drafts" && ["validate", "submit"].includes(op)) {
        route = { kind: op === "validate" ? "validateDraft" : "submitDraft", id };
    }
    else if (segments.length === 3 && group === "submissions" && ["instances", "revision", "decision"].includes(op)) {
        route = { kind: op === "instances" ? "listInstances" : op === "revision" ? "reviseSubmission" : "decideReview", id };
    }
    else if (segments.length === 3 && group === "publications" && ["members", "changes", "activate"].includes(op)) {
        route = { kind: op === "members" ? "listMembers" : op === "changes" ? "listChanges" : "activateRelease", id };
    }
    else
        return "NOT_FOUND";
    const target = questionRouteRequest(route), url = new URL(request.url);
    if (!target)
        return "INVALID_REQUEST";
    if (url.pathname !== target.path.split("?")[0])
        return "INVALID_REQUEST";
    if (request.method !== target.method)
        return route;
    if (url.href.endsWith("?") || url.search.includes("%") || url.search.includes("+"))
        return "INVALID_REQUEST";
    if (questionHasQuery(route.kind)) {
        const q: Record<string, string | number> = {};
        for (const [k, v] of url.searchParams) {
            if (k in q || v === "")
                return "INVALID_REQUEST";
            if (k === "limit" || k === "offset") {
                if (!/^(?:0|[1-9][0-9]*)$/.test(v))
                    return "INVALID_REQUEST";
                q[k] = Number(v);
            }
            else
                q[k] = v;
        }
        ;
        const normalized = normalizeQuestionQuery(route.kind, q as never);
        if (!normalized)
            return "INVALID_REQUEST";
        return { ...route, query: normalized } as QuestionRoute;
    }
    if (url.search !== "")
        return "INVALID_REQUEST";
    return route;
}
export function questionRoutePreflight(request: Request, segments: string[]): Response | null { const route = resolve(request, segments); if (typeof route === "string")
    return questionProxyError(route); const target = questionRouteRequest(route)!; if (request.method === target.method)
    return null; const response = questionProxyError("METHOD_NOT_ALLOWED"); response.headers.set("Allow", segments.length === 1 && segments[0] === "drafts" ? "GET, POST" : segments.length === 2 && segments[0] === "drafts" ? "GET, PUT" : target.method); return response; }
export function createQuestionProxy(rawGoOrigin: string, options: {
    publicOrigin: string;
    production: boolean;
}, fetcher: typeof fetch = fetch) {
    const origin = parseGoOrigin(rawGoOrigin), publicOrigin = parseAuthOrigin(options.publicOrigin, options.production);
    return async (request: Request, segments: string[]): Promise<Response> => {
        const preflight = questionRoutePreflight(request, segments);
        if (preflight)
            return preflight;
        const route = resolve(request, segments);
        if (typeof route === "string")
            return questionProxyError(route);
        const target = questionRouteRequest(route)!;
        const write = target.method !== "GET";
        if (request.headers.get("Sec-Fetch-Site") === "cross-site" || write && request.headers.get("Origin") !== publicOrigin || !write && request.headers.has("Origin") && request.headers.get("Origin") !== publicOrigin || write && !secretPattern.test(request.headers.get("X-CSRF-Token") ?? ""))
            return questionProxyError("CSRF_FAILED");
        if (request.headers.get("X-CSRF-Token")?.includes(","))
            return questionProxyError("CSRF_FAILED");
        if (request.headers.get("Idempotency-Key")?.includes(","))
            return questionProxyError("INVALID_REQUEST");
        if (write && questionIdempotent(route.kind) && !questionUUID.test(request.headers.get("Idempotency-Key") ?? ""))
            return questionProxyError("INVALID_REQUEST");
        const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000), abort = () => controller.abort();
        request.signal.addEventListener("abort", abort, { once: true });
        try {
            if (request.signal.aborted)
                return questionProxyError();
            const h = new Headers({ Accept: "application/json" });
            const cookie = selectAuthCookies(request.headers.get("Cookie") ?? "", options.production, true);
            if (cookie)
                h.set("Cookie", cookie);
            for (const key of ["Origin", "X-CSRF-Token", "Sec-Fetch-Site"]) {
                const v = request.headers.get(key);
                if (v !== null)
                    h.set(key, v);
            }
            let body: ArrayBuffer | undefined;
            if (write) {
                if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(request.headers.get("Content-Type") ?? ""))
                    return questionProxyError("INVALID_REQUEST");
                h.set("Content-Type", "application/json");
                if (questionIdempotent(route.kind))
                    h.set("Idempotency-Key", request.headers.get("Idempotency-Key")!);
                let raw: Uint8Array;
                try {
                    raw = await readContentBytes(new Response(request.body, { headers: request.headers.has("Content-Length") ? { "Content-Length": request.headers.get("Content-Length")! } : {} }), questionInputLimit(route.kind), controller.signal);
                }
                catch (e) {
                    return questionProxyError(controller.signal.aborted ? "SERVICE_UNAVAILABLE" : e instanceof ContentByteLimitError ? "PAYLOAD_TOO_LARGE" : "INVALID_REQUEST");
                }
                try {
                    const e = questionInputError(route.kind, parseQuestionJSON(raw));
                    if (e)
                        return questionProxyError(e);
                }
                catch {
                    return questionProxyError("INVALID_REQUEST");
                }
                ;
                body = raw.slice().buffer as ArrayBuffer;
            }
            else if (request.body !== null)
                return questionProxyError("INVALID_REQUEST");
            const response = await fetcher(origin + target.path, { method: target.method, headers: h, ...(body ? { body } : {}), cache: "no-store", redirect: "error", signal: controller.signal });
            const result = await readQuestionResponse(response, route.kind, controller.signal);
            const out = privateHeaders();
            if (result.ok) {
                out.set("X-Request-ID", response.headers.get("X-Request-ID")!);
                return Response.json(result.data, { status: response.status, headers: out });
            }
            out.set("X-Request-ID", result.requestId);
            if (result.retryAfter !== undefined)
                out.set("Retry-After", String(result.retryAfter));
            return Response.json({ error: { code: result.code, message: result.message, requestId: result.requestId } }, { status: result.status, headers: out });
        }
        catch {
            return questionProxyError();
        }
        finally {
            clearTimeout(timer);
            request.signal.removeEventListener("abort", abort);
        }
    };
}
