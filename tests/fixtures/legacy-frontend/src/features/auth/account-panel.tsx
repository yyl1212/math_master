"use client";
import {useUiI18n} from "@/lib/i18n/provider";
import type {UiNotice} from "@/lib/i18n/types";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import Link from "next/link";
import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { authRequest, notifyAuthChanged } from "@/lib/auth/client";
import { validPrivateInput } from "@/lib/auth/schemas";
import type { User } from "@/lib/auth/types";
import { FormMessage } from "./auth-state";
import styles from "@/styles/auth.module.css";
export function AccountPanel({ user }: { user: User }) {
 const {t}=useUiI18n();
  const router = useRouter(), pending = useRef(false);
  const [currentPassword, setCurrent] = useState(""), [newPassword, setNew] = useState("");
  const [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null);
  async function write(kind: "password" | "logout" | "logout-all") {
    if (pending.current) return;
    pending.current = true; setBusy(true); setError(null);
    const input = kind === "password" ? { currentPassword, newPassword } : {};
    try {
      if (!validPrivateInput(kind, input)) { setError(uiMessage("auth.input.password",{})); return; }
      const result = await authRequest({ kind }, input);
      if (!result.ok) { setError(uiError("auth",result)); return; }
      notifyAuthChanged(); router.replace(kind === "password" ? "/login" : "/"); router.refresh();
    } finally { setCurrent(""); setNew(""); pending.current = false; setBusy(false); }
  }
  return <section className={styles.account}><div className="page-heading"><p className="eyebrow"><UiText notice={uiMessage("account-panel.your.account.cee7fd",{})}/></p><h1>{user.username}</h1><p>{user.roles.map(role=>t(roleKeys[role],{})).join(" · ")}</p></div>
    {user.mustChangePassword && <p className={styles.notice}><UiText notice={uiMessage("account-panel.change.your.password.to.continue.1c1b58",{})}/></p>}
    <div className={styles.card}><h2><UiText notice={uiMessage("account-panel.change.password.3f9c99",{})}/></h2><p className={styles.hint}><UiText notice={uiMessage("account-panel.changing.your.password.signs.you.out.on.every.device.sign.in.agai.c2a7e1",{})}/></p>
      <form className={styles.form} onSubmit={e => { e.preventDefault(); void write("password"); }} aria-busy={busy}>
        <label htmlFor="current-password"><UiText notice={uiMessage("account-panel.current.password.72ed2b",{})}/></label><input id="current-password" type="password" autoComplete="current-password" required value={currentPassword} onChange={e => setCurrent(e.target.value)} disabled={busy} />
        <label htmlFor="new-password"><UiText notice={uiMessage("account-panel.new.password.3dd9df",{})}/></label><input id="new-password" type="password" autoComplete="new-password" required value={newPassword} onChange={e => setNew(e.target.value)} disabled={busy} aria-describedby="new-password-hint" />
        <p id="new-password-hint" className={styles.hint}><UiText notice={uiMessage("credentials-form.15.128.characters.spaces.and.unicode.characters.are.welcome.66a06f",{})}/></p>
        <FormMessage notice={error} error /><button className="button" disabled={busy}><UiText notice={uiMessage("account-panel.change.password.3f9c99",{})}/></button>
      </form></div>
    <div className={styles.actions}><button className="button secondary" disabled={busy} onClick={() => void write("logout")}><UiText notice={uiMessage("account-panel.sign.out.48f0d3",{})}/></button>
      {!user.mustChangePassword && <button className="button secondary" disabled={busy} onClick={() => void write("logout-all")}><UiText notice={uiMessage("account-panel.sign.out.everywhere.af180f",{})}/></button>}
      {!user.mustChangePassword && user.roles.includes("admin") && <Link className="button secondary" prefetch={false} href="/admin/users"><UiText notice={uiMessage("auth-status.manage.users.58606e",{})}/></Link>}
    </div></section>;
}

const roleKeys={learner:"auth.role.learner",editor:"auth.role.editor",reviewer:"auth.role.reviewer",admin:"auth.role.admin"} as const;
