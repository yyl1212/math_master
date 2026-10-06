import "server-only";
import {getGoOrigin} from "../api/server-config";
import {taxonomyRouteRequest,readTaxonomyResponse,taxonomyFailure} from "./protocol";
import type {TaxonomyRoute,TaxonomyResult} from "./types";
export async function readServerTaxonomy<T>(route:TaxonomyRoute):Promise<TaxonomyResult<T>>{
 const target=taxonomyRouteRequest(route);if(!target||!target.path.startsWith("/api/v2/topics"))return taxonomyFailure("INVALID_REQUEST");const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);
 try{const response=await fetch(getGoOrigin()+target.path,{method:"GET",headers:{Accept:"application/json"},cache:"no-store",redirect:"error",signal:controller.signal});return (await readTaxonomyResponse(response,target.kind,controller.signal)).result as TaxonomyResult<T>}catch{return taxonomyFailure()}finally{clearTimeout(timer)}
}
