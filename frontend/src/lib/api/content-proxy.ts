import { readContentBytes, ContentByteLimitError } from "../content/bytes";
import "server-only";
import { parseGoOrigin } from "./server-config";
import { parseAuthOrigin } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { secretPattern } from "../auth/schemas";
import { contentFailure, contentRouteRequest, normalizeContentQuery, contentInputLimit, contentIdempotent, contentInputError, parseContentJSON, readContentResponse, contentResponseHeaders, contentUUID, contentSHA } from "../content/schemas";
import { validateContentSVG } from "../content/svg";
import type { ContentErrorCode, ContentRoute, ContentListQuery } from "../content/types";
const privateHeaders = () => new Headers({ "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "Content-Type": "application/json", "X-Request-ID": "unavailable" });
export function contentProxyError(code: ContentErrorCode = "SERVICE_UNAVAILABLE"): Response { const failure = contentFailure(code); return Response.json({ error: { code: failure.code, message: failure.message, requestId: failure.requestId } }, { status: failure.status, headers: privateHeaders() }); }
function resolveRoute(request: Request, segments: string[]): ContentRoute | ContentErrorCode {
    if (segments.length === 0 || segments.some(s => s === ""))
        return "NOT_FOUND";
    const [group, id, operation, sha] = segments;
    let route: ContentRoute;
    if (segments.length === 1) {
        if (group === "drafts")
            route = { kind: request.method === "POST" ? "createDraft" : "listDrafts" };
        else if (group === "submissions")
            route = { kind: "listSubmissions" };
        else if (group === "publications")
            route = { kind: "listPublications" };
        else if (group === "withdrawals")
            route = { kind: "withdrawVersion" };
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
        if (!contentUUID.test(id))
            return "INVALID_REQUEST";
        route = { kind: group === "drafts" ? (request.method === "PUT" ? "saveDraft" : "readDraft") : group === "submissions" ? "readSubmission" : "readPublication", id };
    }
    else if (segments.length === 3 && group === "drafts" && ["validate", "submit"].includes(operation)) {
        if (!contentUUID.test(id))
            return "INVALID_REQUEST";
        route = { kind: operation === "validate" ? "validateDraft" : "submitDraft", id };
    }
    else if (segments.length === 3 && group === "submissions" && ["revision", "decision"].includes(operation)) {
        if (!contentUUID.test(id))
            return "INVALID_REQUEST";
        route = { kind: operation === "revision" ? "reviseSubmission" : "decideReview", id };
    }
    else if (segments.length === 3 && group === "publications" && operation === "activate") {
        if (!contentUUID.test(id))
            return "INVALID_REQUEST";
        route = { kind: "activateRelease", id };
    }
    else if (segments.length === 4 && ["drafts", "submissions"].includes(group) && operation === "assets") {
        if (!contentUUID.test(id) || !contentSHA.test(sha))
            return "INVALID_REQUEST";
        route = { kind: group === "drafts" ? "readDraftAsset" : "readSubmissionAsset", id, sha };
    }
    else
        return "NOT_FOUND";
    const url = new URL(request.url), target = contentRouteRequest(route);
    if (!target)
        return "INVALID_REQUEST";
    if (url.pathname !== target.path.split("?")[0])
        return "INVALID_REQUEST";
    if (request.method !== target.method)
        return route;
    if (url.href.endsWith("?") || url.search.includes("%") || url.search.includes("+"))
        return "INVALID_REQUEST";
    if (route.kind.startsWith("list")) {
        const query: Record<string, string | number> = {};
        for (const [key, value] of url.searchParams) {
            if (!["scope", "status", "limit", "offset"].includes(key) || key in query)
                return "INVALID_REQUEST";
            if (key === "limit" || key === "offset") {
                if (!/^[0-9]+$/.test(value))
                    return "INVALID_REQUEST";
                query[key] = Number(value);
            }
            else
                query[key] = value;
        }
        const normalized = normalizeContentQuery(route.kind, query as ContentListQuery);
        if (!normalized)
            return "INVALID_REQUEST";
        return { ...route, query: normalized } as ContentRoute;
    }
    if (url.search !== "")
        return "INVALID_REQUEST";
    return route;
}
export function contentRoutePreflight(request: Request, segments: string[]): Response | null {
    const route = resolveRoute(request, segments);
    if (typeof route === "string")
        return contentProxyError(route);
    const target = contentRouteRequest(route)!;
    if (request.method !== target.method) {
        const response = contentProxyError("METHOD_NOT_ALLOWED");
        response.headers.set("Allow", segments.length === 1 && segments[0] === "drafts" ? "GET, POST" : segments.length === 2 && segments[0] === "drafts" && contentUUID.test(segments[1]) ? "GET, PUT" : target.method);
        return response;
    }
    ;
    return null;
}
export function createContentProxy(rawGoOrigin: string, options: {
    publicOrigin: string;
    production: boolean;
}, fetcher: typeof fetch = fetch) {
    const origin = parseGoOrigin(rawGoOrigin), publicOrigin = parseAuthOrigin(options.publicOrigin, options.production);
    return async (request: Request, segments: string[]): Promise<Response> => {
        const preflight = contentRoutePreflight(request, segments);
        if (preflight)
            return preflight;
        const route = resolveRoute(request, segments);
        if (typeof route === "string")
            return contentProxyError(route);
        const target = contentRouteRequest(route)!;
        const write = target.method !== "GET", asset = route.kind === "readDraftAsset" || route.kind === "readSubmissionAsset";
        if (request.headers.get("Sec-Fetch-Site") === "cross-site" || write && request.headers.get("Origin") !== publicOrigin || !write && request.headers.has("Origin") && request.headers.get("Origin") !== publicOrigin || write && !secretPattern.test(request.headers.get("X-CSRF-Token") ?? ""))
            return contentProxyError("CSRF_FAILED");
        if (write && contentIdempotent(route.kind) && !contentUUID.test(request.headers.get("Idempotency-Key") ?? ""))
            return contentProxyError("INVALID_REQUEST");
        const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000), abort = () => controller.abort();
        request.signal.addEventListener("abort", abort, { once: true });
        try {
            if (request.signal.aborted)
                return contentProxyError();
            const selected = new Headers({ Accept: asset ? "image/svg+xml" : "application/json" });
            const cookie = selectAuthCookies(request.headers.get("Cookie") ?? "", options.production, true);
            if (cookie)
                selected.set("Cookie", cookie);
            for (const key of ["Origin", "X-CSRF-Token", "Sec-Fetch-Site"]) {
                const value = request.headers.get(key);
                if (value !== null)
                    selected.set(key, value);
            }
            let body: ArrayBuffer | undefined;
            if (write) {
                if (!/^application\/json(?:;\s*charset=(?:utf-8|"utf-8"))?$/i.test(request.headers.get("Content-Type") ?? ""))
                    return contentProxyError("INVALID_REQUEST");
                selected.set("Content-Type", "application/json");
                if (contentIdempotent(route.kind))
                    selected.set("Idempotency-Key", request.headers.get("Idempotency-Key")!);
                let bytes: Uint8Array;
                try {
                    bytes = await readContentBytes(new Response(request.body, { headers: request.headers.has("Content-Length") ? { "Content-Length": request.headers.get("Content-Length")! } : {} }), contentInputLimit(route.kind), controller.signal);
                }
                catch (error) {
                    if (controller.signal.aborted)
                        return contentProxyError();
                    return contentProxyError(error instanceof ContentByteLimitError ? "PAYLOAD_TOO_LARGE" : "INVALID_REQUEST");
                }
                try {
                    const error = contentInputError(route.kind, parseContentJSON(bytes));
                    if (error)
                        return contentProxyError(error);
                }
                catch {
                    return contentProxyError("INVALID_REQUEST");
                }
                body = bytes.slice().buffer as ArrayBuffer;
            }
            else if (request.body !== null)
                return contentProxyError("INVALID_REQUEST");
            const response = await fetcher(origin + target.path, { method: target.method, headers: selected, ...(body ? { body } : {}), cache: "no-store", redirect: "error", signal: controller.signal });
            if (asset && response.status === 200) {
                const requestId = contentResponseHeaders(response);
                if (response.headers.get("Content-Type") !== "image/svg+xml" || response.headers.get("Content-Security-Policy") !== "sandbox; default-src 'none'" || response.headers.get("X-Content-Type-Options") !== "nosniff" || response.headers.get("Cache-Control") !== "private, no-store" || response.headers.has("Retry-After"))
                    throw new Error("Invalid content asset response.");
                const bytes = await readContentBytes(response, 1 << 20, controller.signal);
                if (!validateContentSVG(bytes, route.sha))
                    throw new Error("Invalid content asset response.");
                const h = privateHeaders();
                h.set("Content-Type", "image/svg+xml");
                h.set("Content-Security-Policy", "sandbox; default-src 'none'");
                h.set("X-Request-ID", requestId);
                h.set("Content-Length", String(bytes.byteLength));
                return new Response(bytes.slice().buffer as ArrayBuffer, { status: 200, headers: h });
            }
            const parsed = await readContentResponse(response, route.kind, controller.signal), h = privateHeaders();
            h.set("X-Request-ID", response.headers.get("X-Request-ID")!);
            if (!parsed.result.ok && parsed.result.retryAfter !== undefined)
                h.set("Retry-After", String(parsed.result.retryAfter));
            return Response.json(parsed.payload, { status: response.status, headers: h });
        }
        catch {
            return contentProxyError();
        }
        finally {
            clearTimeout(timer);
            request.signal.removeEventListener("abort", abort);
        }
    };
}
