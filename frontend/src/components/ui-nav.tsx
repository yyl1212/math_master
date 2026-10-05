"use client";
import type {ReactNode} from "react";
import type {UiNotice} from "@/lib/i18n/types";
import {useUiI18n} from "@/lib/i18n/provider";
import {formatUiNotice} from "@/lib/i18n/format";
export function UiNav({label,children}:{label:UiNotice;children:ReactNode}){const{locale}=useUiI18n();return <nav aria-label={formatUiNotice(locale,label)}>{children}</nav>;}
