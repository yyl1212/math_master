import "server-only";
import {z} from "zod";
import {getGoOrigin} from "../api/server-config";
import {contentResponseHeaders} from "../content/schemas";
import {readContentBytes} from "../content/bytes";
import {parseContentJSON} from "../content/raw-json";
export async function readExperienceMode():Promise<"legacy"|"topics"|null>{const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);try{const response=await fetch(getGoOrigin()+"/api/v2/topics/experience-mode",{method:"GET",credentials:"omit",cache:"no-store",redirect:"error",headers:{Accept:"application/json"},signal:controller.signal});contentResponseHeaders(response);if(!response.ok||response.headers.get("Cache-Control")!=="private, no-store"||!/^application\/json/.test(response.headers.get("Content-Type")??""))return null;return z.object({mode:z.enum(["legacy","topics"])}).strict().parse(parseContentJSON(await readContentBytes(response,2<<20,controller.signal))).mode}catch{return null}finally{clearTimeout(timer)}}
