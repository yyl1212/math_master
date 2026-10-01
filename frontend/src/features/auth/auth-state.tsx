"use client";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getAuthContext } from "@/lib/auth/client";
import styles from "@/styles/auth.module.css";
export function FormMessage({ message, error = false }: { message: string | null; error?: boolean }) {
  const ref = useRef<HTMLParagraphElement>(null);
  useEffect(() => { if (message && error) ref.current?.focus(); }, [message, error]);
  return message ? <p ref={ref} role={error ? "alert" : "status"} tabIndex={error ? -1 : undefined} className={error ? styles.error : styles.notice}>{message}</p> : null;
}
export function AuthState({ kind }: { kind: "anonymous" | "forbidden" | "unavailable" | "invalid-cookie" }) {
  const router = useRouter(), pending = useRef(false);
  const [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null);
  async function recover() {
    if (pending.current) return;
    pending.current = true; setBusy(true);
    try {
      const result = await getAuthContext();
      if (result.ok || result.code === "INVALID_COOKIE") router.refresh();
      else setError("Accounts are temporarily unavailable.");
    } finally { pending.current = false; setBusy(false); }
  }
  const text = { anonymous: "Sign in to view your account", forbidden: "You do not have permission.", unavailable: "Accounts are temporarily unavailable.", "invalid-cookie": "Your sign-in cookie needs to be cleared." }[kind];
  return <section className="content-state"><p className="eyebrow">YOUR ACCOUNT</p><h1>{text}</h1>
    {kind === "anonymous" && <Link className="button" prefetch={false} href="/login">Sign in</Link>}
    {kind === "forbidden" && <Link className="button secondary" prefetch={false} href="/account">View account</Link>}
    {kind === "invalid-cookie" && <button className="button" disabled={busy} onClick={recover}>Clear sign-in cookie</button>}
    {kind === "unavailable" && <button className="button secondary" onClick={() => router.refresh()}>Try again</button>}
    <FormMessage message={error} error />
    <Link prefetch={false} href="/knowledge">Explore mathematics</Link>
  </section>;
}
