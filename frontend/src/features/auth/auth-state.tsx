"use client";
import type {UiNotice} from "@/lib/i18n/types";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getAuthContext } from "@/lib/auth/client";
import styles from "@/styles/auth.module.css";
export function FormMessage(props:({notice:UiNotice|null;message?:never}|{message:string|null;notice?:never})&{error?:boolean}) {
  const ref=useRef<HTMLParagraphElement>(null);const error=props.error??false;
  const value=props.notice??props.message;
  useEffect(()=>{if(value&&error)ref.current?.focus();},[value,error]);
  const notice=props.notice??(props.message?{kind:"literal" as const,text:props.message}:null);
  return notice?<p ref={ref} role={error?"alert":"status"} tabIndex={error?-1:undefined} className={error?styles.error:styles.notice}><UiText notice={notice}/></p>:null;
}
export function AuthState({ kind }: { kind: "anonymous" | "forbidden" | "unavailable" | "invalid-cookie" }) {
  const router = useRouter(), pending = useRef(false);
  const [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null);
  async function recover() {
    if (pending.current) return;
    pending.current = true; setBusy(true);
    try {
      const result = await getAuthContext();
      if (result.ok || result.code === "INVALID_COOKIE") router.refresh();
      else setError(uiError("auth",{code:"AUTH_NOT_CONFIGURED"}));
    } finally { pending.current = false; setBusy(false); }
  }
  const text={anonymous:uiMessage("auth.state.anonymous",{}),forbidden:uiError("auth",{code:"FORBIDDEN"}),unavailable:uiError("auth",{code:"AUTH_NOT_CONFIGURED"}),"invalid-cookie":uiMessage("auth.state.invalidCookie",{})}[kind];
  return <section className="content-state"><p className="eyebrow"><UiText notice={uiMessage("account-panel.your.account.cee7fd",{})}/></p><h1><UiText notice={text}/></h1>
    {kind === "anonymous" && <Link className="button" prefetch={false} href="/login"><UiText notice={uiMessage("page.login",{})}/></Link>}
    {kind === "forbidden" && <Link className="button secondary" prefetch={false} href="/account"><UiText notice={uiMessage("auth-state.view.account.407143",{})}/></Link>}
    {kind === "invalid-cookie" && <button className="button" disabled={busy} onClick={recover}><UiText notice={uiMessage("auth-state.clear.sign.in.cookie.b7ae51",{})}/></button>}
    {kind === "unavailable" && <button className="button secondary" onClick={() => router.refresh()}><UiText notice={uiMessage("content-state.try.again.d8b839",{})}/></button>}
    <FormMessage notice={error} error />
    <Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("auth-state.explore.mathematics.088a8c",{})}/></Link>
  </section>;
}
