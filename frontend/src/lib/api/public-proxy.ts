import "server-only";
import { z } from "zod";
import {
  errorSchema,
  domainListSchema,
  domainDetailSchema,
  pathViewSchema,
  knowledgeViewSchema,
} from "./schemas";
import {
  readBoundedBytes,
  JSON_MAX_BYTES,
  SVG_MAX_BYTES,
  validId,
  safeRequestId,
} from "./server-client";
import { parseGoOrigin } from "./server-config";
export function proxyError(status: number): Response {
  const code =
    status === 404
      ? "NOT_FOUND"
      : status === 400
        ? "INVALID_QUERY"
        : status === 405
          ? "METHOD_NOT_ALLOWED"
          : "SERVICE_UNAVAILABLE";
  const message =
    status === 404
      ? "Resource not found."
      : status === 400
        ? "Invalid query parameters."
        : status === 405
          ? "Method not allowed."
          : "Service temporarily unavailable.";
  return Response.json(
    { error: { code, message, requestId: crypto.randomUUID() } },
    {
      status,
      headers: {
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
        ...(status === 405 ? { Allow: "GET, HEAD" } : {}),
      },
    },
  );
}
export function createPublicProxy(
  rawOrigin: string,
  fetcher: typeof fetch = fetch,
) {
  const origin = parseGoOrigin(rawOrigin);
  return async (request: Request, segments: string[]): Promise<Response> => {
    const head = request.method === "HEAD";
    const finish = (r: Response) =>
      head ? new Response(null, { status: r.status, headers: r.headers }) : r;
    if (!["GET", "HEAD"].includes(request.method))
      return finish(proxyError(405));
    const [group, id] = segments;
    const list = segments.length === 1 && group === "domains";
    const asset =
      segments.length === 2 && group === "assets" && /^[a-f0-9]{64}$/.test(id);
    const detail =
      segments.length === 2 &&
      ["domains", "paths", "knowledge"].includes(group) &&
      validId(id);
    if (!list && !asset && !detail) return finish(proxyError(404));
    const query = new URL(request.url).search;
    if (!list && query) return finish(proxyError(400));
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 5000);
    try {
      const response = await fetcher(
        origin + "/api/v1/" + segments.join("/") + query,
        {
          method: request.method,
          cache: "no-store",
          redirect: "error",
          signal: controller.signal,
          headers: { Accept: asset ? "image/svg+xml" : "application/json" },
        },
      );
      if (response.status !== 200) {
        const status = [400, 404].includes(response.status)
          ? response.status
          : 503;
        if (
          status === 503 ||
          response.headers.get("Content-Type")?.split(";")[0] !==
            "application/json"
        ) {
          void response.body?.cancel();
          return finish(proxyError(503));
        }
        // HEAD intentionally has no error body. Validate status and MIME only.
        if (head) {
          void response.body?.cancel();
          return finish(proxyError(status));
        }
        const bytes = await readBoundedBytes(
          response,
          JSON_MAX_BYTES,
          controller.signal,
        );
        const raw = JSON.parse(
          new TextDecoder("utf-8", { fatal: true }).decode(bytes),
        );
        return proxyError(errorSchema.safeParse(raw).success ? status : 503);
      }
      const expected = asset ? "image/svg+xml" : "application/json";
      if (response.headers.get("Content-Type")?.split(";")[0] !== expected) {
        void response.body?.cancel();
        return finish(proxyError(503));
      }
      const max = asset ? SVG_MAX_BYTES : JSON_MAX_BYTES;
      const headers = new Headers({
        "Content-Type": expected,
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      });
      if (asset)
        headers.set("Content-Security-Policy", "sandbox; default-src 'none'");
      const requestId = safeRequestId(response.headers.get("X-Request-ID"));
      if (requestId) headers.set("X-Request-ID", requestId);
      if (head) {
        const size = response.headers.get("Content-Length");
        if (size) {
          if (!/^\d+$/.test(size) || Number(size) > max) {
            void response.body?.cancel();
            return finish(proxyError(503));
          }
          headers.set("Content-Length", size);
        }
        void response.body?.cancel();
        return new Response(null, { status: 200, headers });
      }
      const bytes = await readBoundedBytes(response, max, controller.signal);
      if (asset) {
        headers.set("Content-Length", String(bytes.byteLength));
        return new Response(bytes.buffer as ArrayBuffer, { headers });
      }
      const raw = JSON.parse(
        new TextDecoder("utf-8", { fatal: true }).decode(bytes),
      );
      const validator = list
        ? domainListSchema
        : z.object({
            data:
              group === "domains"
                ? domainDetailSchema
                : group === "paths"
                  ? pathViewSchema
                  : knowledgeViewSchema,
          });
      const result = validator.safeParse(raw);
      return result.success
        ? Response.json(result.data, { headers })
        : proxyError(503);
    } catch {
      return finish(proxyError(503));
    } finally {
      clearTimeout(timer);
    }
  };
}
