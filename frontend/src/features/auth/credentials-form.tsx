"use client";
import type {UiNotice} from "@/lib/i18n/types";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { authRequest, getAuthContext, notifyAuthChanged } from "@/lib/auth/client";
import { validPrivateInput } from "@/lib/auth/schemas";
import { FormMessage } from "./auth-state";
import styles from "@/styles/auth.module.css";
export function CredentialsForm({ mode }: { mode: "register" | "login" }) {
 const {t}=useUiI18n();

  const router = useRouter(), pending = useRef(false);
  const [username, setUsername] = useState(""), [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null);
  useEffect(() => { let live = true; void getAuthContext().then(result => { if (live && !result.ok) setError(uiError("auth",result)); }); return () => { live = false; }; }, []);
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (pending.current) return;
    pending.current = true; setBusy(true); setError(null);
    const input = { username, password };
    try {
      if (!validPrivateInput(mode, input)) { setError(uiMessage("auth.input.credentials",{})); return; }
      const result = await authRequest({ kind: mode }, input);
      if (!result.ok) { setError(uiError("auth",result)); return; }
      notifyAuthChanged(); router.replace(mode === "register" ? "/login" : "/account"); router.refresh();
    } finally { setPassword(""); pending.current = false; setBusy(false); }
  }
  return <section className={styles.credentials}><div className={styles.intro}><p className="eyebrow"><UiText notice={uiMessage("site-header.a.world.of.ideas.883eb5",{})}/></p>
    <h1>{mode === "register" ? t("credentials-form.make.room.for.mathematics.56b94d",{}) : t("credentials-form.welcome.back.dfdbca",{})}</h1>
    <p>{mode === "register" ? t("credentials-form.create.an.account.to.begin.your.next.chapter.16d5e3",{}) : t("credentials-form.sign.in.to.your.math.master.account.ced01b",{})}</p>
    <p><UiText notice={uiMessage("credentials-form.from.first.principles.to.new.frontiers.explore.a.connected.world..c2f0a1",{})}/></p>
    <Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("credentials-form.explore.the.knowledge.map.d12c1a",{})}/></Link></div>
    <div className={styles.card}><h2>{mode === "register" ? t("page.register",{}) : t("page.login",{})}</h2>
    <form className={styles.form} onSubmit={submit} aria-busy={busy}>
      <label htmlFor="username"><UiText notice={uiMessage("credentials-form.username.e3b89e",{})}/></label><input id="username" name="username" autoComplete="username" autoCapitalize="none" spellCheck={false} required minLength={3} maxLength={32} pattern="[A-Za-z0-9_]{3,32}" value={username} onChange={e => setUsername(e.target.value)} disabled={busy} />
      <label htmlFor="password"><UiText notice={uiMessage("credentials-form.password.e7cf3e",{})}/></label><input id="password" name="password" type="password" autoComplete={mode === "register" ? "new-password" : "current-password"} required value={password} onChange={e => setPassword(e.target.value)} disabled={busy} aria-describedby="password-hint" />
      <p id="password-hint" className={styles.hint}><UiText notice={uiMessage("credentials-form.15.128.characters.spaces.and.unicode.characters.are.welcome.66a06f",{})}/></p>
      <FormMessage notice={error} error />
      <button className="button" type="submit" disabled={busy}>{busy ? t("credentials-form.please.wait.4660a9",{}) : mode === "register" ? t("page.register",{}) : t("page.login",{})}</button>
    </form><p className={styles.switch}>{mode === "register" ? t("credentials-form.already.have.an.account.04a38d",{}) : t("credentials-form.new.to.math.master.733bb8",{})}<Link prefetch={false} href={mode === "register" ? "/login" : "/register"}>{mode === "register" ? t("page.login",{}) : t("page.register",{})}</Link></p></div>
  </section>;
}
