"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AuthState } from "@/features/auth/auth-state";
export function QuestionState({ status, code }: {
    status: number;
    code?: string;
}) { const router = useRouter(); if (code === "INVALID_COOKIE")
    return <AuthState kind="invalid-cookie"/>; if (status === 401)
    return <AuthState kind="anonymous"/>; if (status === 403)
    return <AuthState kind="forbidden"/>; return <section className="content-state"><h1>{status === 404 ? "Question data is not available." : code === "QUESTION_BANK_NOT_CONFIGURED" ? "The trusted question bank is not configured yet." : "Question management is temporarily unavailable."}</h1>{status !== 404 && <button className="button secondary" onClick={() => router.refresh()}>Try again</button>}<p><Link prefetch={false} href="/knowledge">Explore mathematics</Link></p></section>; }
