import { getAuthContext } from "../auth/client";
import { questionFailure, questionPolicies, questionRouteRequest, questionInputError, questionIdempotent, questionUUID, readQuestionResponse } from "./schemas";
import type { QuestionResult, QuestionRoute, QuestionErrorCode } from "./types";
function untilAbort<T>(pending: Promise<T>, signal: AbortSignal): Promise<T> {
    return new Promise((resolve, reject) => {
        const abort = () => reject(new Error("Question request expired."));
        if (signal.aborted) {
            abort();
            return;
        }
        signal.addEventListener("abort", abort, { once: true });
        void pending.then(resolve, reject).finally(() => signal.removeEventListener("abort", abort));
    });
}
export async function requestQuestion<T>(route: QuestionRoute, input?: unknown, pendingKey?: string, signal?: AbortSignal): Promise<QuestionResult<T>> {
    const target = questionRouteRequest(route);
    if (!target)
        return questionFailure("INVALID_REQUEST");
    const write = target.method !== "GET";
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000), abort = () => controller.abort();
    signal?.addEventListener("abort", abort, { once: true });
    try {
        if (signal?.aborted)
            return questionFailure();
        const h = new Headers({ Accept: "application/json" });
        let body: string | undefined;
        if (write) {
            const e = questionInputError(route.kind, input);
            if (e)
                return questionFailure(e);
            if (questionIdempotent(route.kind)) {
                const key = pendingKey ?? crypto.randomUUID();
                if (!questionUUID.test(key))
                    return questionFailure("INVALID_REQUEST");
                h.set("Idempotency-Key", key);
            }
            else if (pendingKey !== undefined && !questionUUID.test(pendingKey))
                return questionFailure("INVALID_REQUEST");
            body = JSON.stringify(input);
            const context = await untilAbort(getAuthContext(), controller.signal);
            if (!context.ok) {
                if (!Object.hasOwn(questionPolicies, context.code))
                    return questionFailure();
                const failure = questionFailure(context.code as QuestionErrorCode);
                if (context.retryAfter !== undefined)
                    failure.retryAfter = context.retryAfter;
                return failure;
            }
            if (context.data.user === null)
                return questionFailure("AUTHENTICATION_REQUIRED");
            if (context.data.user.mustChangePassword)
                return questionFailure("PASSWORD_CHANGE_REQUIRED");
            h.set("Content-Type", "application/json");
            h.set("X-CSRF-Token", context.data.csrfToken);
        }
        else if (input !== undefined || pendingKey !== undefined)
            return questionFailure("INVALID_REQUEST");
        if (controller.signal.aborted)
            return questionFailure();
        const response = await fetch(target.path, { method: target.method, headers: h, ...(body ? { body } : {}), credentials: "same-origin", cache: "no-store", redirect: "error", signal: controller.signal });
        return await readQuestionResponse(response, route.kind, controller.signal) as QuestionResult<T>;
    }
    catch {
        return questionFailure();
    }
    finally {
        clearTimeout(timer);
        signal?.removeEventListener("abort", abort);
    }
}
