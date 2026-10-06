"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AuthState } from "@/features/auth/auth-state";
export function QuestionState({ status, code }: {
    status: number;
    code?: string;
}) {
 const {t}=useUiI18n();
 const router = useRouter(); if (code === "INVALID_COOKIE")
    return <AuthState kind="invalid-cookie"/>; if (status === 401)
    return <AuthState kind="anonymous"/>; if (status === 403)
    return <AuthState kind="forbidden"/>; return <section className="content-state"><h1>{status === 404 ? t("question-state.question.data.is.not.available.fa945f",{}) : code === "QUESTION_BANK_NOT_CONFIGURED" ? t("question-state.the.trusted.question.bank.is.not.configured.yet.3d8238",{}) : t("question-state.question.management.is.temporarily.unavailable.53681c",{})}</h1>{status !== 404 && <button className="button secondary" onClick={() => router.refresh()}><UiText notice={uiMessage("content-state.try.again.d8b839",{})}/></button>}<p><Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("auth-state.explore.mathematics.088a8c",{})}/></Link></p></section>; }
