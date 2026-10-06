"use client";
import {createContext,useContext,useState,useCallback,useMemo,useEffect,type ReactNode} from "react";
import {buildUiLocaleCookie,isUiLocale,type UiLocale} from "./config";
import {formatUiNotice,uiMessage} from "./format";
import type {MessageKey,MessageValues} from "./types";
type UiI18n={locale:UiLocale;setLocale:(locale:UiLocale)=>void;t:<K extends MessageKey>(key:K,values:MessageValues<K>)=>string};
const defaults:UiI18n={locale:"en",setLocale:()=>{},t:(key,values)=>formatUiNotice("en",uiMessage(key,values))};
const UiLocaleContext=createContext<UiI18n>(defaults);
export function UiLocaleProvider({initialLocale,children}:{initialLocale:UiLocale;children:ReactNode}) {
 const[locale,setCurrent]=useState<UiLocale>(isUiLocale(initialLocale)?initialLocale:"en");
 const setLocale=useCallback((next:UiLocale)=>{if(!isUiLocale(next))return;setCurrent(next);try{document.cookie=buildUiLocaleCookie(next,location.protocol==="https:");}catch{/* This page still uses the selected language. */}},[]);
 const t=useCallback(<K extends MessageKey>(key:K,values:MessageValues<K>)=>formatUiNotice(locale,uiMessage(key,values)),[locale]);
 useEffect(()=>{document.documentElement.lang=locale;},[locale]);
 const value=useMemo(()=>({locale,setLocale,t}),[locale,setLocale,t]);
 return <UiLocaleContext.Provider value={value}>{children}</UiLocaleContext.Provider>;
}
export const useUiI18n=()=>useContext(UiLocaleContext);
