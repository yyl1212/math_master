import { proxyFeedback } from '@/lib/api/feedback-proxy';
export const dynamic = 'force-dynamic';
async function handle(request: Request, context: { params: Promise<{ segments: string[] }> }) {
  return proxyFeedback(request, (await context.params).segments);
}
export { handle as GET, handle as POST, handle as PUT, handle as DELETE, handle as PATCH, handle as OPTIONS, handle as HEAD };
