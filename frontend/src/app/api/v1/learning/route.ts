import { learningProxyError } from "@/lib/api/learning-proxy";
export const dynamic = "force-dynamic";
function missing() { return learningProxyError("NOT_FOUND"); }
export { missing as GET, missing as POST, missing as PUT, missing as DELETE, missing as PATCH, missing as OPTIONS, missing as HEAD };
