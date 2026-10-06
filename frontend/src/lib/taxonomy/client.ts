import {taxonomyRouteRequest,readTaxonomyResponse,taxonomyFailure} from "./protocol";
import type {TaxonomyRoute,TaxonomyResult} from "./types";
export async function taxonomyRequest<T>(route:TaxonomyRoute,signal?:AbortSignal):Promise<TaxonomyResult<T>>{
 const target=taxonomyRouteRequest(route);if(!target||!target.path.startsWith("/api/v2/topics"))return taxonomyFailure("INVALID_REQUEST");
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000),abort=()=>controller.abort();signal?.addEventListener("abort",abort,{once:true});
 try{if(signal?.aborted)return taxonomyFailure();const response=await fetch(target.path,{method:"GET",headers:{Accept:"application/json"},credentials:"omit",cache:"no-store",redirect:"error",signal:controller.signal});return (await readTaxonomyResponse(response,target.kind,controller.signal)).result as TaxonomyResult<T>}catch{return taxonomyFailure()}finally{clearTimeout(timer);signal?.removeEventListener("abort",abort)}
}
