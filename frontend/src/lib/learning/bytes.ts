import { parseContentJSON } from "../content/raw-json";
import { learningInputError } from "./schemas";
import type { LearningAction, LearningErrorCode } from "./types";
export class LearningInputError extends Error {
    constructor(public readonly code: LearningErrorCode) { super("Invalid learning input."); }
}
export function validateLearningBytes(raw: Uint8Array, action: LearningAction): unknown { if (raw.byteLength > 8192)
    throw new LearningInputError("PAYLOAD_TOO_LARGE"); let value: unknown; try {
    value = parseContentJSON(raw);
}
catch {
    throw new LearningInputError("INVALID_REQUEST");
} ; const error = learningInputError(action, value); if (error)
    throw new LearningInputError(error); return value; }
export function learningAwait<T>(pending: Promise<T>, signal: AbortSignal): Promise<T> { return new Promise((resolve, reject) => { const abort = () => reject(new Error("Learning request expired.")); if (signal.aborted) {
    abort();
    return;
} ; signal.addEventListener("abort", abort, { once: true }); void pending.then(resolve, reject).finally(() => signal.removeEventListener("abort", abort)); }); }
