import { createPrivateProxy, privateProxyError, privateRoutePreflight } from "@/lib/api/private-proxy";
import { getGoOrigin } from "@/lib/api/server-config";
import { AuthNotConfiguredError, getAuthConfig } from "@/lib/auth/config";
export const dynamic = "force-dynamic";
async function handle(request: Request, context: {
    params: Promise<{
        segments: string[];
    }>;
}) {
    try {
        const segments = ["admin", ...(await context.params).segments];
        const preflight = privateRoutePreflight(request, segments);
        if (preflight)
            return preflight;
        const config = getAuthConfig();
        return await createPrivateProxy(getGoOrigin(), config)(request, segments);
    }
    catch (error) {
        return privateProxyError(error instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
    }
}
export { handle as GET, handle as POST, handle as PUT, handle as DELETE, handle as PATCH, handle as OPTIONS, handle as HEAD };
