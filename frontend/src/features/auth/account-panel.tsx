"use client";
import Link from "next/link";
import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { authRequest, notifyAuthChanged } from "@/lib/auth/client";
import { validPrivateInput } from "@/lib/auth/schemas";
import type { User } from "@/lib/auth/types";
import { FormMessage } from "./auth-state";
import styles from "@/styles/auth.module.css";
export function AccountPanel({ user }: { user: User }) {
  const router = useRouter(), pending = useRef(false);
  const [currentPassword, setCurrent] = useState(""), [newPassword, setNew] = useState("");
  const [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null);
  async function write(kind: "password" | "logout" | "logout-all") {
    if (pending.current) return;
    pending.current = true; setBusy(true); setError(null);
    const input = kind === "password" ? { currentPassword, newPassword } : {};
    try {
      if (!validPrivateInput(kind, input)) { setError("Passwords must contain 15–128 characters."); return; }
      const result = await authRequest({ kind }, input);
      if (!result.ok) { setError(result.message); return; }
      notifyAuthChanged(); router.replace(kind === "password" ? "/login" : "/"); router.refresh();
    } finally { setCurrent(""); setNew(""); pending.current = false; setBusy(false); }
  }
  return <section className={styles.account}><div className="page-heading"><p className="eyebrow">YOUR ACCOUNT</p><h1>{user.username}</h1><p>{user.roles.join(" · ")}</p></div>
    {user.mustChangePassword && <p className={styles.notice}>Change your password to continue</p>}
    <div className={styles.card}><h2>Change password</h2><p className={styles.hint}>Changing your password signs you out on every device. Sign in again with your new password.</p>
      <form className={styles.form} onSubmit={e => { e.preventDefault(); void write("password"); }} aria-busy={busy}>
        <label htmlFor="current-password">Current password</label><input id="current-password" type="password" autoComplete="current-password" required value={currentPassword} onChange={e => setCurrent(e.target.value)} disabled={busy} />
        <label htmlFor="new-password">New password</label><input id="new-password" type="password" autoComplete="new-password" required value={newPassword} onChange={e => setNew(e.target.value)} disabled={busy} aria-describedby="new-password-hint" />
        <p id="new-password-hint" className={styles.hint}>15–128 characters. Spaces and Unicode characters are welcome.</p>
        <FormMessage message={error} error /><button className="button" disabled={busy}>Change password</button>
      </form></div>
    <div className={styles.actions}><button className="button secondary" disabled={busy} onClick={() => void write("logout")}>Sign out</button>
      {!user.mustChangePassword && <button className="button secondary" disabled={busy} onClick={() => void write("logout-all")}>Sign out everywhere</button>}
      {!user.mustChangePassword && user.roles.includes("admin") && <Link className="button secondary" prefetch={false} href="/admin/users">Manage users</Link>}
    </div></section>;
}
