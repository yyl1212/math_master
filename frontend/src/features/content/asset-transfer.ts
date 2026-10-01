import { readContentAsset } from "@/lib/content/client";
import { contentInputError, parseContentJSON, contentSHA } from "@/lib/content/schemas";
import { validSVGMarkup } from "@/lib/content/svg-markup";
import type { DraftInput, AssetScope, AssetInput } from "@/lib/content/types";
export const toBase64 = (bytes: Uint8Array) => { let raw = ""; for (let i = 0; i < bytes.length; i += 8192)
    raw += String.fromCharCode(...bytes.subarray(i, i + 8192)); return btoa(raw); };
export const fromBase64 = (text: string) => Uint8Array.from(atob(text), c => c.charCodeAt(0));
export async function svgDigest(bytes: Uint8Array): Promise<string> { if (!validSVGMarkup(bytes))
    throw new Error("Choose a valid, original SVG illustration."); const raw = await crypto.subtle.digest("SHA-256", bytes.slice().buffer as ArrayBuffer); return [...new Uint8Array(raw)].map(b => b.toString(16).padStart(2, "0")).join(""); }
export function importDraftInput(bytes: Uint8Array): DraftInput {
    if (bytes.byteLength > 8 << 20)
        throw new Error("This content exceeds the request size limit.");
    const value = parseContentJSON(bytes);
    const error = contentInputError("createDraft", value);
    if (error)
        throw new Error("The file must contain a valid DraftInput envelope.");
    const input = value as DraftInput;
    for (const a of input.assetBytes)
        if (!validSVGMarkup(fromBase64(a.base64)))
            throw new Error("Choose a valid, original SVG illustration.");
    return input;
}
export async function exportDraftInput(input: Omit<DraftInput, "assetBytes">, scope: AssetScope, local: AssetInput[] = []): Promise<string> {
    const assetBytes: AssetInput[] = [];
    for (const a of input.package.assets) {
        const existing = local.find(v => v.id === a.id);
        if (existing) {
            const bytes = fromBase64(existing.base64);
            if (await svgDigest(bytes) !== a.sha256)
                throw new Error("Illustration digest does not match.");
            assetBytes.push({ ...existing });
            continue;
        }
        if (!contentSHA.test(a.sha256))
            throw new Error("Illustration digest does not match.");
        const result = await readContentAsset(scope, a.sha256);
        if (!result.ok)
            throw new Error(result.message);
        assetBytes.push({ id: a.id, base64: toBase64(result.data) });
    }
    const envelope: DraftInput = { ...input, assetBytes };
    if (contentInputError("createDraft", envelope))
        throw new Error("Complete the content fields before exporting.");
    return JSON.stringify(envelope, null, 2);
}
