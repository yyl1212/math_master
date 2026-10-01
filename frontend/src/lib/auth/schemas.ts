import { z } from "zod";
import type { AuthResult, PrivateEndpoint, PrivateErrorCode, PrivateRoute, UserQuery } from "./types";
export const secretPattern = /^[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]$/;
export const userIdPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
export const roleSchema = z.enum(["learner", "editor", "reviewer", "admin"]);
const roleOrder = ["learner", "editor", "reviewer", "admin"];
export const userSchema = z.object({ id: z.string().regex(userIdPattern), username: z.string().regex(/^[a-z0-9_]{3,32}$/), roles: z.array(roleSchema).min(1).max(4).refine(roles => roles[0] === "learner" && new Set(roles).size === roles.length && roles.every((r, i) => i === 0 || roleOrder.indexOf(roles[i - 1]) < roleOrder.indexOf(r))), mustChangePassword: z.boolean() }).strict();
export const authContextSchema = z.object({ user: userSchema.nullable(), csrfToken: z.string().regex(secretPattern) }).strict();
export const userPageSchema = z.object({ items: z.array(userSchema).max(100), total: z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER), limit: z.number().int().min(1).max(100), offset: z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER) }).strict().refine(page => page.items.length <= page.limit && page.items.length <= page.total);
export const errorPolicies: Record<PrivateErrorCode, {
    status: number;
    message: string;
}> = {
    INVALID_REQUEST: { status: 400, message: "Invalid request." }, INVALID_COOKIE: { status: 400, message: "Invalid sign-in cookie." }, INVALID_CREDENTIALS: { status: 401, message: "Invalid username or password." }, AUTHENTICATION_REQUIRED: { status: 401, message: "Please sign in to continue." }, CSRF_FAILED: { status: 403, message: "Request verification failed." }, FORBIDDEN: { status: 403, message: "You do not have permission." }, PASSWORD_CHANGE_REQUIRED: { status: 403, message: "Change your password to continue." }, NOT_FOUND: { status: 404, message: "Resource not found." }, METHOD_NOT_ALLOWED: { status: 405, message: "Method not allowed." }, USERNAME_UNAVAILABLE: { status: 409, message: "This username is unavailable." }, ALREADY_AUTHENTICATED: { status: 409, message: "Sign out before using another account." }, LAST_ADMIN_REQUIRED: { status: 409, message: "At least one administrator is required." }, REAUTHENTICATION_REQUIRED: { status: 428, message: "Verify your password before continuing." }, RATE_LIMITED: { status: 429, message: "Too many requests. Try again later." }, AUTH_NOT_CONFIGURED: { status: 503, message: "Accounts are temporarily unavailable." }, SERVICE_UNAVAILABLE: { status: 503, message: "Service temporarily unavailable." },
};
const errorCodeSchema = z.enum(["INVALID_REQUEST", "INVALID_COOKIE", "INVALID_CREDENTIALS", "AUTHENTICATION_REQUIRED", "CSRF_FAILED", "FORBIDDEN", "PASSWORD_CHANGE_REQUIRED", "NOT_FOUND", "METHOD_NOT_ALLOWED", "USERNAME_UNAVAILABLE", "ALREADY_AUTHENTICATED", "LAST_ADMIN_REQUIRED", "REAUTHENTICATION_REQUIRED", "RATE_LIMITED", "AUTH_NOT_CONFIGURED", "SERVICE_UNAVAILABLE"]);
export const privateErrorSchema = z.object({ error: z.object({ code: errorCodeSchema, message: z.string(), requestId: z.string().regex(/^(?:[0-9a-f]{32}|unavailable)$/) }).strict() }).strict().refine(body => body.error.message === errorPolicies[body.error.code].message);
export const failure = (code: PrivateErrorCode = "SERVICE_UNAVAILABLE"): Extract<AuthResult<never>, {
    ok: false;
}> => ({ ok: false, code, ...errorPolicies[code] });
function validUnicode(value: string): boolean { return [...value].every(c => { const n = c.codePointAt(0)!; return n < 0xd800 || n > 0xdfff; }); }
export const passwordSchema = z.string().refine(value => validUnicode(value) && [...value].length >= 15 && [...value].length <= 128 && new TextEncoder().encode(value).byteLength <= 512);
const noteSchema = z.string().refine(value => validUnicode(value) && value.trim().length > 0 && [...value].length >= 10 && [...value].length <= 1000);
const credentialsSchema = z.object({ username: z.string().regex(/^[A-Za-z0-9_]{3,32}$/), password: passwordSchema }).strict();
const inputSchemas = { register: credentialsSchema, login: credentialsSchema, logout: z.object({}).strict(), "logout-all": z.object({}).strict(), password: z.object({ currentPassword: passwordSchema, newPassword: passwordSchema }).strict(), reauth: z.object({ password: passwordSchema }).strict(), roles: z.object({ roles: z.array(roleSchema).min(1).refine(roles => roles.includes("learner")), reason: noteSchema }).strict(), reset: z.object({ temporaryPassword: passwordSchema, reason: noteSchema, ownershipNote: noteSchema }).strict() };
export function validPrivateInput(kind: PrivateEndpoint, input: unknown): boolean { return kind in inputSchemas && inputSchemas[kind as keyof typeof inputSchemas].safeParse(input).success; }
export function normalizeUserQuery(query: UserQuery): Required<UserQuery> | null {
    const { q = "", limit = 20, offset = 0 } = query;
    if (typeof q !== "string" || !validUnicode(q) || new TextEncoder().encode(q).byteLength > 128 || !Number.isSafeInteger(limit) || limit < 1 || limit > 100 || !Number.isSafeInteger(offset) || offset < 0)
        return null;
    return { q, limit, offset };
}
export function routeRequest(route: PrivateRoute): {
    kind: PrivateEndpoint;
    method: string;
    path: string;
} | null {
    switch (route.kind) {
        case "register":
        case "login":
        case "logout":
        case "logout-all":
        case "password":
        case "reauth": return { kind: route.kind, method: "POST", path: "/api/v1/auth/" + route.kind };
        case "roles":
        case "reset":
            if (!userIdPattern.test(route.userId))
                return null;
            return { kind: route.kind, method: route.kind === "roles" ? "PUT" : "POST", path: "/api/v1/admin/users/" + route.userId + "/" + (route.kind === "roles" ? "roles" : "password-reset") };
        case "users": {
            const query = normalizeUserQuery(route.query);
            if (!query)
                return null;
            return { kind: "users", method: "GET", path: "/api/v1/admin/users?" + new URLSearchParams({ q: query.q, limit: String(query.limit), offset: String(query.offset) }) };
        }
        default: return null;
    }
}
export async function readPrivateBytes(response: Response, max: number, signal: AbortSignal): Promise<Uint8Array> {
    const length = response.headers.get("Content-Length");
    if (length !== null && (!/^\d+$/.test(length) || !Number.isSafeInteger(Number(length)) || Number(length) > max)) {
        void response.body?.cancel().catch(() => { });
        throw new Error("Invalid private response.");
    }
    const reader = response.body?.getReader();
    if (!reader)
        return new Uint8Array();
    const cancel = () => { void reader.cancel().catch(() => { }); };
    signal.addEventListener("abort", cancel, { once: true });
    const chunks: Uint8Array[] = [];
    let size = 0;
    try {
        if (signal.aborted)
            throw new Error("Private request expired.");
        for (;;) {
            const { done, value } = await reader.read();
            if (signal.aborted)
                throw new Error("Private request expired.");
            if (done)
                break;
            size += value.byteLength;
            if (size > max) {
                cancel();
                throw new Error("Private body too large.");
            }
            chunks.push(value);
        }
        const out = new Uint8Array(size);
        let offset = 0;
        for (const chunk of chunks) {
            out.set(chunk, offset);
            offset += chunk.byteLength;
        }
        return out;
    }
    finally {
        signal.removeEventListener("abort", cancel);
        reader.releaseLock();
    }
}
const sessionSchema = z.object({ data: z.object({ user: userSchema.nullable() }).strict() }).strict();
const contextResponseSchema = z.object({ data: authContextSchema }).strict();
const userResponseSchema = z.object({ data: z.object({ user: userSchema }).strict() }).strict();
const reauthResponseSchema = z.object({ data: z.object({ validUntil: z.iso.datetime({ offset: true }) }).strict() }).strict();
const successStatus: Record<PrivateEndpoint, number> = { session: 200, context: 200, register: 201, login: 200, logout: 204, "logout-all": 204, password: 204, reauth: 200, users: 200, roles: 204, reset: 204 };
export async function readPrivateResponse(response: Response, kind: PrivateEndpoint, signal: AbortSignal): Promise<{
    result: AuthResult<unknown>;
    payload: unknown;
}> {
    if (response.status === 204) {
        const bytes = await readPrivateBytes(response, 0, signal);
        if (successStatus[kind] !== 204 || bytes.byteLength !== 0)
            throw new Error("Unexpected private status.");
        return { result: { ok: true, data: undefined }, payload: undefined };
    }
    if (response.headers.get("Content-Type")?.split(";")[0].trim().toLowerCase() !== "application/json") {
        void response.body?.cancel().catch(() => { });
        throw new Error("Invalid private content type.");
    }
    const bytes = await readPrivateBytes(response, 1024 * 1024, signal);
    const raw: unknown = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
    if (!response.ok) {
        const parsed = privateErrorSchema.safeParse(raw);
        if (!parsed.success || errorPolicies[parsed.data.error.code].status !== response.status)
            throw new Error("Invalid private error.");
        const { code, message } = parsed.data.error;
        const result: AuthResult<never> = { ok: false, status: response.status, code, message };
        if (code === "RATE_LIMITED") {
            const retry = response.headers.get("Retry-After");
            if (!retry || !/^\d+$/.test(retry) || !Number.isSafeInteger(Number(retry)) || Number(retry) < 1 || Number(retry) > 86400)
                throw new Error("Invalid private retry.");
            result.retryAfter = Number(retry);
        }
        return { result, payload: parsed.data };
    }
    if (response.status !== successStatus[kind])
        throw new Error("Unexpected private success status.");
    if (kind === "users") {
        const parsed = userPageSchema.parse(raw);
        return { result: { ok: true, data: parsed }, payload: parsed };
    }
    if (kind === "context") {
        const parsed = contextResponseSchema.parse(raw);
        return { result: { ok: true, data: parsed.data }, payload: parsed };
    }
    if (kind === "session") {
        const parsed = sessionSchema.parse(raw);
        return { result: { ok: true, data: parsed.data.user }, payload: parsed };
    }
    if (kind === "login" || kind === "register") {
        const parsed = userResponseSchema.parse(raw);
        return { result: { ok: true, data: parsed.data.user }, payload: parsed };
    }
    if (kind === "reauth") {
        const parsed = reauthResponseSchema.parse(raw);
        return { result: { ok: true, data: parsed.data }, payload: parsed };
    }
    throw new Error("Unexpected private body.");
}
