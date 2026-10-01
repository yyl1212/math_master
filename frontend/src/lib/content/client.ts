import { getAuthContext } from "../auth/client";
import { readContentBytes } from "./bytes";
import { contentFailure, contentPolicies, contentRouteRequest, contentIdempotent, contentUUID, contentSHA, contentInputError, readContentResponse, contentResponseHeaders } from "./schemas";
import type { AssetScope, ContentRoute, ContentResult, ContentErrorCode } from "./types";
export async function contentRequest<T>(route: ContentRoute, input?: unknown, key?: string): Promise<ContentResult<T>> {
    const target = contentRouteRequest(route);
    if (!target || target.kind === "readDraftAsset" || target.kind === "readSubmissionAsset")
        return contentFailure("INVALID_REQUEST");
    const write = target.method !== "GET", headers = new Headers({ Accept: "application/json" });
    let body: string | undefined;
    if (write) {
        const inputError = contentInputError(route.kind, input);
        if (inputError)
            return contentFailure(inputError);
        if (contentIdempotent(route.kind)) {
            const operationKey = key ?? crypto.randomUUID();
            if (!contentUUID.test(operationKey))
                return contentFailure("INVALID_REQUEST");
            headers.set("Idempotency-Key", operationKey);
        }
        body = JSON.stringify(input);
        const context = await getAuthContext();
        if (!context.ok) {
            if (!Object.hasOwn(contentPolicies, context.code))
                return contentFailure();
            const failure = contentFailure(context.code as ContentErrorCode);
            if (context.retryAfter !== undefined)
                failure.retryAfter = context.retryAfter;
            return failure;
        }
        ;
        if (context.data.user === null)
            return contentFailure("AUTHENTICATION_REQUIRED");
        if (context.data.user.mustChangePassword)
            return contentFailure("PASSWORD_CHANGE_REQUIRED");
        headers.set("Content-Type", "application/json");
        headers.set("X-CSRF-Token", context.data.csrfToken);
    }
    else if (input !== undefined || key !== undefined)
        return contentFailure("INVALID_REQUEST");
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000);
    try {
        const response = await fetch(target.path, { method: target.method, headers, ...(body ? { body } : {}), credentials: "same-origin", cache: "no-store", redirect: "error", signal: controller.signal });
        return (await readContentResponse(response, route.kind, controller.signal)).result as ContentResult<T>;
    }
    catch {
        return contentFailure();
    }
    finally {
        clearTimeout(timer);
    }
}
export async function readContentAsset(scope: AssetScope, sha: string): Promise<ContentResult<Uint8Array>> {
    if (!scope || !contentUUID.test(scope.id) || !contentSHA.test(sha) || scope.kind !== "draft" && scope.kind !== "submission")
        return contentFailure("INVALID_REQUEST");
    const route: ContentRoute = { kind: scope.kind === "draft" ? "readDraftAsset" : "readSubmissionAsset", id: scope.id, sha }, target = contentRouteRequest(route)!;
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000);
    try {
        const response = await fetch(target.path, { method: "GET", headers: { Accept: "image/svg+xml" }, credentials: "same-origin", cache: "no-store", redirect: "error", signal: controller.signal });
        if (response.status !== 200)
            return (await readContentResponse(response, route.kind, controller.signal)).result as ContentResult<Uint8Array>;
        contentResponseHeaders(response);
        if (response.headers.get("Content-Type") !== "image/svg+xml")
            throw new Error("Invalid content asset response.");
        const bytes = await readContentBytes(response, 1 << 20, controller.signal);
        const digest = await crypto.subtle.digest("SHA-256", bytes.slice().buffer as ArrayBuffer);
        const actual = [...new Uint8Array(digest)].map(b => b.toString(16).padStart(2, "0")).join("");
        if (actual !== sha)
            throw new Error("Invalid content asset digest.");
        return { ok: true, data: bytes };
    }
    catch {
        return contentFailure();
    }
    finally {
        clearTimeout(timer);
    }
}
