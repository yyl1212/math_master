import {enMessages} from "./messages/en";
import {zhMessages} from "./messages/zh-CN";
import {errorMessageKey} from "./errors";
import type {UiLocale} from "./config";
import type {MessageKey,MessageValues,UiMessage,UiNotice} from "./types";
export function uiMessage<K extends MessageKey>(key:K,values:MessageValues<K>):UiMessage {return {kind:"system",key,values} as UiMessage;}
export function formatUiNotice(locale:UiLocale,notice:UiNotice):string {
 if(notice.kind==="literal")return notice.text;
 const key=notice.kind==="error"?errorMessageKey(notice.namespace,notice.code):notice.key;
 if(!Object.hasOwn(enMessages,key))return locale==="zh-CN"?zhMessages["common.unavailable"]:enMessages["common.unavailable"].text;
 const values=notice.kind==="system"?notice.values:{};
 const entry=enMessages[key];const text=locale==="zh-CN"?zhMessages[key]:entry.text;
 return text.replace(/\{([a-zA-Z0-9_]+)\}/g,(_,name:string)=>String((values as Readonly<Record<string,string|number>>)[name]??""));
}

export const uiValue=(value:string|number|boolean|null|undefined):string|number=>typeof value==="string"||typeof value==="number"?value:"";
