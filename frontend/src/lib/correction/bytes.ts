import { parseContentJSON } from '../content/raw-json';
import { readContentBytes } from '../content/bytes';
import { correctionInputSchemas } from './schemas';
import { CorrectionRequestError } from './types';
export function correctionAwait<T>(pending: Promise<T>, signal: AbortSignal): Promise<T> { return new Promise((resolve, reject) => { const abort = () => reject(new CorrectionRequestError()); const settled = () => signal.removeEventListener('abort', abort); void pending.then(v => { settled(); if (!signal.aborted)
    resolve(v); }, e => { settled(); reject(e); }); if (signal.aborted) {
    abort();
    return;
} signal.addEventListener('abort', abort, { once: true }); }); }
export async function withCorrectionDeadline<T>(signal: AbortSignal | undefined, work: (active: AbortSignal) => Promise<T>): Promise<T> { const c = new AbortController(), abort = () => c.abort(), timer = setTimeout(abort, 10000); signal?.addEventListener('abort', abort, { once: true }); if (signal?.aborted)
    abort(); try {
    if (c.signal.aborted)
        throw new CorrectionRequestError();
    return await correctionAwait(work(c.signal), c.signal);
}
catch (e) {
    throw e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
}
finally {
    clearTimeout(timer);
    signal?.removeEventListener('abort', abort);
} }
export function readCorrectionBytes(response: Response, max: number, signal: AbortSignal): Promise<Uint8Array> { return correctionAwait(readContentBytes(response, max, signal), signal); }
export function validateCorrectionBytes(raw: Uint8Array, action: string): unknown { try {
    if (raw.byteLength > 65536)
        throw new Error();
    const value = parseContentJSON(raw);
    switch (action) {
        case 'createCase': return correctionInputSchemas.createCase.parse(value);
        case 'createPlan': return correctionInputSchemas.createPlan.parse(value);
        case 'updatePlan': return correctionInputSchemas.updatePlan.parse(value);
        case 'submitPlan': return correctionInputSchemas.submitPlan.parse(value);
        case 'decidePlan': return correctionInputSchemas.decidePlan.parse(value);
        case 'retryJob': return correctionInputSchemas.retryJob.parse(value);
        default: throw new Error();
    }
}
catch {
    throw new CorrectionRequestError('INVALID_REQUEST');
} }
