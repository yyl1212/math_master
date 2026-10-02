import { questionProxyError } from "@/lib/api/question-proxy";
export const dynamic = "force-dynamic";
function missing() { return questionProxyError("NOT_FOUND"); }
export { missing as GET, missing as POST, missing as PUT, missing as DELETE, missing as PATCH, missing as OPTIONS, missing as HEAD };
