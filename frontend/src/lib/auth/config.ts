import "server-only";
import { isIP } from "node:net";
export class AuthNotConfiguredError extends Error {
    constructor() { super("Accounts are not configured."); }
}
export function parseAuthOrigin(raw: string, production: boolean): string {
    const fail = () => new Error("Invalid account configuration.");
    if (!/^https?:\/\/[^/?#]+$/.test(raw))
        throw fail();
    let url: URL;
    try {
        url = new URL(raw);
    }
    catch {
        throw fail();
    }
    if (url.username || url.password || url.search || url.hash || url.pathname !== "/" || production && url.protocol !== "https:")
        throw fail();
    const rawHost = raw.slice(raw.indexOf("://") + 3).replace(/:\d+$/, "").replace(/^\[|\]$/g, "");
    if (/[^\x21-\x7e]|[%\\]/.test(rawHost) || isIP(rawHost) === 0 && url.hostname.toLowerCase() !== rawHost.toLowerCase())
        throw fail();
    const hostname = url.hostname.replace(/^\[|\]$/g, "");
    const loopback = rawHost.toLowerCase() === "localhost" || isIP(rawHost) !== 0 && (hostname === "::1" || /^127\./.test(hostname) || /^::ffff:7f[0-9a-f]{2}:/.test(hostname));
    if (!production && url.protocol === "http:" && !loopback)
        throw fail();
    return url.origin;
}
export function getAuthConfig(): {
    publicOrigin: string;
    production: boolean;
} {
    const env = process.env.APP_ENV || "development";
    if (env !== "development" && env !== "production")
        throw new Error("Invalid account configuration.");
    const raw = process.env.AUTH_PUBLIC_ORIGIN;
    if (!raw)
        throw new AuthNotConfiguredError();
    const production = env === "production";
    return { publicOrigin: parseAuthOrigin(raw, production), production };
}
