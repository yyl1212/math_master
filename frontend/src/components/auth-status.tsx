"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import { getAuthContext } from "@/lib/auth/client";
import type { AuthContext, AuthResult } from "@/lib/auth/types";
import styles from "@/styles/auth.module.css";
export function AuthStatus() {
  const path = usePathname(), [state, setState] = useState<AuthResult<AuthContext> | null>(null);
  useEffect(() => {
    let live = true, revision = 0;
    const refresh = () => { const current = ++revision; void getAuthContext().then(result => { if (live && revision === current) setState(result); }); };
    refresh(); window.addEventListener("math-master:auth-change", refresh);
    return () => { live = false; window.removeEventListener("math-master:auth-change", refresh); };
  }, [path]);
  return <div className={styles.status} aria-live="polite">{!state ? <span>Checking account…</span> : !state.ok ? <Link prefetch={false} href="/account">Accounts unavailable</Link> : state.data.user ? <>
    <Link prefetch={false} href="/account">{state.data.user.username}</Link>
    {!state.data.user.mustChangePassword&&state.data.user.roles.includes("learner")&&<><Link prefetch={false} href="/learn">My learning</Link><Link prefetch={false} href="/learning-history">Learning history</Link></>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("editor") && <Link prefetch={false} href="/editor">Edit content</Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("reviewer") && <Link prefetch={false} href="/review">Review content</Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("admin") && <Link prefetch={false} href="/admin/publications">Publish content</Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("editor") && <Link prefetch={false} href="/editor/questions">Write questions</Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("reviewer") && <Link prefetch={false} href="/review/questions">Review questions</Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("admin") && <Link prefetch={false} href="/admin/question-publications">Publish question bank</Link>}
    {!state.data.user.mustChangePassword && state.data.user.roles.includes("admin") && <Link prefetch={false} href="/admin/users">Manage users</Link>}
  </> : <Link prefetch={false} href="/login">Sign in</Link>}</div>;
}
