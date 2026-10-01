"use client";
import { useState, useRef, useEffect } from "react";
import { authRequest } from "@/lib/auth/client";
import { createPendingCommand, retryPendingCommand, type PendingCommand } from "./pending-command";
import { contentInputError, contentPolicies } from "@/lib/content/schemas";
import { FormMessage } from "@/features/auth/auth-state";
import type { ContentRoute, ContentResult } from "@/lib/content/types";
import styles from "@/styles/content.module.css";
export function useContentCommand() {
    const busyRef = useRef(false), pending = useRef<PendingCommand | null>(null), success = useRef<((v: unknown) => void | Promise<void>) | null>(null), trigger = useRef<HTMLElement | null>(null);
    const [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null), [message, setMessage] = useState<string | null>(null), [retry, setRetry] = useState(false), [verify, setVerify] = useState(false);
    const [failure, setFailure] = useState<Extract<ContentResult<never>, {
        ok: false;
    }> | null>(null);
    async function execute(command: PendingCommand) {
        if (busyRef.current)
            return;
        trigger.current = document.activeElement as HTMLElement;
        busyRef.current = true;
        setBusy(true);
        setError(null);
        setMessage(null);
        setFailure(null);
        try {
            const result = await retryPendingCommand(command);
            if (result.ok) {
                pending.current = null;
                setRetry(false);
                await success.current?.(result.data);
            }
            else {
                setFailure(result);
                setError(result.message);
                setRetry(result.status === 503 || result.status === 429);
                if (result.code === "REAUTHENTICATION_REQUIRED") {
                    setVerify(true);
                }
            }
        }
        finally {
            busyRef.current = false;
            setBusy(false);
        }
    }
    async function run(route: ContentRoute, input: unknown, onSuccess: (v: unknown) => void | Promise<void>) { if (busyRef.current)
        return; const error = contentInputError(route.kind, input); if (error) {
        setError(contentPolicies[error].message);
        return;
    } const command = createPendingCommand(route, input); pending.current = command; success.current = onSuccess; await execute(command); }
    function close() { setVerify(false); setError(null); setTimeout(() => trigger.current?.focus(), 0); }
    const controls = <><FormMessage message={error} error/><FormMessage message={message}/>{retry && <div className={styles.actions}><p>Retry sends the original input and request key.</p><button className="button secondary" disabled={busy} onClick={() => { if (pending.current)
        void execute(pending.current); }}>Retry previous request</button><button className="button secondary" disabled={busy} onClick={() => { pending.current = null; setRetry(false); setError(null); }}>Discard pending request</button></div>}{verify && <PasswordDialog close={close} verified={() => { close(); setMessage("Password verified. Submit your change again."); }}/>}</>;
    return { busy, run, controls, message: setMessage, error: setError, failure, clearPending() { if (!busyRef.current) {
            pending.current = null;
            setRetry(false);
        } } };
}
function PasswordDialog({ close, verified }: {
    close: () => void;
    verified: () => void;
}) {
    const [password, setPassword] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null);
    const lock = useRef(false), input = useRef<HTMLInputElement>(null);
    useEffect(() => { input.current?.focus(); return () => { }; }, []);
    function cancel() { if (lock.current)
        return; setPassword(""); close(); }
    return <div className={styles.backdrop}><section role="dialog" aria-modal="true" aria-labelledby="content-verify-title" className={styles.dialog} onKeyDown={e => { if (e.key === "Escape")
        cancel(); if (e.key === "Tab") {
        const nodes = [...e.currentTarget.querySelectorAll<HTMLElement>("input:not(:disabled),button:not(:disabled),[tabindex='-1']")];
        const first = nodes[0], last = nodes[nodes.length - 1];
        if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last?.focus();
        }
        else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first?.focus();
        }
    } }}><h2 id="content-verify-title">Verify your password</h2><p>Verification lasts five minutes. Submit your change separately.</p><form onSubmit={async (e) => { e.preventDefault(); if (lock.current)
        return; lock.current = true; setBusy(true); try {
        const result = await authRequest({ kind: "reauth" }, { password });
        setPassword("");
        if (result.ok)
            verified();
        else
            setError(result.message);
    }
    finally {
        lock.current = false;
        setBusy(false);
        setPassword("");
    } }}><label htmlFor="content-verify-password">Your password</label><input ref={input} id="content-verify-password" type="password" autoComplete="current-password" required value={password} disabled={busy} onChange={e => setPassword(e.target.value)}/><FormMessage message={error} error/><div className={styles.actions}><button className="button" disabled={busy}>Verify password</button><button className="button secondary" type="button" disabled={busy} onClick={cancel}>Cancel</button></div></form></section></div>;
}
