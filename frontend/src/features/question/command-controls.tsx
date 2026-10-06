"use client";
import {uiError} from "@/lib/i18n/errors";
import {LanguageSwitch} from "@/components/language-switch";
import type {UiNotice} from "@/lib/i18n/types";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import { useEffect, useRef, useState } from "react";
import { authRequest } from "@/lib/auth/client";
import { FormMessage } from "@/features/auth/auth-state";
import { questionInputError, questionPolicies } from "@/lib/question/schemas";
import type { QuestionRoute, QuestionResult } from "@/lib/question/types";
import { createQuestionPendingCommand, type QuestionPendingCommand } from "./pending-command";
import styles from "@/styles/question.module.css";
export function useQuestionCommand() {
 const {t}=useUiI18n();

    const pending = useRef<QuestionPendingCommand | null>(null), busyRef = useRef(false), success = useRef<((data: unknown) => void | Promise<void>) | null>(null), trigger = useRef<HTMLElement | null>(null), retryButton = useRef<HTMLButtonElement | null>(null);
    const [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null), [message, setMessage] = useState<UiNotice | null>(null), [retry, setRetry] = useState(false), [verify, setVerify] = useState(false), [failure, setFailure] = useState<Extract<QuestionResult<never>, {
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
            setError(uiError("question",result));
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
        setError(uiError("question",{code:e}));
        return;
    } trigger.current = document.activeElement as HTMLElement; const command = createQuestionPendingCommand(route, input); pending.current = command; success.current = onSuccess; setMessage(null); await execute(command); }
    function clearPending() { if (busyRef.current)
        return; pending.current = null; setRetry(false); setFailure(null); setError(null); }
    const controls = <><FormMessage notice={error} error/><FormMessage notice={message}/>{busy && <button className="button secondary" type="button" onClick={() => pending.current?.cancel()}><UiText notice={uiMessage("command-controls.stop.waiting.d4af24",{})}/></button>}{retry && <section className={styles.notice}><p>{failure?.status === 503 ? t("command-controls.the.result.is.unconfirmed.stopping.or.timing.out.does.not.prove.t.fba207",{}) : t("command-controls.the.previous.command.is.preserved.03b9de",{})}</p><p><UiText notice={uiMessage("command-controls.retry.sends.the.same.input.and.request.key.no.write.is.retried.au.4ced2e",{})}/></p>{failure?.retryAfter && <p><UiText notice={uiMessage("command-controls.try.again.after.value.seconds.a723d0",{v0:uiValue(failure.retryAfter)})}/></p>}<div className={styles.actions}><button ref={retryButton} className="button secondary" disabled={busy || verify} onClick={() => { if (pending.current)
        void execute(pending.current); }}><UiText notice={uiMessage("command-controls.retry.previous.request.34b089",{})}/></button><button className="button secondary" disabled={busy} onClick={clearPending}><UiText notice={uiMessage("command-controls.discard.local.pending.request.14700a",{})}/></button></div></section>}{verify && <PasswordDialog close={close} verified={() => { close(); setMessage(uiMessage("command-controls.password.verified.retry.the.preserved.command.when.ready.aef9f8",{})); }}/>}</>;
    return { busy, blocked: busy || retry, run, controls, message: (v:UiNotice|string|null)=>setMessage(typeof v==="string"?{kind:"literal",text:v}:v), error: (v:UiNotice|string|null)=>setError(typeof v==="string"?{kind:"literal",text:v}:v), failure, clearPending };
}
function PasswordDialog({ close, verified }: {
    close: () => void;
    verified: () => void;
}) {
 const {t}=useUiI18n();

    const [password, setPassword] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null), input = useRef<HTMLInputElement>(null), lock = useRef(false);
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
    } }}><h2 id="question-password-title"><UiText notice={uiMessage("admin-users.verify.your.password.0ed67a",{})}/></h2><p><UiText notice={uiMessage("command-controls.verification.lasts.five.minutes.the.preserved.command.is.sent.onl.64a5cc",{})}/></p><form onSubmit={async (e) => { e.preventDefault(); if (lock.current)
        return; lock.current = true; setBusy(true); try {
        const result = await authRequest({ kind: "reauth" }, { password });
        setPassword("");
        if (result.ok)
            verified();
        else
            setError(uiError("auth",result));
    }
    finally {
        setPassword("");
        lock.current = false;
        setBusy(false);
    } }}><label htmlFor="question-password"><UiText notice={uiMessage("admin-users.your.password.bbda70",{})}/></label><input ref={input} id="question-password" type="password" autoComplete="current-password" required disabled={busy} value={password} onChange={e => setPassword(e.target.value)}/><LanguageSwitch/><FormMessage notice={error} error/><div className={styles.actions}><button className="button" disabled={busy}><UiText notice={uiMessage("admin-users.verify.password.f226eb",{})}/></button><button className="button secondary" type="button" disabled={busy} onClick={cancel}><UiText notice={uiMessage("admin-users.cancel.19766e",{})}/></button></div></form></section></div>;
}
