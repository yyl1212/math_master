export type UiLocale = "en" | "zh-CN";
export const UI_LOCALE_COOKIE="math_master_ui_locale";
export const UI_LOCALE_MAX_AGE=31536000;
export const isUiLocale=(value:unknown):value is UiLocale=>value==="en"||value==="zh-CN";
export function parseUiLocaleCookie(rawHeader:string):UiLocale {
 const pairs=rawHeader.split(";").map(p=>p.trim()).filter(p=>p.split("=",1)[0].trim()===UI_LOCALE_COOKIE);
 if(pairs.length!==1)return "en";
 const value=pairs[0].slice(pairs[0].indexOf("=")+1);
 return isUiLocale(value)?value:"en";
}
export function buildUiLocaleCookie(locale:UiLocale,secure:boolean):string {
 const safe=isUiLocale(locale)?locale:"en";
 return `${UI_LOCALE_COOKIE}=${safe}; Path=/; Max-Age=${UI_LOCALE_MAX_AGE}; SameSite=Lax${secure?"; Secure":""}`;
}
