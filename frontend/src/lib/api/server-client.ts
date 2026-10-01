import "server-only";
import type { z } from "zod";
import { z as schema } from "zod";
import {
  domainListSchema,
  domainDetailSchema,
  pathViewSchema,
  knowledgeViewSchema,
  errorSchema,
} from "./schemas";
import { getGoOrigin, parseGoOrigin } from "./server-config";
import type { GoClient, ApiResult } from "./types";
export const JSON_MAX_BYTES = 10 * 1024 * 1024;
export const SVG_MAX_BYTES = 1024 * 1024;
export const validId = (id: string) => /^[a-z][a-z0-9-]{0,63}$/.test(id);
export function safeRequestId(value: unknown): string | undefined {
  return typeof value === "string" && /^[a-zA-Z0-9-]{1,64}$/.test(value)
    ? value
    : undefined;
}
export async function readBoundedBytes(
  response: Response,
  max: number,
  signal: AbortSignal,
): Promise<Uint8Array> {
  const length = response.headers.get("Content-Length");
  if (length && /^\d+$/.test(length) && Number(length) > max) {
    void response.body?.cancel();
    throw new Error("Response too large.");
  }
  const reader = response.body?.getReader();
  if (!reader) return new Uint8Array();
  const cancel = () => {
    void reader.cancel().catch(() => {});
  };
  signal.addEventListener("abort", cancel, { once: true });
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    if (signal.aborted) throw new Error("Request expired.");
    while (true) {
      const { done, value } = await reader.read();
      if (signal.aborted) throw new Error("Request expired.");
      if (done) break;
      size += value.byteLength;
      if (size > max) {
        cancel();
        throw new Error("Response too large.");
      }
      chunks.push(value);
    }
    const out = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
      out.set(chunk, offset);
      offset += chunk.byteLength;
    }
    return out;
  } finally {
    signal.removeEventListener("abort", cancel);
    try {
      reader.releaseLock();
    } catch {}
  }
}
export function createGoClient(
  rawOrigin: string,
  fetcher: typeof fetch = fetch,
): GoClient {
  const origin = parseGoOrigin(rawOrigin);
  async function read<T>(
    path: string,
    validator: z.ZodType<T>,
  ): Promise<ApiResult<T>> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 5000);
    let response: Response | undefined;
    try {
      response = await fetcher(origin + path, {
        cache: "no-store",
        redirect: "error",
        signal: controller.signal,
        headers: { Accept: "application/json" },
      });
      if (
        response.headers.get("Content-Type")?.split(";")[0] !==
        "application/json"
      ) {
        void response.body?.cancel();
        return { ok: false, kind: "unavailable" };
      }
      const raw = JSON.parse(
        new TextDecoder("utf-8", { fatal: true }).decode(
          await readBoundedBytes(response, JSON_MAX_BYTES, controller.signal),
        ),
      );
      if (response.status !== 200) {
        const parsed = errorSchema.safeParse(raw);
        const requestId = parsed.success
          ? safeRequestId(parsed.data.error.requestId)
          : undefined;
        const kind =
          response.status === 404
            ? "not-found"
            : response.status === 400
              ? "invalid-query"
              : "unavailable";
        return { ok: false, kind, ...(requestId ? { requestId } : {}) };
      }
      const parsed = validator.safeParse(raw);
      return parsed.success
        ? { ok: true, data: parsed.data }
        : { ok: false, kind: "unavailable" };
    } catch {
      return { ok: false, kind: "unavailable" };
    } finally {
      clearTimeout(timer);
    }
  }
  const detail = <T>(
    group: string,
    id: string,
    v: z.ZodType<T>,
  ): Promise<ApiResult<T>> =>
    validId(id)
      ? read("/api/v1/" + group + "/" + id, schema.object({ data: v })).then(
          (r) => (r.ok ? { ok: true, data: r.data.data } : r),
        )
      : Promise.resolve({ ok: false, kind: "not-found" });
  return {
    listDomains: ({ q, limit, offset }) =>
      read(
        "/api/v1/domains?" +
          new URLSearchParams({
            q,
            limit: String(limit),
            offset: String(offset),
          }),
        domainListSchema,
      ),
    getDomain: (id) => detail("domains", id, domainDetailSchema),
    getPath: (id) => detail("paths", id, pathViewSchema),
    getKnowledge: (id) => detail("knowledge", id, knowledgeViewSchema),
  };
}
export function getGoClient(): GoClient {
  return createGoClient(getGoOrigin());
}
