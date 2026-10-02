import "server-only";
import { cookies } from "next/headers";
import { getGoOrigin } from "../api/server-config";
import { getAuthConfig, AuthNotConfiguredError } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { validateContentSVG } from "../content/svg";
import { learningAwait } from "./bytes";
import { learningFailure, learningRouteRequest, readLearningResponse } from "./schemas";
import type { LearningReadClient, LearningResult, LearningRoute } from "./types";
async function read<T>(route: LearningRoute): Promise<LearningResult<T>> { const target = learningRouteRequest(route); if (!target || target.method !== "GET")
    return learningFailure("INVALID_REQUEST"); const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000); try {
    const config = getAuthConfig(), origin = getGoOrigin(), jar = await learningAwait(cookies(), controller.signal), cookie = selectAuthCookies(jar.toString(), config.production, true);
    const response = await learningAwait(fetch(origin + target.path, { method: "GET", headers: { Accept: route.kind === "readAsset" ? "image/svg+xml" : "application/json", ...(cookie ? { Cookie: cookie } : {}) }, cache: "no-store", redirect: "error", signal: controller.signal }), controller.signal);
    return await readLearningResponse(response, route, controller.signal, validateContentSVG) as LearningResult<T>;
}
catch (e) {
    return learningFailure(e instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
}
finally {
    clearTimeout(timer);
} }
// No client or private response is stored globally; cookies are read for each call.
export function getLearningClient(): LearningReadClient {
    return {
        readLearningOverview: () => read({ kind: "readOverview" }), listLearningKnowledge: query => read({ kind: "listKnowledge", query }), readLearningKnowledge: (id, version) => read({ kind: "readKnowledge", id, version }), listLearningPaths: query => read({ kind: "listPaths", query }), readLearningPath: id => read({ kind: "readPath", id }), listLearningPathNodes: (id, query) => read({ kind: "listPathNodes", id, query }), readPractice: id => read({ kind: "readPractice", id }), readAssessment: id => read({ kind: "readAssessment", id }), readAssessmentResult: id => read({ kind: "readAssessmentResult", id }), listLearningHistory: query => read({ kind: "listHistory", query }), readLearningAsset: (id, sha) => read({ kind: "readAsset", id, sha })
    };
}
