"use client";
import { useEffect, useRef, useState } from "react";
import { authRequest } from "@/lib/auth/client";
import { FormMessage } from "@/features/auth/auth-state";
import { questionInputError, questionPolicies } from "@/lib/question/schemas";
import type { QuestionRoute, QuestionResult } from "@/lib/question/types";
import { createQuestionPendingCommand, type QuestionPendingCommand } from "./pending-command";
import styles from "@/styles/question.module.css";
export function useQuestionCommand() {
    const pending = useRef<QuestionPendingCommand | null>(null), busyRef = useRef(false), success = useRef<((data: unknown) => void | Promise<void>) | null>(null), trigger = useRef<HTMLElement | null>(null), retryButton = useRef<HTMLButtonElement | null>(null);
    const [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null), [message, setMessage] = useState<string | null>(null), [retry, setRetry] = useState(false), [verify, setVerify] = useState(false), [failure, setFailure] = useState<Extract<QuestionResult<never>, {
        ok: false;
    }> | null>(null);
    useEffect(() => () => pending.current?.cancel(), []);
    function close() { setVerify(false); setTimeout(() => { const origin = trigger.current; if(origin?.isConnected && !origin.matches(":disabled")) origin.focus(); else retryButton.current?.focus(); }, 0); }
    async function execute(command: QuestionPendingCommand) { if (busyRef.current)
        return; busyRef.current = true; setBusy(true); setError(null); setFailure(null); try {
        const result = await command.execute();
        if (result.ok) {
            pending.current = null;
            setRetry(false);
            await success.current?.(result.data);
        }
        else {
            setFailure(result);
            setError(result.message);
            setRetry([428, 429, 503].includes(result.status));
            if (result.code === "REAUTHENTICATION_REQUIRED")
                setVerify(true);
        }
    }
    finally {
        busyRef.current = false;
        setBusy(false);
    } }
    async function run(route: QuestionRoute, input: unknown, onSuccess: (v: unknown) => void | Promise<void>) { if (busyRef.current || retry)
        return; const e = questionInputError(route.kind, input); if (e) {
        setError(questionPolicies[e].message);
        return;
    } trigger.current = document.activeElement as HTMLElement; const command = createQuestionPendingCommand(route, input); pending.current = command; success.current = onSuccess; setMessage(null); await execute(command); }
    function clearPending() { if (busyRef.current)
        return; pending.current = null; setRetry(false); setFailure(null); setError(null); }
    const controls = <><FormMessage message={error} error/><FormMessage message={message}/>{busy && <button className="button secondary" type="button" onClick={() => pending.current?.cancel()}>Stop waiting</button>}{retry && <section className={styles.notice}><p>{failure?.status === 503 ? "The result is unconfirmed. Stopping or timing out does not prove that the server rejected the change." : "The previous command is preserved."}</p><p>Retry sends the same input and request key. No write is retried automatically.</p>{failure?.retryAfter && <p>Try again after {failure.retryAfter} seconds.</p>}<div className={styles.actions}><button ref={retryButton} className="button secondary" disabled={busy || verify} onClick={() => { if (pending.current)
        void execute(pending.current); }}>Retry previous request</button><button className="button secondary" disabled={busy} onClick={clearPending}>Discard local pending request</button></div></section>}{verify && <PasswordDialog close={close} verified={() => { close(); setMessage("Password verified. Retry the preserved command when ready."); }}/>}</>;
    return { busy, blocked: busy || retry, run, controls, error: setError, message: setMessage, failure, clearPending };
}
function PasswordDialog({ close, verified }: {
    close: () => void;
    verified: () => void;
}) {
    const [password, setPassword] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null), input = useRef<HTMLInputElement>(null), lock = useRef(false);
    useEffect(() => { input.current?.focus(); }, []);
    const cancel = () => { if (!lock.current) {
        setPassword("");
        close();
    } };
    return <div className={styles.backdrop}><section className={styles.dialog} role="dialog" aria-modal="true" aria-labelledby="question-password-title" onKeyDown={e => { if (e.key === "Escape")
        cancel(); if (e.key === "Tab") {
        const nodes = [...e.currentTarget.querySelectorAll<HTMLElement>("input:not(:disabled),button:not(:disabled)")];
        const first = nodes[0], last = nodes[nodes.length - 1];
        if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last?.focus();
        }
        else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first?.focus();
        }
    } }}><h2 id="question-password-title">Verify your password</h2><p>Verification lasts five minutes. The preserved command is sent only when you choose Retry.</p><form onSubmit={async (e) => { e.preventDefault(); if (lock.current)
        return; lock.current = true; setBusy(true); try {
        const result = await authRequest({ kind: "reauth" }, { password });
        setPassword("");
        if (result.ok)
            verified();
        else
            setError(result.message);
    }
    finally {
        setPassword("");
        lock.current = false;
        setBusy(false);
    } }}><label htmlFor="question-password">Your password</label><input ref={input} id="question-password" type="password" autoComplete="current-password" required disabled={busy} value={password} onChange={e => setPassword(e.target.value)}/><FormMessage message={error} error/><div className={styles.actions}><button className="button" disabled={busy}>Verify password</button><button className="button secondary" type="button" disabled={busy} onClick={cancel}>Cancel</button></div></form></section></div>;
}
