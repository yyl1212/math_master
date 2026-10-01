import { createPublicProxy, proxyError } from "@/lib/api/public-proxy";
import { getGoOrigin } from "@/lib/api/server-config";
export const dynamic = "force-dynamic";
export async function GET(
  request: Request,
  context: { params: Promise<{ segments: string[] }> },
) {
  try {
    return await createPublicProxy(getGoOrigin())(
      request,
      (await context.params).segments,
    );
  } catch {
    return proxyError(503);
  }
}
export async function HEAD(
  request: Request,
  context: { params: Promise<{ segments: string[] }> },
) {
  const r = await GET(request, context);
  return new Response(null, { status: r.status, headers: r.headers });
}
