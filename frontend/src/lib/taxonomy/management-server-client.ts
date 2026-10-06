import "server-only";
import {getGoOrigin} from "../api/server-config";
import {getAuthConfig,AuthNotConfiguredError} from "../auth/config";
import {selectAuthCookies} from "../auth/cookies";
import {taxonomyRouteRequest,readTaxonomyResponse,taxonomyFailure} from "./protocol";
import type {TopicManagementRoute,TaxonomyResult} from "./types";
export async function readServerTopicManagement<T>(route:TopicManagementRoute,cookieHeader:string):Promise<TaxonomyResult<T>>{
 const target=taxonomyRouteRequest(route);if(!target||target.method!=="GET"||target.path.startsWith("/api/v2/topics"))return taxonomyFailure("INVALID_REQUEST");const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);
 try{const config=getAuthConfig(),cookie=selectAuthCookies(cookieHeader,config.production,true);const response=await fetch(getGoOrigin()+target.path,{method:"GET",headers:{Accept:"application/json",...(cookie?{Cookie:cookie}:{})},cache:"no-store",redirect:"error",signal:controller.signal});return (await readTaxonomyResponse(response,target.kind,controller.signal)).result as TaxonomyResult<T>}catch(error){return taxonomyFailure(error instanceof AuthNotConfiguredError?"AUTH_NOT_CONFIGURED":"SERVICE_UNAVAILABLE")}finally{clearTimeout(timer)}
}
