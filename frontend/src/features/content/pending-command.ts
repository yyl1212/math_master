import { contentRequest } from "@/lib/content/client";
import { contentRouteRequest, contentInputError, contentFailure } from "@/lib/content/schemas";
import type { ContentRoute, ContentResult } from "@/lib/content/types";
export type PendingCommand = {
    readonly route: ContentRoute;
    readonly key: string;
    readonly input: unknown;
    readonly state: "pending" | "running" | "done";
    execute: () => Promise<ContentResult<unknown>>;
};
function freeze<T>(v: T): T { if (v && typeof v === "object") {
    Object.values(v).forEach(freeze);
    Object.freeze(v);
} return v; }
export function createPendingCommand(route: ContentRoute, input: unknown): PendingCommand {
    const target = contentRouteRequest(route);
    if (!target || target.method === "GET" || contentInputError(route.kind, input))
        throw new Error("Invalid content command.");
    const fixedRoute = freeze(JSON.parse(JSON.stringify(route))) as ContentRoute;
    let fixedInput: unknown = freeze(JSON.parse(JSON.stringify(input)));
    const key = crypto.randomUUID();
    let state: PendingCommand["state"] = "pending", running: Promise<ContentResult<unknown>> | null = null, result: ContentResult<unknown> | null = null;
    return { route: fixedRoute, key, get input() { return fixedInput; }, get state() { return state; }, execute() {
            if (state === "done")
                return Promise.resolve(result!);
            if (running)
                return running;
            state = "running";
            running = contentRequest(fixedRoute, fixedInput, key).catch(() => contentFailure()).then(value => { result = value; state = value.ok ? "done" : "pending"; if (value.ok)
                fixedInput = null; return value; }).finally(() => { running = null; });
            return running;
        } };
}
export const retryPendingCommand = (command: PendingCommand) => command.execute();
