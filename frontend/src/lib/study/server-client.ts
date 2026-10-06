import "server-only";
import {getGoOrigin} from "../api/server-config";
import {getAuthConfig,AuthNotConfiguredError} from "../auth/config";
import {selectAuthCookies} from "../auth/cookies";
import {studyRouteRequest,readStudyResponse,studyFailure} from "./protocol";
import type {StudyRoute,StudyResult} from "./types";
export async function readServerStudy<T>(route:StudyRoute,cookieHeader:string):Promise<StudyResult<T>>{const target=studyRouteRequest(route);if(!target||target.method!=="GET")return studyFailure("INVALID_REQUEST");const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);try{const config=getAuthConfig(),cookie=selectAuthCookies(cookieHeader,config.production,true);const response=await fetch(getGoOrigin()+target.path,{method:"GET",headers:{Accept:"application/json",...(cookie?{Cookie:cookie}:{})},cache:"no-store",redirect:"error",signal:controller.signal});return(await readStudyResponse(response,target.kind,controller.signal)).result as StudyResult<T>}catch(error){return studyFailure(error instanceof AuthNotConfiguredError?"AUTH_NOT_CONFIGURED":"SERVICE_UNAVAILABLE")}finally{clearTimeout(timer)}}
