import "server-only";
import { getGoOrigin } from "../api/server-config";
import { getAuthConfig, AuthNotConfiguredError } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { readContentBytes } from "./bytes";
import { contentFailure, contentRouteRequest, readContentResponse, contentResponseHeaders } from "./schemas";
import { validateContentSVG } from "./svg";
import type { ContentRoute, ContentResult } from "./types";
export async function readServerContent<T>(route: ContentRoute, cookieHeader: string): Promise<ContentResult<T>> {
    const target = contentRouteRequest(route);
    if (!target || target.method !== "GET")
        return contentFailure("INVALID_REQUEST");
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000);
    try {
        const config = getAuthConfig(), origin = getGoOrigin(), cookie = selectAuthCookies(cookieHeader, config.production, true), asset = route.kind === "readDraftAsset" || route.kind === "readSubmissionAsset";
        const response = await fetch(origin + target.path, { method: "GET", cache: "no-store", redirect: "error", signal: controller.signal, headers: { Accept: asset ? "image/svg+xml" : "application/json", ...(cookie ? { Cookie: cookie } : {}) } });
        if (asset && response.status === 200) {
            contentResponseHeaders(response);
            if (response.headers.get("Content-Type") !== "image/svg+xml")
                throw new Error("Invalid content asset response.");
            const bytes = await readContentBytes(response, 1 << 20, controller.signal);
            if (!validateContentSVG(bytes, route.sha))
                throw new Error("Invalid content asset response.");
            return { ok: true, data: bytes as T };
        }
        return (await readContentResponse(response, route.kind, controller.signal)).result as ContentResult<T>;
    }
    catch (error) {
        return contentFailure(error instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
    }
    finally {
        clearTimeout(timer);
    }
}
