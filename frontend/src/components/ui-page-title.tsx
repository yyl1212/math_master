"use client";
import {useEffect,useRef} from "react";
import {usePathname} from "next/navigation";
import {useUiI18n} from "@/lib/i18n/provider";
import type {PageTitleKey} from "@/lib/i18n/page-titles";
/** Owns only functional metadata. Mathematical content is never inspected. */
export function UiPageTitle({messageKey}:{messageKey:PageTitleKey}){
 const {t}=useUiI18n();const pathname=usePathname();const owner=useRef(pathname);
 const title=t(messageKey,{})+" | Math Master";
 useEffect(()=>{
  if(pathname!==owner.current)return;
  const apply=()=>{if(document.title!==title)document.title=title;};apply();
  // Next may stream metadata after hydration. Observe the head only.
  const observer=new MutationObserver(apply);observer.observe(document.head,{subtree:true,childList:true,characterData:true});
  return ()=>observer.disconnect();
 },[title,pathname]);return null;
}
