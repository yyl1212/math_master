import { getAuthContext } from "../auth/client";
import { learningFailure, learningPolicies, learningRouteRequest, readLearningResponse, learningUUID } from "./schemas";
import { learningAwait, validateLearningBytes, LearningInputError } from "./bytes";
import type { LearningResult, LearningRoute, LearningErrorCode } from "./types";
const actorBindings = new WeakMap<object, string>();
// A frozen pending input belongs to its original browser actor, never a later login.
export function bindLearningInput(input: object, actorId: string): void { actorBindings.set(input, actorId); }
export async function requestLearning<T>(route: LearningRoute, input?: unknown, pendingKey?: string, signal?: AbortSignal): Promise<LearningResult<T>> {
    const target = learningRouteRequest(route);
    if (!target)
        return learningFailure("INVALID_REQUEST");
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000), abort = () => controller.abort();
    signal?.addEventListener("abort", abort, { once: true });
    try {
        if (signal?.aborted)
            return learningFailure();
        const headers = new Headers({ Accept: route.kind === "readAsset" ? "image/svg+xml" : "application/json" });
        let body: string | undefined;
        if (target.method === "POST") {
            const key = pendingKey ?? crypto.randomUUID();
            if (!learningUUID.test(key))
                return learningFailure("INVALID_REQUEST");
            body = JSON.stringify(input);
            validateLearningBytes(new TextEncoder().encode(body), route.kind);
            const context = await learningAwait(getAuthContext(), controller.signal);
            if (!context.ok) {
                const code = Object.hasOwn(learningPolicies, context.code) ? context.code as LearningErrorCode : "SERVICE_UNAVAILABLE";
                return { ...learningFailure(code), ...(context.retryAfter === undefined ? {} : { retryAfter: context.retryAfter }) };
            }
            ;
            if (context.data.user === null)
                return learningFailure("AUTHENTICATION_REQUIRED");
            if (context.data.user.mustChangePassword)
                return learningFailure("PASSWORD_CHANGE_REQUIRED");
            const actor = input !== null && typeof input === "object" ? actorBindings.get(input) : undefined;
            if (actor !== undefined && actor !== context.data.user.id)
                return learningFailure("FORBIDDEN");
            headers.set("Content-Type", "application/json");
            headers.set("X-CSRF-Token", context.data.csrfToken);
            headers.set("Idempotency-Key", key);
        }
        else if (input !== undefined || pendingKey !== undefined)
            return learningFailure("INVALID_REQUEST");
        const response = await learningAwait(fetch(target.path, { method: target.method, headers, ...(body === undefined ? {} : { body }), credentials: "same-origin", cache: "no-store", redirect: "error", signal: controller.signal }), controller.signal);
        return await readLearningResponse(response, route, controller.signal) as LearningResult<T>;
    }
    catch (e) {
        return learningFailure(e instanceof LearningInputError ? e.code : "SERVICE_UNAVAILABLE");
    }
    finally {
        clearTimeout(timer);
        signal?.removeEventListener("abort", abort);
    }
}
