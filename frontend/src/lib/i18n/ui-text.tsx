"use client";
import {useUiI18n} from "./provider";
import {formatUiNotice} from "./format";
import type {UiNotice} from "./types";
export function UiText({notice}:{notice:UiNotice}){const{locale}=useUiI18n();return <>{formatUiNotice(locale,notice)}</>;}
