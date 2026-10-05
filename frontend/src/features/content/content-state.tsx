"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AuthState } from "@/features/auth/auth-state";
export function ContentState({ status, code }: {
    status: number;
    code?: string;
}) {
 const {t}=useUiI18n();
 const router = useRouter(); if (code === "INVALID_COOKIE")
    return <AuthState kind="invalid-cookie"/>; if (status === 401)
    return <AuthState kind="anonymous"/>; if (status === 403)
    return <AuthState kind="forbidden"/>; return <section className="content-state"><h1>{status === 404 ? t("content-state.content.is.not.available.3b7767",{}) : t("content-state.content.management.is.temporarily.unavailable.0bcb14",{})}</h1>{status !== 404 && <button className="button secondary" onClick={() => router.refresh()}><UiText notice={uiMessage("content-state.try.again.d8b839",{})}/></button>}<p><Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("auth-state.explore.mathematics.088a8c",{})}/></Link></p></section>; }
