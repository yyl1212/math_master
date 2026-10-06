"use client";
import {useUiI18n} from "@/lib/i18n/provider";
export function LanguageSwitch(){const{locale,setLocale,t}=useUiI18n();return <div className="language-switch" role="group" aria-label={t("locale.label",{})}><button type="button" aria-pressed={locale==="zh-CN"} onClick={()=>setLocale("zh-CN")} lang="zh-CN">中文</button><button type="button" aria-pressed={locale==="en"} onClick={()=>setLocale("en")} lang="en">English</button></div>;}
