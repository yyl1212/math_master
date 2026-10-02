import "server-only";
import { parseGoOrigin } from "./server-config";
import { parseAuthOrigin } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { secretPattern } from "../auth/schemas";
import { readContentBytes, ContentByteLimitError } from "../content/bytes";
import { validateContentSVG } from "../content/svg";
import { validateLearningBytes, LearningInputError, learningAwait } from "../learning/bytes";
import { learningFailure, learningRouteRequest, learningUUID, readLearningResponse } from "../learning/schemas";
import type { LearningRoute, LearningErrorCode } from "../learning/types";
const privateHeaders = () => new Headers({ "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "Content-Type": "application/json", "X-Request-ID": "unavailable" });
export function learningProxyError(code: LearningErrorCode = "SERVICE_UNAVAILABLE"): Response { const e = learningFailure(code); return Response.json({ error: { code: e.code, message: e.message, requestId: e.requestId } }, { status: e.status, headers: privateHeaders() }); }
export function resolveLearningRoute(request: Request): LearningRoute | LearningErrorCode {
    const url = new URL(request.url), base = "/api/v1/learning/";
    if (!url.pathname.startsWith(base))
        return "NOT_FOUND";
    if (url.pathname.includes("%") || url.href.endsWith("?") || url.search.includes("%") || url.search.includes("+"))
        return "INVALID_REQUEST";
    const p = url.pathname.slice(base.length).split("/");
    if (p.some(v => v === ""))
        return "NOT_FOUND";
    const [group, id, op] = p;
    let route: LearningRoute;
    if (p.length === 1) {
        switch (group) {
            case "overview":
                route = { kind: "readOverview" };
                break;
            case "knowledge":
                route = { kind: "listKnowledge" };
                break;
            case "paths":
                route = { kind: "listPaths" };
                break;
            case "history":
                route = { kind: "listHistory" };
                break;
            case "practice":
                route = { kind: "createPractice" };
                break;
            case "assessments":
                route = { kind: "createAssessment" };
                break;
            default: return "NOT_FOUND";
        }
    }
    else if (p.length === 2) {
        switch (group) {
            case "knowledge":
                route = { kind: "readKnowledge", id, version: 0 };
                break;
            case "paths":
                route = { kind: "readPath", id };
                break;
            case "practice":
                route = { kind: "readPractice", id };
                break;
            case "assessments":
                route = { kind: "readAssessment", id };
                break;
            default: return "NOT_FOUND";
        }
    }
    else if (p.length === 3) {
        if (group === "knowledge" && (op === "start" || op === "complete"))
            route = { kind: op === "start" ? "startKnowledge" : "completeKnowledge", id };
        else if (group === "paths" && op === "enroll")
            route = { kind: "enrollPath", id };
        else if (group === "paths" && op === "nodes")
            route = { kind: "listPathNodes", id };
        else if (group === "practice" && ["answer", "reveal", "abandon"].includes(op))
            route = { kind: op === "answer" ? "answerPractice" : op === "reveal" ? "revealPractice" : "abandonPractice", id };
        else if (group === "assessments" && ["submit", "abandon", "result"].includes(op))
            route = { kind: op === "submit" ? "submitAssessment" : op === "abandon" ? "abandonAssessment" : "readAssessmentResult", id };
        else if (group === "assets")
            route = { kind: "readAsset", id, sha: op };
        else
            return "NOT_FOUND";
    }
    else
        return "NOT_FOUND";
    const values: Record<string, number> = {};
    for (const [k, v] of url.searchParams) {
        if (Object.hasOwn(values, k) || !/^(?:0|[1-9][0-9]*)$/.test(v) || !Number.isSafeInteger(Number(v)))
            return "INVALID_REQUEST";
        values[k] = Number(v);
    }
    ;
    if (route.kind === "readKnowledge") {
        if (Object.keys(values).length !== 1 || !Object.hasOwn(values, "version"))
            return "INVALID_REQUEST";
        route.version = values.version;
    }
    else if (["listKnowledge", "listPaths", "listPathNodes", "listHistory"].includes(route.kind)) {
        if (Object.keys(values).some(k => k !== "limit" && k !== "offset"))
            return "INVALID_REQUEST";
        route = { ...route, query: values } as LearningRoute;
    }
    else if (Object.keys(values).length)
        return "INVALID_REQUEST";
    const target = learningRouteRequest(route);
    if (!target || url.pathname !== target.path.split("?")[0])
        return "INVALID_REQUEST";
    return route;
}
export function learningRoutePreflight(request: Request): Response | null { const route = resolveLearningRoute(request); if (typeof route === "string")
    return learningProxyError(route); const target = learningRouteRequest(route)!; if (request.method !== target.method) {
    const response = learningProxyError("METHOD_NOT_ALLOWED");
    response.headers.set("Allow", target.method);
    return response;
} ; return null; }
export function createLearningProxy(rawGoOrigin: string, config: {
    publicOrigin: string;
    production: boolean;
}, fetcher: typeof fetch = fetch): (request: Request) => Promise<Response> {
    const origin = parseGoOrigin(rawGoOrigin), publicOrigin = parseAuthOrigin(config.publicOrigin, config.production);
    return async (request) => {
        const preflight = learningRoutePreflight(request);
        if (preflight)
            return preflight;
        const route = resolveLearningRoute(request) as LearningRoute, target = learningRouteRequest(route)!, write = target.method === "POST";
        if (request.headers.get("Sec-Fetch-Site") === "cross-site" || write && request.headers.get("Origin") !== publicOrigin || !write && request.headers.has("Origin") && request.headers.get("Origin") !== publicOrigin || request.headers.get("X-CSRF-Token")?.includes(",") || write && !secretPattern.test(request.headers.get("X-CSRF-Token") ?? ""))
            return learningProxyError("CSRF_FAILED");
        if (request.headers.get("Idempotency-Key")?.includes(",") || write && !learningUUID.test(request.headers.get("Idempotency-Key") ?? ""))
            return learningProxyError("INVALID_REQUEST");
        const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000), abort = () => controller.abort();
        request.signal.addEventListener("abort", abort, { once: true });
        try {
            if (request.signal.aborted)
                return learningProxyError();
            const headers = new Headers({ Accept: route.kind === "readAsset" ? "image/svg+xml" : "application/json" }), cookie = selectAuthCookies(request.headers.get("Cookie") ?? "", config.production, true);
            if (cookie)
                headers.set("Cookie", cookie);
            for (const k of ["Origin", "X-CSRF-Token", "Sec-Fetch-Site"]) {
                const v = request.headers.get(k);
                if (v !== null)
                    headers.set(k, v);
            }
            ;
            let body: ArrayBuffer | undefined;
            if (write) {
                if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(request.headers.get("Content-Type") ?? ""))
                    return learningProxyError("INVALID_REQUEST");
                headers.set("Content-Type", "application/json");
                headers.set("Idempotency-Key", request.headers.get("Idempotency-Key")!);
                let raw: Uint8Array;
                try {
                    raw = await readContentBytes(new Response(request.body, { headers: request.headers.has("Content-Length") ? { "Content-Length": request.headers.get("Content-Length")! } : {} }), 8192, controller.signal);
                }
                catch (e) {
                    return learningProxyError(controller.signal.aborted ? "SERVICE_UNAVAILABLE" : e instanceof ContentByteLimitError ? "PAYLOAD_TOO_LARGE" : "INVALID_REQUEST");
                }
                ;
                validateLearningBytes(raw, route.kind);
                body = raw.slice().buffer as ArrayBuffer;
            }
            else if (request.body !== null)
                return learningProxyError("INVALID_REQUEST");
            const response = await learningAwait(fetcher(origin + target.path, { method: target.method, headers, ...(body === undefined ? {} : { body }), cache: "no-store", redirect: "error", signal: controller.signal }), controller.signal);
            const result = await readLearningResponse(response, route, controller.signal, validateContentSVG), out = privateHeaders();
            if (result.ok) {
                out.set("X-Request-ID", response.headers.get("X-Request-ID")!);
                if (route.kind === "readAsset") {
                    const bytes = result.data as Uint8Array;
                    out.set("Content-Type", "image/svg+xml");
                    out.set("Content-Length", String(bytes.byteLength));
                    out.set("Content-Security-Policy", "sandbox; default-src 'none'");
                    return new Response(bytes.slice().buffer as ArrayBuffer, { status: 200, headers: out });
                }
                ;
                return Response.json(result.data, { status: response.status, headers: out });
            }
            ;
            out.set("X-Request-ID", result.requestId);
            if (result.retryAfter !== undefined)
                out.set("Retry-After", String(result.retryAfter));
            const { ok: _ok, status, retryAfter: _retryAfter, ...error } = result;
            return Response.json({ error }, { status, headers: out });
        }
        catch (e) {
            return learningProxyError(controller.signal.aborted ? "SERVICE_UNAVAILABLE" : e instanceof LearningInputError ? e.code : e instanceof ContentByteLimitError ? "PAYLOAD_TOO_LARGE" : "SERVICE_UNAVAILABLE");
        }
        finally {
            clearTimeout(timer);
            request.signal.removeEventListener("abort", abort);
        }
    };
}
