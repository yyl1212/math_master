"use client";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { authRequest, getAuthContext, notifyAuthChanged } from "@/lib/auth/client";
import { validPrivateInput } from "@/lib/auth/schemas";
import type { AuthResult, Role, User, UserPage } from "@/lib/auth/types";
import { AuthState, FormMessage } from "./auth-state";
import styles from "@/styles/auth.module.css";
export function AdminUsers({ initial }: { initial: UserPage }) {
  const router = useRouter(), pending = useRef(false), verifyInput = useRef<HTMLInputElement>(null);
  const [access, setAccess] = useState<"loading" | "allowed" | "anonymous" | "forbidden" | "unavailable" | "invalid-cookie">("loading");
  const [page, setPage] = useState(initial), [query, setQuery] = useState(""), [applied, setApplied] = useState("");
  const [selected, setSelected] = useState(initial.items[0]?.id ?? ""), [roles, setRoles] = useState<Role[]>(initial.items[0]?.roles ?? ["learner"]);
  const [reason, setReason] = useState(""), [ownershipNote, setOwnership] = useState(""), [temporaryPassword, setTemporary] = useState(""), [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false), [verify, setVerify] = useState(false), [error, setError] = useState<string | null>(null), [message, setMessage] = useState<string | null>(null);
  useEffect(() => { let live = true; void getAuthContext().then(result => {
    if (!live) return;
    setAccess(!result.ok ? result.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable" : !result.data.user ? "anonymous" : result.data.user.mustChangePassword || !result.data.user.roles.includes("admin") ? "forbidden" : "allowed");
  }); return () => { live = false; }; }, []);
  useEffect(() => { if (verify) verifyInput.current?.focus(); }, [verify]);
  function choose(user: User | undefined) { setSelected(user?.id ?? ""); setRoles(user?.roles ?? ["learner"]); setTemporary(""); setOwnership(""); setReason(""); }
  function handleFailure(result: Extract<AuthResult<unknown>, { ok: false }>) {
    if (result.code === "AUTHENTICATION_REQUIRED") { notifyAuthChanged(); router.replace("/login"); router.refresh(); }
    else if (result.code === "FORBIDDEN" || result.code === "PASSWORD_CHANGE_REQUIRED") { setAccess("forbidden"); notifyAuthChanged(); }
    else if (result.code === "REAUTHENTICATION_REQUIRED") { setVerify(true); setError(result.message); }
    else setError(result.message);
  }
  async function load(q: string, offset: number) {
    const result = await authRequest<UserPage>({ kind: "users", query: { q, limit: page.limit, offset } }, undefined);
    if (!result.ok) { handleFailure(result); return false; }
    setPage(result.data); setApplied(q); choose(result.data.items.find(u => u.id === selected) ?? result.data.items[0]); return true;
  }
  async function search(offset = 0, q = query) {
    if (pending.current) return;
    pending.current = true; setBusy(true); setError(null); setMessage(null);
    try { await load(q, offset); } finally { pending.current = false; setBusy(false); }
  }
  async function write(kind: "roles" | "reset" | "reauth") {
    if (pending.current) return;
    pending.current = true; setBusy(true); setError(null); setMessage(null);
    const input = kind === "roles" ? { roles, reason } : kind === "reset" ? { temporaryPassword, reason, ownershipNote } : { password };
    try {
      if (!validPrivateInput(kind, input) || (kind !== "reauth" && !selected)) { setError(kind === "reauth" ? "Passwords must contain 15–128 characters." : "Provide a reason of 10–1000 characters. A password reset also requires ownership verification and a password of 15–128 characters."); return; }
      const result = await authRequest(kind === "reauth" ? { kind } : { kind, userId: selected }, input);
      if (!result.ok) { handleFailure(result); return; }
      if (kind === "reauth") { setVerify(false); setMessage("Password verified. Submit your change again."); return; }
      notifyAuthChanged();
      const context = await getAuthContext();
      if (!context.ok) { setError(context.message); return; }
      if (!context.data.user) { router.replace("/login"); router.refresh(); return; }
      if (!context.data.user.roles.includes("admin") || context.data.user.mustChangePassword) { setAccess("forbidden"); return; }
      if (await load(applied, page.offset)) setMessage("Changes saved.");
    } finally { setTemporary(""); setPassword(""); pending.current = false; setBusy(false); }
  }
  if (access === "loading") return <p role="status">Checking account…</p>;
  if (access !== "allowed") return <AuthState kind={access} />;
  return <section className={styles.admin}><div className="page-heading"><p className="eyebrow">ADMINISTRATION</p><h1>People & permissions</h1><p>Manage roles and help verified account owners regain access.</p></div>
    <form className={styles.search} onSubmit={e => { e.preventDefault(); void search(); }}><label htmlFor="user-query">Search usernames</label><input id="user-query" value={query} onChange={e => setQuery(e.target.value)} maxLength={128} disabled={busy} /><button className="button" disabled={busy}>Search</button></form>
    <p className={styles.hint}>{page.total} {page.total === 1 ? "account" : "accounts"}</p>
    <div className={styles.adminGrid}><div className={styles.card}><h2>Accounts</h2><ul className={styles.users}>{page.items.map(user => <li key={user.id}><button aria-pressed={selected === user.id} disabled={busy} onClick={() => choose(user)}><strong>{user.username}</strong><span>{user.roles.join(" · ")}</span>{user.mustChangePassword && <span>Password change required</span>}</button></li>)}</ul>
      {page.items.length === 0 && <p>No matching accounts.</p>}<div className={styles.actions}><button className="button secondary" disabled={busy || page.offset === 0} onClick={() => void search(Math.max(0, page.offset - page.limit), applied)}>Previous</button><button className="button secondary" disabled={busy || page.offset + page.limit >= page.total} onClick={() => void search(page.offset + page.limit, applied)}>Next</button></div></div>
    {selected && <div className={styles.card}><h2>{page.items.find(u => u.id === selected)?.username}</h2><form className={styles.form} onSubmit={e => { e.preventDefault(); void write("roles"); }}>
      <fieldset disabled={busy}><legend>Roles</legend>{(["learner", "editor", "reviewer", "admin"] as Role[]).map(role => <label className={styles.check} key={role}><input type="checkbox" checked={roles.includes(role)} disabled={role === "learner"} onChange={e => setRoles(current => (["learner", "editor", "reviewer", "admin"] as Role[]).filter(r => r === role ? e.target.checked : current.includes(r)))} />{role}</label>)}</fieldset>
      <label htmlFor="role-reason">Reason</label><textarea id="role-reason" required minLength={10} maxLength={1000} value={reason} onChange={e => setReason(e.target.value)} disabled={busy} /><button className="button" disabled={busy}>Save roles</button></form>
      <form className={styles.form} onSubmit={e => { e.preventDefault(); void write("reset"); }}><h3>Reset password</h3><p className={styles.hint}>Verify ownership outside this website before resetting an account. The owner must change this temporary password after signing in.</p>
        <label htmlFor="ownership">Ownership verification</label><textarea id="ownership" required minLength={10} maxLength={1000} value={ownershipNote} onChange={e => setOwnership(e.target.value)} disabled={busy} />
        <label htmlFor="temporary-password">Temporary password</label><input id="temporary-password" type="password" autoComplete="new-password" required value={temporaryPassword} onChange={e => setTemporary(e.target.value)} disabled={busy} />
        <p className={styles.hint}>The reason above also applies to this reset.</p><button className="button secondary" disabled={busy}>Reset password</button></form>
    </div>}</div>
    <FormMessage message={error} error /><FormMessage message={message} />
    {verify && <div className={styles.backdrop}><section role="dialog" aria-modal="true" aria-labelledby="verify-title" className={styles.dialog} onKeyDown={e => { if (e.key === "Escape" && !pending.current) { setVerify(false); setPassword(""); } if (e.key === "Tab") { const nodes = [...e.currentTarget.querySelectorAll<HTMLElement>("input:not(:disabled),button:not(:disabled)")]; const first = nodes[0], last = nodes[nodes.length - 1]; if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last?.focus(); } else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first?.focus(); } } }}><h2 id="verify-title">Verify your password</h2><p>Verification lasts five minutes. You will submit your change separately.</p>
      <form className={styles.form} onSubmit={e => { e.preventDefault(); void write("reauth"); }}><label htmlFor="verify-password">Your password</label><input ref={verifyInput} id="verify-password" type="password" autoComplete="current-password" required value={password} onChange={e => setPassword(e.target.value)} disabled={busy} /><FormMessage message={error} error /><button className="button" disabled={busy}>Verify password</button><button className="button secondary" type="button" disabled={busy} onClick={() => { setVerify(false); setPassword(""); setError(null); }}>Cancel</button></form></section></div>}
  </section>;
}
