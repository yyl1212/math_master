import {createTaxonomyProxy,topicProxyError} from "@/lib/api/taxonomy-proxy";
import {getGoOrigin} from "@/lib/api/server-config";
export const dynamic="force-dynamic";
async function handle(request:Request,context:{params:Promise<{segments?:string[]}>}){try{return await createTaxonomyProxy(getGoOrigin())(request,(await context.params).segments??[])}catch{return topicProxyError()}}
export {handle as GET,handle as POST,handle as PUT,handle as DELETE,handle as PATCH,handle as OPTIONS,handle as HEAD};
