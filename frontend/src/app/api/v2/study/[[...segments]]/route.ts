import {getGoOrigin} from "@/lib/api/server-config";
import {getAuthConfig} from "@/lib/auth/config";
import {createStudyProxy,studyProxyError} from "@/lib/api/study-proxy";
export const dynamic="force-dynamic";
async function handle(request:Request){try{return await createStudyProxy(getGoOrigin(),getAuthConfig())(request)}catch{return studyProxyError("AUTH_NOT_CONFIGURED")}}
export const GET=handle,POST=handle,PUT=handle,DELETE=handle,PATCH=handle,HEAD=handle,OPTIONS=handle;
