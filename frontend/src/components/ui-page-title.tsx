"use client";
import {useEffect} from "react";
import {usePathname} from "next/navigation";
import {useUiI18n} from "@/lib/i18n/provider";
import type {PageTitleKey} from "@/lib/i18n/page-titles";
/** Owns only functional metadata. Mathematical content is never inspected. */
export function UiPageTitle({messageKey}:{messageKey:PageTitleKey}){
 const {t}=useUiI18n();const pathname=usePathname();
 const title=t(messageKey,{})+" | Math Master";
 useEffect(()=>{
  const apply=()=>{if(document.title!==title)document.title=title;};apply();
  // Next may stream metadata after hydration. Observe the head only.
  const observer=new MutationObserver(apply);observer.observe(document.head,{subtree:true,childList:true,characterData:true});
  return ()=>observer.disconnect();
 },[title,pathname]);return null;
}
