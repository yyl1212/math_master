import {createTopicManagementProxy,topicManagementPreflight} from "@/lib/api/topic-management-proxy";
import {topicProxyError} from "@/lib/api/taxonomy-proxy";
import {getGoOrigin} from "@/lib/api/server-config";
import {getAuthConfig,AuthNotConfiguredError} from "@/lib/auth/config";
export const dynamic="force-dynamic";
async function handle(request:Request,context:{params:Promise<{segments?:string[]}>}){
 try{const segments=(await context.params).segments??[],first=topicManagementPreflight(request,segments,"publications");if(first)return first;return await createTopicManagementProxy(getGoOrigin(),getAuthConfig())(request,segments,"publications")}catch(error){return topicProxyError(error instanceof AuthNotConfiguredError?"AUTH_NOT_CONFIGURED":"SERVICE_UNAVAILABLE")}
}
export {handle as GET,handle as POST,handle as PUT,handle as DELETE,handle as PATCH,handle as OPTIONS,handle as HEAD};
