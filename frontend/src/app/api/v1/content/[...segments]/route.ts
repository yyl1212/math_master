import {createContentProxy,contentProxyError,contentRoutePreflight} from "@/lib/api/content-proxy";
import {getGoOrigin} from "@/lib/api/server-config";
import {getAuthConfig,AuthNotConfiguredError} from "@/lib/auth/config";
export const dynamic="force-dynamic";
async function handle(request:Request,context:{params:Promise<{segments:string[]}>}){
 try{const segments=(await context.params).segments,preflight=contentRoutePreflight(request,segments);if(preflight)return preflight;return await createContentProxy(getGoOrigin(),getAuthConfig())(request,segments)}catch(error){return contentProxyError(error instanceof AuthNotConfiguredError?"AUTH_NOT_CONFIGURED":"SERVICE_UNAVAILABLE")}
}
export {handle as GET,handle as POST,handle as PUT,handle as DELETE,handle as PATCH,handle as OPTIONS,handle as HEAD};
