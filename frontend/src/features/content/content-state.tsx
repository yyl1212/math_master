"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AuthState } from "@/features/auth/auth-state";
export function ContentState({ status, code }: {
    status: number;
    code?: string;
}) { const router = useRouter(); if (code === "INVALID_COOKIE")
    return <AuthState kind="invalid-cookie"/>; if (status === 401)
    return <AuthState kind="anonymous"/>; if (status === 403)
    return <AuthState kind="forbidden"/>; return <section className="content-state"><h1>{status === 404 ? "Content is not available." : "Content management is temporarily unavailable."}</h1>{status !== 404 && <button className="button secondary" onClick={() => router.refresh()}>Try again</button>}<p><Link prefetch={false} href="/knowledge">Explore mathematics</Link></p></section>; }
