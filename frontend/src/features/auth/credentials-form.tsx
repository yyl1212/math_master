"use client";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { authRequest, getAuthContext, notifyAuthChanged } from "@/lib/auth/client";
import { validPrivateInput } from "@/lib/auth/schemas";
import { FormMessage } from "./auth-state";
import styles from "@/styles/auth.module.css";
export function CredentialsForm({ mode }: { mode: "register" | "login" }) {
  const router = useRouter(), pending = useRef(false);
  const [username, setUsername] = useState(""), [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null);
  useEffect(() => { let live = true; void getAuthContext().then(result => { if (live && !result.ok) setError(result.message); }); return () => { live = false; }; }, []);
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (pending.current) return;
    pending.current = true; setBusy(true); setError(null);
    const input = { username, password };
    try {
      if (!validPrivateInput(mode, input)) { setError("Use a username of 3–32 letters, digits or underscores, and a password of 15–128 characters."); return; }
      const result = await authRequest({ kind: mode }, input);
      if (!result.ok) { setError(result.message); return; }
      notifyAuthChanged(); router.replace(mode === "register" ? "/login" : "/account"); router.refresh();
    } finally { setPassword(""); pending.current = false; setBusy(false); }
  }
  return <section className={styles.credentials}><div className={styles.intro}><p className="eyebrow">A WORLD OF IDEAS</p>
    <h1>{mode === "register" ? "Make room for mathematics." : "Welcome back."}</h1>
    <p>{mode === "register" ? "Create an account to begin your next chapter." : "Sign in to your Math Master account."}</p>
    <p>From first principles to new frontiers, explore a connected world of mathematical ideas.</p>
    <Link prefetch={false} href="/knowledge">Explore the Knowledge Map →</Link></div>
    <div className={styles.card}><h2>{mode === "register" ? "Create account" : "Sign in"}</h2>
    <form className={styles.form} onSubmit={submit} aria-busy={busy}>
      <label htmlFor="username">Username</label><input id="username" name="username" autoComplete="username" autoCapitalize="none" spellCheck={false} required minLength={3} maxLength={32} pattern="[A-Za-z0-9_]{3,32}" value={username} onChange={e => setUsername(e.target.value)} disabled={busy} />
      <label htmlFor="password">Password</label><input id="password" name="password" type="password" autoComplete={mode === "register" ? "new-password" : "current-password"} required value={password} onChange={e => setPassword(e.target.value)} disabled={busy} aria-describedby="password-hint" />
      <p id="password-hint" className={styles.hint}>15–128 characters. Spaces and Unicode characters are welcome.</p>
      <FormMessage message={error} error />
      <button className="button" type="submit" disabled={busy}>{busy ? "Please wait…" : mode === "register" ? "Create account" : "Sign in"}</button>
    </form><p className={styles.switch}>{mode === "register" ? "Already have an account? " : "New to Math Master? "}<Link prefetch={false} href={mode === "register" ? "/login" : "/register"}>{mode === "register" ? "Sign in" : "Create account"}</Link></p></div>
  </section>;
}
