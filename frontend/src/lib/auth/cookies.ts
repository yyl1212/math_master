import type { PrivateEndpoint } from "./types";
import { secretPattern } from "./schemas";
export const authCookieNames = (production: boolean) => production ? ["__Host-mm_session", "__Host-mm_preauth"] as const : ["mm_session_dev", "mm_preauth_dev"] as const;
// Keep selected raw pairs and duplicates: only Go decides whether an identity cookie is valid.
export function selectAuthCookies(raw: string, production: boolean, sessionOnly = false): string {
    const names = authCookieNames(production);
    return raw.split(";").map(part => part.replace(/^[ \t]+/, "")).filter(part => { const name = part.split("=", 1)[0].trim(); return name === names[0] || !sessionOnly && name === names[1]; }).join("; ");
}
type ValidCookie = {
    name: string;
    clear: boolean;
    raw: string;
};
function parseCookie(raw: string, production: boolean): ValidCookie {
    const names = authCookieNames(production), parts = raw.split(";"), first = parts.shift()!;
    const equal = first.indexOf("=");
    if (equal < 0)
        throw new Error("Invalid private cookie.");
    const name = first.slice(0, equal), value = first.slice(equal + 1);
    if (name !== names[0] && name !== names[1])
        throw new Error("Unexpected private cookie.");
    const attributes = new Map<string, string | undefined>();
    for (const part of parts) {
        const trimmed = part.trim();
        if (!trimmed)
            throw new Error("Empty cookie attribute.");
        const split = trimmed.indexOf("=");
        const key = (split < 0 ? trimmed : trimmed.slice(0, split)).toLowerCase();
        const attribute = split < 0 ? undefined : trimmed.slice(split + 1);
        if (attributes.has(key) || !["path", "max-age", "httponly", "samesite", "secure", "expires"].includes(key))
            throw new Error("Invalid cookie attributes.");
        attributes.set(key, attribute);
    }
    if (attributes.get("path") !== "/" || !attributes.has("httponly") || attributes.get("httponly") !== undefined || attributes.get("samesite")?.toLowerCase() !== "lax" || attributes.has("secure") !== production || attributes.get("secure") !== undefined)
        throw new Error("Unsafe cookie attributes.");
    const expires = attributes.get("expires");
    if (attributes.has("expires") && (!expires || !/^\w{3}, \d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2} GMT$/.test(expires) || !Number.isFinite(Date.parse(expires))))
        throw new Error("Invalid cookie expiry.");
    const clear = value === "";
    if (clear ? attributes.get("max-age") !== "0" : !secretPattern.test(value) || attributes.get("max-age") !== (name === names[0] ? "604800" : "600"))
        throw new Error("Invalid cookie value.");
    return { name, clear, raw };
}
export function validatedSetCookies(headers: Headers, production: boolean, kind: PrivateEndpoint, status: number, errorCode?: string): string[] {
    const cookies = headers.getSetCookie().map(raw => parseCookie(raw, production));
    if (new Set(cookies.map(c => c.name)).size !== cookies.length)
        throw new Error("Duplicate private cookies.");
    if (cookies.length === 0) {
        if (status === 200 && kind === "login" || status === 201 && kind === "register" || status === 204 && ["logout", "logout-all", "password"].includes(kind))
            throw new Error("Missing identity cookie transition.");
        return [];
    }
    const [session, preauth] = authCookieNames(production);
    const clears = (name: string) => cookies.some(c => c.name === name && c.clear);
    if (status === 400 && errorCode === "INVALID_COOKIE" && kind !== "session" && cookies.every(c => c.clear))
        return cookies.map(c => c.raw);
    if (status === 200 && kind === "context" && cookies.every(c => c.name === preauth && !c.clear || c.name === session && c.clear))
        return cookies.map(c => c.raw);
    if (status === 201 && kind === "register" && clears(preauth) && cookies.every(c => c.clear))
        return cookies.map(c => c.raw);
    if (status === 200 && kind === "login" && cookies.length === 2 && cookies.some(c => c.name === session && !c.clear) && clears(preauth))
        return cookies.map(c => c.raw);
    if (status === 204 && ["logout", "logout-all", "password", "roles", "reset"].includes(kind) && cookies.length === 2 && clears(session) && clears(preauth))
        return cookies.map(c => c.raw);
    throw new Error("Unexpected identity cookie transition.");
}
