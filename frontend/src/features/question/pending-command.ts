import { requestQuestion } from "@/lib/question/client";
import { questionRouteRequest, questionInputError, questionFailure } from "@/lib/question/schemas";
import type { QuestionRoute, QuestionResult } from "@/lib/question/types";
export type QuestionPendingCommand = {
    readonly key: string;
    readonly route: QuestionRoute;
    readonly input: unknown;
    readonly status: "pending" | "running" | "done";
    execute: () => Promise<QuestionResult<unknown>>;
    cancel: () => void;
};
function freeze<T>(value: T): T { if (value && typeof value === "object") {
    Object.values(value).forEach(freeze);
    Object.freeze(value);
} return value; }
export function createQuestionPendingCommand(route: QuestionRoute, input: unknown): QuestionPendingCommand {
    const target = questionRouteRequest(route);
    if (!target || target.method === "GET" || questionInputError(route.kind, input))
        throw new Error("Invalid question command.");
    const fixedRoute = freeze(structuredClone(route)), fixedInput = freeze(structuredClone(input)), key = crypto.randomUUID();
    let status: QuestionPendingCommand["status"] = "pending", running: Promise<QuestionResult<unknown>> | null = null, result: QuestionResult<unknown> | null = null, controller: AbortController | null = null;
    return { key, route: fixedRoute, input: fixedInput, get status() { return status; }, cancel() { controller?.abort(); }, execute() { if (status === "done")
            return Promise.resolve(result!); if (running)
            return running; status = "running"; controller = new AbortController(); running = requestQuestion(fixedRoute, fixedInput, key, controller.signal).catch(() => questionFailure()).then(value => { result = value; status = value.ok ? "done" : "pending"; return value; }).finally(() => { running = null; controller = null; }); return running; } };
}
