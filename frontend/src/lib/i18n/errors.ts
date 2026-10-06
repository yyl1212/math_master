import type {UiErrorNamespace,UiNotice,MessageKey} from "./types";
export function uiError(namespace:UiErrorNamespace,failure:{code:string;requestId?:string;retryAfter?:number}):UiNotice {
 return {kind:"error",namespace,code:failure.code,...(failure.requestId?{requestId:failure.requestId}:{}),...(failure.retryAfter!==undefined?{retryAfter:failure.retryAfter}:{})};
}
import {errorKeys} from "./error-keys";
export const errorMessageKey=(namespace:UiErrorNamespace,code:string):MessageKey=>{
 const scoped=(errorKeys as Partial<Record<UiErrorNamespace,Record<string,MessageKey>>>)[namespace];
 if(scoped&&Object.hasOwn(scoped,code))return scoped[code];
 if(namespace==="auth"&&code==="FORBIDDEN")return "auth.error.forbidden";
 if(namespace==="auth"&&code==="NOT_FOUND")return "auth.error.notFound";
 if(namespace==="feedback"&&code==="NOT_FOUND")return "feedback.error.notFound";
 return "common.unavailable";
};
