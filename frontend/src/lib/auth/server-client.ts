import "server-only";
import { getGoOrigin } from "../api/server-config";
import { AuthNotConfiguredError, getAuthConfig } from "./config";
import { selectAuthCookies } from "./cookies";
import { failure, readPrivateResponse, routeRequest } from "./schemas";
import type { AuthResult, PrivateEndpoint, User, UserPage, UserQuery } from "./types";
async function read<T>(path: string, kind: PrivateEndpoint, cookieHeader: string): Promise<AuthResult<T>> {
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 5000);
    try {
        const config = getAuthConfig(), origin = getGoOrigin();
        const cookie = selectAuthCookies(cookieHeader, config.production, true);
        const response = await fetch(origin + path, { method: "GET", cache: "no-store", redirect: "error", signal: controller.signal, headers: { Accept: "application/json", ...(cookie ? { Cookie: cookie } : {}) } });
        return (await readPrivateResponse(response, kind, controller.signal)).result as AuthResult<T>;
    }
    catch (error) {
        return failure(error instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
    }
    finally {
        clearTimeout(timer);
    }
}
export const readServerSession = (cookieHeader: string): Promise<AuthResult<User | null>> => read("/api/v1/auth/session", "session", cookieHeader);
export function readServerUsers(cookieHeader: string, query: UserQuery): Promise<AuthResult<UserPage>> {
    const target = routeRequest({ kind: "users", query });
    if (!target)
        return Promise.resolve(failure("INVALID_REQUEST"));
    return read(target.path, "users", cookieHeader);
}
