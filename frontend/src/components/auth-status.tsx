"use client";
import {useExperienceMode} from "@/features/study/experience-mode";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import Link from "next/link";
import { useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import { getAuthContext } from "@/lib/auth/client";
import type { AuthContext, AuthResult } from "@/lib/auth/types";
import styles from "@/styles/auth.module.css";
export function AuthStatus() {
  const path = usePathname(),mode=useExperienceMode(path), [state, setState] = useState<AuthResult<AuthContext> | null>(null);
  useEffect(() => {
    let live = true, revision = 0;
    const refresh = () => { const current = ++revision; void getAuthContext().then(result => { if (live && revision === current) setState(result); }); };
    refresh(); window.addEventListener("math-master:auth-change", refresh);
    return () => { live = false; window.removeEventListener("math-master:auth-change", refresh); };
  }, [path]);
  return <div className={styles.status} aria-live="polite">{!state ? <span><UiText notice={uiMessage("auth-status.checking.account.d18f41",{})}/></span> : !state.ok ? <Link prefetch={false} href="/account"><UiText notice={uiMessage("auth-status.accounts.unavailable.5b2487",{})}/></Link> : state.data.user ? <>
    <Link prefetch={false} href={state.data.user.mustChangePassword?"/account":"/learn"}>{state.data.user.username}</Link>
 {!state.data.user.mustChangePassword&&<Link prefetch={false} href="/account"><UiText notice={uiMessage("page.account",{})}/></Link>}
    {!state.data.user.mustChangePassword&&state.data.user.roles.includes("learner")&&<><Link prefetch={false} href="/learn"><UiText notice={uiMessage("auth-status.my.learning.c60fdf",{})}/></Link><Link prefetch={false} href="/learning-history"><UiText notice={uiMessage("auth-status.learning.history.35b7a5",{})}/></Link></>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("editor") && <Link prefetch={false} href="/editor"><UiText notice={uiMessage("auth-status.edit.content.f57e8e",{})}/></Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("reviewer") && <Link prefetch={false} href="/review"><UiText notice={uiMessage("auth-status.review.content.9e6e6c",{})}/></Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("admin") && <Link prefetch={false} href="/admin/publications"><UiText notice={uiMessage("auth-status.publish.content.35b610",{})}/></Link>}
    {mode==="legacy" && !state.data.user.mustChangePassword && state.data.user.roles.includes("editor") && <Link prefetch={false} href="/editor/questions"><UiText notice={uiMessage("auth-status.write.questions.4f6b81",{})}/></Link>}
    {mode==="legacy" && !state.data.user.mustChangePassword && state.data.user.roles.includes("reviewer") && <Link prefetch={false} href="/review/questions"><UiText notice={uiMessage("auth-status.review.questions.e2fd5a",{})}/></Link>}
    {mode==="legacy" && !state.data.user.mustChangePassword && state.data.user.roles.includes("admin") && <Link prefetch={false} href="/admin/question-publications"><UiText notice={uiMessage("auth-status.publish.question.bank.545b0c",{})}/></Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("admin") && <Link prefetch={false} href="/admin/users"><UiText notice={uiMessage("auth-status.manage.users.58606e",{})}/></Link>}
  </> : <Link prefetch={false} href="/login"><UiText notice={uiMessage("auth-status.sign.in.bfd402",{})}/></Link>}</div>;
}
