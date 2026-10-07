import "server-only";
import { cookies } from "next/headers";
import { getGoOrigin } from "../api/server-config";
import { getAuthConfig, AuthNotConfiguredError } from "../auth/config";
import { selectAuthCookies } from "../auth/cookies";
import { validateContentSVG } from "../content/svg";
import { learningAwait } from "./bytes";
import { learningFailure, learningRouteRequest, readLearningResponse } from "./schemas";
import type { LearningReadClient, LearningResult, LearningRoute } from "./types";
async function read<T>(route: LearningRoute,outerSignal?:AbortSignal): Promise<LearningResult<T>> { const target = learningRouteRequest(route); if (!target || target.method !== "GET")
    return learningFailure("INVALID_REQUEST"); const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 10000); const abort=()=>controller.abort();outerSignal?.addEventListener("abort",abort,{once:true});try {
    if(outerSignal?.aborted)return learningFailure();
    const config = getAuthConfig(), origin = getGoOrigin(), jar = await learningAwait(cookies(), controller.signal), cookie = selectAuthCookies(jar.toString(), config.production, true);
    const response = await learningAwait(fetch(origin + target.path, { method: "GET", headers: { Accept: route.kind === "readAsset" ? "image/svg+xml" : "application/json", ...(cookie ? { Cookie: cookie } : {}) }, cache: "no-store", redirect: "error", signal: controller.signal }), controller.signal);
    return await readLearningResponse(response, route, controller.signal, validateContentSVG) as LearningResult<T>;
}
catch (e) {
    return learningFailure(e instanceof AuthNotConfiguredError ? "AUTH_NOT_CONFIGURED" : "SERVICE_UNAVAILABLE");
}
finally {
    clearTimeout(timer);outerSignal?.removeEventListener("abort",abort);
} }
// No client or private response is stored globally; cookies are read for each call.
export function getLearningClient(signal?:AbortSignal): LearningReadClient {
    return {
        readLearningOverview: () => read({ kind: "readOverview" },signal), listLearningKnowledge: query => read({ kind: "listKnowledge", query },signal), readLearningKnowledge: (id, version) => read({ kind: "readKnowledge", id, version },signal), listLearningPaths: query => read({ kind: "listPaths", query },signal), readLearningPath: id => read({ kind: "readPath", id },signal), listLearningPathNodes: (id, query) => read({ kind: "listPathNodes", id, query },signal), readPractice: id => read({ kind: "readPractice", id },signal), readAssessment: id => read({ kind: "readAssessment", id },signal), readAssessmentResult: id => read({ kind: "readAssessmentResult", id },signal), listLearningHistory: query => read({ kind: "listHistory", query },signal), readLearningAsset: (id, sha) => read({ kind: "readAsset", id, sha },signal)
    };
}
