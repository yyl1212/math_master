import { createLearningProxy, learningProxyError, learningRoutePreflight } from "@/lib/api/learning-proxy";
import { getGoOrigin } from "@/lib/api/server-config";
import { getAuthConfig, AuthNotConfiguredError } from "@/lib/auth/config";
export const dynamic = "force-dynamic";
async function handle(request: Request) { try {
    const error = learningRoutePreflight(request);
    if (error)
        return error;
    return await createLearningProxy(getGoOrigin(), getAuthConfig())(request);
}
catch (e) {
    return learningProxyError(e instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
} }
export { handle as GET, handle as POST, handle as PUT, handle as DELETE, handle as PATCH, handle as OPTIONS, handle as HEAD };
