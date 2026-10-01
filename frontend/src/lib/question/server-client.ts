import "server-only";
import { getGoOrigin } from "../api/server-config";
import { getAuthConfig, AuthNotConfiguredError } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { questionFailure, questionRouteRequest, readQuestionResponse } from "./schemas";
import type { QuestionResult, QuestionRoute } from "./types";
export async function readServerQuestion<T>(route: QuestionRoute, cookieHeader: string): Promise<QuestionResult<T>> {
    const target = questionRouteRequest(route);
    if (!target || target.method !== "GET")
        return questionFailure("INVALID_REQUEST");
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000);
    try {
        const config = getAuthConfig(), origin = getGoOrigin(), cookie = selectAuthCookies(cookieHeader, config.production, true);
        const response = await fetch(origin + target.path, { method: "GET", headers: { Accept: "application/json", ...(cookie ? { Cookie: cookie } : {}) }, cache: "no-store", redirect: "error", signal: controller.signal });
        return await readQuestionResponse(response, route.kind, controller.signal) as QuestionResult<T>;
    }
    catch (e) {
        return questionFailure(e instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
    }
    finally {
        clearTimeout(timer);
    }
}
