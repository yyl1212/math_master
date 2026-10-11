"use client";
import type {UiNotice} from "@/lib/i18n/types";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import {LanguageSwitch} from "@/components/language-switch";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { authRequest, getAuthContext, notifyAuthChanged } from "@/lib/auth/client";
import { validPrivateInput } from "@/lib/auth/schemas";
import type { AuthResult, Role, User, UserPage } from "@/lib/auth/types";
import { AuthState, FormMessage } from "./auth-state";
import styles from "@/styles/auth.module.css";
export function AdminUsers({ initial }: { initial: UserPage }) {
 const {t}=useUiI18n();

  const router = useRouter(), pending = useRef(false), verifyInput = useRef<HTMLInputElement>(null);
  const [access, setAccess] = useState<"loading" | "allowed" | "anonymous" | "forbidden" | "unavailable" | "invalid-cookie">("loading");
  const [page, setPage] = useState(initial), [query, setQuery] = useState(""), [applied, setApplied] = useState("");
  const [selected, setSelected] = useState(initial.items[0]?.id ?? ""), [roles, setRoles] = useState<Role[]>(initial.items[0]?.roles ?? ["learner"]);
  const [reason, setReason] = useState(""), [ownershipNote, setOwnership] = useState(""), [temporaryPassword, setTemporary] = useState(""), [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false), [verify, setVerify] = useState(false), [error, setError] = useState<UiNotice | null>(null), [message, setMessage] = useState<UiNotice | null>(null);
  useEffect(() => { let live = true; void getAuthContext().then(result => {
    if (!live) return;
    setAccess(!result.ok ? result.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable" : !result.data.user ? "anonymous" : result.data.user.mustChangePassword || !result.data.user.roles.includes("admin") ? "forbidden" : "allowed");
  }); return () => { live = false; }; }, []);
  useEffect(() => { if (verify) verifyInput.current?.focus(); }, [verify]);
  function choose(user: User | undefined) { setSelected(user?.id ?? ""); setRoles(user?.roles ?? ["learner"]); setTemporary(""); setOwnership(""); setReason(""); }
  function handleFailure(result: Extract<AuthResult<unknown>, { ok: false }>) {
    if (result.code === "AUTHENTICATION_REQUIRED") { notifyAuthChanged(); router.replace("/login"); router.refresh(); }
    else if (result.code === "FORBIDDEN" || result.code === "PASSWORD_CHANGE_REQUIRED") { setAccess("forbidden"); notifyAuthChanged(); }
    else if (result.code === "REAUTHENTICATION_REQUIRED") { setVerify(true); setError(uiError("auth",result)); }
    else setError(uiError("auth",result));
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
      if (!validPrivateInput(kind, input) || (kind !== "reauth" && !selected)) { setError(kind === "reauth" ? uiMessage("auth.input.password",{}) : uiMessage("auth.input.admin",{})); return; }
      const result = await authRequest(kind === "reauth" ? { kind } : { kind, userId: selected }, input);
      if (!result.ok) { handleFailure(result); return; }
      if (kind === "reauth") { setVerify(false); setMessage(uiMessage("auth.verified",{})); return; }
      notifyAuthChanged();
      const context = await getAuthContext();
      if (!context.ok) { setError(uiError("auth",context)); return; }
      if (!context.data.user) { router.replace("/login"); router.refresh(); return; }
      if (!context.data.user.roles.includes("admin") || context.data.user.mustChangePassword) { setAccess("forbidden"); return; }
      if (await load(applied, page.offset)) setMessage(uiMessage("auth.saved",{}));
    } finally { setTemporary(""); setPassword(""); pending.current = false; setBusy(false); }
  }
  if (access === "loading") return <p role="status"><UiText notice={uiMessage("auth-status.checking.account.d18f41",{})}/></p>;
  if (access !== "allowed") return <AuthState kind={access} />;
  return <section className={styles.admin}><div className="page-heading"><p className="eyebrow"><UiText notice={uiMessage("admin-users.administration.bbdbcb",{})}/></p><h1><UiText notice={uiMessage("page.admin.users",{})}/></h1><p><UiText notice={uiMessage("admin-users.manage.roles.and.help.verified.account.owners.regain.access.3f1280",{})}/></p></div>
    <form className={styles.search} onSubmit={e => { e.preventDefault(); void search(); }}><label htmlFor="user-query"><UiText notice={uiMessage("admin-users.search.usernames.adb993",{})}/></label><input id="user-query" value={query} onChange={e => setQuery(e.target.value)} maxLength={128} disabled={busy} /><button className="button" disabled={busy}><UiText notice={uiMessage("knowledge-map.search.49c266",{})}/></button></form>
    <p className={styles.hint}>{page.total} {page.total === 1 ? t("admin-users.account.9af211",{}) : t("admin-users.accounts.bc62a3",{})}</p>
    <div className={styles.adminGrid}><div className={styles.card}><h2><UiText notice={uiMessage("admin-users.accounts.8a7c8b",{})}/></h2><ul className={styles.users}>{page.items.map(user => <li key={user.id}><button aria-pressed={selected === user.id} disabled={busy} onClick={() => choose(user)}><strong>{user.username}</strong><span>{user.roles.map(role=>t(roleKeys[role],{})).join(" · ")}</span>{user.mustChangePassword && <span><UiText notice={uiMessage("admin-users.password.change.required.77288b",{})}/></span>}</button></li>)}</ul>
      {page.items.length === 0 && <p><UiText notice={uiMessage("admin-users.no.matching.accounts.277fda",{})}/></p>}<div className={styles.actions}><button className="button secondary" disabled={busy || page.offset === 0} onClick={() => void search(Math.max(0, page.offset - page.limit), applied)}><UiText notice={uiMessage("admin-users.previous.a57b08",{})}/></button><button className="button secondary" disabled={busy || page.offset + page.limit >= page.total} onClick={() => void search(page.offset + page.limit, applied)}><UiText notice={uiMessage("admin-users.next.1ff57a",{})}/></button></div></div>
    {selected && <div className={styles.card}><h2>{page.items.find(u => u.id === selected)?.username}</h2><form className={styles.form} onSubmit={e => { e.preventDefault(); void write("roles"); }}>
      <fieldset disabled={busy}><legend><UiText notice={uiMessage("admin-users.roles.c25337",{})}/></legend>{(["learner", "editor", "reviewer", "admin"] as Role[]).map(role => <label className={styles.check} key={role}><input type="checkbox" checked={roles.includes(role)} disabled={role === "learner"} onChange={e => setRoles(current => (["learner", "editor", "reviewer", "admin"] as Role[]).filter(r => r === role ? e.target.checked : current.includes(r)))} />{t(roleKeys[role],{})}</label>)}</fieldset>
      <label htmlFor="role-reason"><UiText notice={uiMessage("admin-users.reason.f81ab8",{})}/></label><textarea id="role-reason" required minLength={10} maxLength={1000} value={reason} onChange={e => setReason(e.target.value)} disabled={busy} /><button className="button" disabled={busy}><UiText notice={uiMessage("admin-users.save.roles.1a0378",{})}/></button></form>
      <form className={styles.form} onSubmit={e => { e.preventDefault(); void write("reset"); }}><h3><UiText notice={uiMessage("admin-users.reset.password.e0edfe",{})}/></h3><p className={styles.hint}><UiText notice={uiMessage("admin-users.verify.ownership.outside.this.website.before.resetting.an.account.0883de",{})}/></p>
        <label htmlFor="ownership"><UiText notice={uiMessage("admin-users.ownership.verification.7438d7",{})}/></label><textarea id="ownership" required minLength={10} maxLength={1000} value={ownershipNote} onChange={e => setOwnership(e.target.value)} disabled={busy} />
        <label htmlFor="temporary-password"><UiText notice={uiMessage("admin-users.temporary.password.b20862",{})}/></label><input id="temporary-password" type="password" autoComplete="new-password" required value={temporaryPassword} onChange={e => setTemporary(e.target.value)} disabled={busy} />
        <p className={styles.hint}><UiText notice={uiMessage("admin-users.the.reason.above.also.applies.to.this.reset.6ef8fa",{})}/></p><button className="button secondary" disabled={busy}><UiText notice={uiMessage("admin-users.reset.password.e0edfe",{})}/></button></form>
    </div>}</div>
    <FormMessage notice={error} error /><FormMessage notice={message} />
    {verify && <div className={styles.backdrop}><section role="dialog" aria-modal="true" aria-labelledby="verify-title" className={styles.dialog} onKeyDown={e => { if (e.key === "Escape" && !pending.current) { setVerify(false); setPassword(""); } if (e.key === "Tab") { const nodes = [...e.currentTarget.querySelectorAll<HTMLElement>("input:not(:disabled),button:not(:disabled)")]; const first = nodes[0], last = nodes[nodes.length - 1]; if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last?.focus(); } else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first?.focus(); } } }}><h2 id="verify-title"><UiText notice={uiMessage("admin-users.verify.your.password.0ed67a",{})}/></h2><p><UiText notice={uiMessage("admin-users.verification.lasts.five.minutes.you.will.submit.your.change.separ.332ca5",{})}/></p>
      <form className={styles.form} onSubmit={e => { e.preventDefault(); void write("reauth"); }}><label htmlFor="verify-password"><UiText notice={uiMessage("admin-users.your.password.bbda70",{})}/></label><input ref={verifyInput} id="verify-password" type="password" autoComplete="current-password" required value={password} onChange={e => setPassword(e.target.value)} disabled={busy} /><LanguageSwitch/><FormMessage notice={error} error /><button className="button" disabled={busy}><UiText notice={uiMessage("admin-users.verify.password.f226eb",{})}/></button><button className="button secondary" type="button" disabled={busy} onClick={() => { setVerify(false); setPassword(""); setError(null); }}><UiText notice={uiMessage("admin-users.cancel.19766e",{})}/></button></form></section></div>}
  </section>;
}

const roleKeys={learner:"auth.role.learner",editor:"auth.role.editor",reviewer:"auth.role.reviewer",admin:"auth.role.admin"} as const;
