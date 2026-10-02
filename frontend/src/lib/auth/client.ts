import { failure, readPrivateResponse, routeRequest, validPrivateInput } from "./schemas";
import type { AuthContext, AuthResult, PrivateEndpoint, PrivateRoute } from "./types";
let inFlight: Promise<AuthResult<AuthContext>> | undefined;
async function requestPrivate<T>(path: string, kind: PrivateEndpoint, options: RequestInit): Promise<AuthResult<T>> {
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 5000);
    try {
        const response = await fetch(path, { ...options, credentials: "same-origin", cache: "no-store", redirect: "error", signal: controller.signal });
        return (await readPrivateResponse(response, kind, controller.signal)).result as AuthResult<T>;
    }
    catch {
        return failure();
    }
    finally {
        clearTimeout(timer);
    }
}
export function getAuthContext(fresh = false): Promise<AuthResult<AuthContext>> {
    if (inFlight && !fresh)
        return inFlight;
    const pending = requestPrivate<AuthContext>("/api/v1/auth/context", "context", { method: "GET", headers: { Accept: "application/json", "X-Requested-With": "MathMaster" } });
    inFlight = pending;
    void pending.finally(() => {
        if (inFlight === pending)
            inFlight = undefined;
    });
    return pending;
}
export async function authRequest<T>(route: PrivateRoute, input: unknown): Promise<AuthResult<T>> {
    const target = routeRequest(route);
    if (!target)
        return failure("INVALID_REQUEST");
    if (target.method === "GET")
        return requestPrivate<T>(target.path, target.kind, { method: "GET", headers: { Accept: "application/json" } });
    if (!validPrivateInput(target.kind, input))
        return failure("INVALID_REQUEST");
    const context = await getAuthContext();
    if (!context.ok)
        return context;
    return requestPrivate<T>(target.path, target.kind, { method: target.method, headers: { Accept: "application/json", "Content-Type": "application/json", "X-Requested-With": "MathMaster", "X-CSRF-Token": context.data.csrfToken }, body: JSON.stringify(input) });
}
export function notifyAuthChanged(): void {
    if (typeof window === "undefined") return;
    window.dispatchEvent(new Event("math-master:auth-change"));
    // Only an ephemeral invalidation signal crosses tabs, with no user or draft data.
    if (typeof BroadcastChannel !== "undefined") {
        try {
            const channel = new BroadcastChannel("math-master-auth");
            try { channel.postMessage("changed"); } finally { channel.close(); }
        } catch { /* Focus and command-time checks also verify identity. */ }
    }
}
