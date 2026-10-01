import "server-only";
import { parseGoOrigin } from "./server-config";
import { parseAuthOrigin } from "../auth/config";
import { selectAuthCookies, validatedSetCookies } from "../auth/cookies";
import { failure, normalizeUserQuery, readPrivateBytes, readPrivateResponse, routeRequest, userIdPattern } from "../auth/schemas";
import type { AuthContext, PrivateEndpoint, PrivateErrorCode } from "../auth/types";
const headers = () => new Headers({ "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff", "Content-Type": "application/json" });
export function privateProxyError(code: PrivateErrorCode = "SERVICE_UNAVAILABLE"): Response { const result = failure(code); return Response.json({ error: { code: result.code, message: result.message, requestId: "unavailable" } }, { status: result.status, headers: headers() }); }
function parseRoute(segments: string[], query: URLSearchParams): {
    kind: PrivateEndpoint;
    path: string;
    method: string;
} | PrivateErrorCode {
    const [group, kind, id, operation] = segments;
    let target: {
        kind: PrivateEndpoint;
        path: string;
        method: string;
    } | null = null;
    if (group === "auth" && segments.length === 2) {
        if (kind === "session" || kind === "context")
            target = { kind, path: "/api/v1/auth/" + kind, method: "GET" };
        else if (["register", "login", "logout", "logout-all", "password", "reauth"].includes(kind))
            target = routeRequest({ kind: kind as "register" | "login" | "logout" | "logout-all" | "password" | "reauth" });
    }
    if (group === "admin" && kind === "users" && segments.length === 2) {
        for (const key of query.keys())
            if (!["q", "limit", "offset"].includes(key) || query.getAll(key).length !== 1)
                return "INVALID_REQUEST";
        for (const key of ["limit", "offset"])
            if (query.has(key) && !/^\d+$/.test(query.get(key)!))
                return "INVALID_REQUEST";
        const normalized = normalizeUserQuery({ q: query.get("q") ?? "", limit: query.has("limit") ? Number(query.get("limit")) : 20, offset: query.has("offset") ? Number(query.get("offset")) : 0 });
        if (!normalized)
            return "INVALID_REQUEST";
        target = routeRequest({ kind: "users", query: normalized });
    }
    if (group === "admin" && kind === "users" && segments.length === 4 && ["roles", "password-reset"].includes(operation)) {
        if (!userIdPattern.test(id))
            return "INVALID_REQUEST";
        target = routeRequest({ kind: operation === "roles" ? "roles" : "reset", userId: id });
    }
    if (!target)
        return "NOT_FOUND";
    if (target.kind !== "users" && query.size !== 0)
        return "INVALID_REQUEST";
    return target;
}
function requestRoute(request: Request, segments: string[]): ReturnType<typeof parseRoute> {
    const url = new URL(request.url);
    const base = parseRoute(segments, new URLSearchParams());
    if (typeof base === "string")
        return base;
    if (url.pathname !== base.path.split("?")[0])
        return "NOT_FOUND";
    if (request.method !== base.method)
        return base;
    try {
        decodeURIComponent(url.search.replace(/\+/g, " "));
    }
    catch {
        return "INVALID_REQUEST";
    }
    if (url.href.endsWith("?"))
        return "INVALID_REQUEST";
    return parseRoute(segments, url.searchParams);
}
export function privateRoutePreflight(request: Request, segments: string[]): Response | null {
    const route = requestRoute(request, segments);
    if (typeof route === "string")
        return privateProxyError(route);
    if (request.method !== route.method) {
        const response = privateProxyError("METHOD_NOT_ALLOWED");
        response.headers.set("Allow", route.method);
        return response;
    }
    return null;
}
export function createPrivateProxy(rawGoOrigin: string, options: {
    publicOrigin: string;
    production: boolean;
}, fetcher: typeof fetch = fetch) {
    const origin = parseGoOrigin(rawGoOrigin), publicOrigin = parseAuthOrigin(options.publicOrigin, options.production);
    return async (request: Request, segments: string[]): Promise<Response> => {
        const preflight = privateRoutePreflight(request, segments);
        if (preflight)
            return preflight;
        const route = requestRoute(request, segments);
        if (typeof route === "string")
            return privateProxyError(route);
        const write = route.method !== "GET";
        if ((write || route.kind === "context") && (request.headers.get("Sec-Fetch-Site") === "cross-site" || write && request.headers.get("Origin") !== publicOrigin || !write && request.headers.has("Origin") && request.headers.get("Origin") !== publicOrigin))
            return privateProxyError("CSRF_FAILED");
        const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 5000);
        try {
            const selected = new Headers({ Accept: "application/json" });
            const cookie = selectAuthCookies(request.headers.get("Cookie") ?? "", options.production);
            if (cookie)
                selected.set("Cookie", cookie);
            for (const key of ["Origin", "X-CSRF-Token", "X-Requested-With", "Sec-Fetch-Site"])
                if (request.headers.has(key))
                    selected.set(key, request.headers.get(key)!);
            let body: ArrayBuffer | undefined;
            if (write) {
                if (request.headers.get("Content-Type")?.split(";")[0].trim().toLowerCase() !== "application/json")
                    return privateProxyError("INVALID_REQUEST");
                selected.set("Content-Type", "application/json");
                try {
                    const bytes = await readPrivateBytes(new Response(request.body, { headers: request.headers.has("Content-Length") ? { "Content-Length": request.headers.get("Content-Length")! } : {} }), 8192, controller.signal);
                    body = bytes.buffer as ArrayBuffer;
                }
                catch {
                    return privateProxyError("INVALID_REQUEST");
                }
            }
            const upstream = await fetcher(origin + route.path, { method: route.method, headers: selected, ...(body ? { body } : {}), cache: "no-store", redirect: "error", signal: controller.signal });
            const parsed = await readPrivateResponse(upstream, route.kind, controller.signal);
            const responseHeaders = headers();
            const cookies = validatedSetCookies(upstream.headers, options.production, route.kind, upstream.status, parsed.result.ok ? undefined : parsed.result.code);
            if (route.kind === "context" && parsed.result.ok && (parsed.result.data as AuthContext).user !== null && cookies.length !== 0)
                throw new Error("Context cookie contradicts identity.");
            for (const cookie of cookies)
                responseHeaders.append("Set-Cookie", cookie);
            const requestId = upstream.headers.get("X-Request-ID");
            if (requestId !== null) {
                if (!/^(?:[0-9a-f]{32}|unavailable)$/.test(requestId))
                    throw new Error("Invalid private request ID.");
                responseHeaders.set("X-Request-ID", requestId);
            }
            if (!parsed.result.ok && parsed.result.retryAfter !== undefined)
                responseHeaders.set("Retry-After", String(parsed.result.retryAfter));
            return upstream.status === 204 ? new Response(null, { status: 204, headers: responseHeaders }) : Response.json(parsed.payload, { status: upstream.status, headers: responseHeaders });
        }
        catch {
            return privateProxyError();
        }
        finally {
            clearTimeout(timer);
        }
    };
}
