"use client";
import {enumKeys} from "./enum-keys";
import {uiMessage,formatUiNotice} from "./format";
import type {UiLocale} from "./config";
import type {UiNotice,StaticMessageKey} from "./types";
import {UiText} from "./ui-text";
export type UiEnumGroup=keyof typeof enumKeys;
export function enumNotice(group:UiEnumGroup,code:string):UiNotice{
 const keys:Readonly<Record<string,StaticMessageKey>>=enumKeys[group];return Object.hasOwn(keys,code)?uiMessage(keys[code],{}):{kind:"literal",text:code};
}
export const enumOptionLabels=(group:UiEnumGroup,values:readonly string[]):Readonly<Record<string,UiNotice>>=>Object.fromEntries(values.map(v=>[v,enumNotice(group,v)]));
export const formatUiEnum=(locale:UiLocale,group:UiEnumGroup,code:string)=>formatUiNotice(locale,enumNotice(group,code));
export function UiEnum({group,value}:{group:UiEnumGroup;value:string}){return <UiText notice={enumNotice(group,value)}/>;}
