"use client";
import {useUiI18n} from '@/lib/i18n/provider';
// Canonical names come from the directory or knowledge record, never from code matching.
export function LocalizedName({english,chinese}:{english?:string;chinese:string}){
 const {locale}=useUiI18n();
 return <>{locale==='zh-CN'?chinese||english:english||chinese}</>;
}
