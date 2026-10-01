// Dedicated content bounds keep malformed streams distinct from size limits.
export class ContentByteLimitError extends Error {
}
export async function readContentBytes(response: Response, max: number, signal: AbortSignal): Promise<Uint8Array> {
    const length = response.headers.get("Content-Length");
    const cancelBody = () => { void response.body?.cancel().catch(() => { }); };
    if (length !== null && (!/^[0-9]+$/.test(length) || !Number.isSafeInteger(Number(length)))) {
        cancelBody();
        throw new Error("Invalid content length.");
    }
    if (length !== null && Number(length) > max) {
        cancelBody();
        throw new ContentByteLimitError("Content exceeds byte limit.");
    }
    const reader = response.body?.getReader();
    if (!reader) {
        if (length !== null && Number(length) !== 0)
            throw new Error("Invalid content length.");
        return new Uint8Array();
    }
    const cancel = () => { void reader.cancel().catch(() => { }); };
    signal.addEventListener("abort", cancel, { once: true });
    let size = 0;
    const chunks: Uint8Array[] = [];
    try {
        if (signal.aborted)
            throw new Error("Content request expired.");
        for (;;) {
            const { done, value } = await reader.read();
            if (signal.aborted)
                throw new Error("Content request expired.");
            if (done)
                break;
            size += value.byteLength;
            if (size > max)
                throw new ContentByteLimitError("Content exceeds byte limit.");
            chunks.push(value);
        }
        if (length !== null && Number(length) !== size)
            throw new Error("Invalid content length.");
        const out = new Uint8Array(size);
        let offset = 0;
        for (const chunk of chunks) {
            out.set(chunk, offset);
            offset += chunk.byteLength;
        }
        return out;
    }
    catch (error) {
        cancel();
        throw error;
    }
    finally {
        signal.removeEventListener("abort", cancel);
        reader.releaseLock();
    }
}
