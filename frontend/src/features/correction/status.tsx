"use client";
import type {UiNotice} from "@/lib/i18n/types";
import {LanguageSwitch} from "@/components/language-switch";
import {UiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import { useContext, useEffect, useId, useRef, useState } from 'react';
import { authRequest } from '@/lib/auth/client';
import { currentCorrectionActor } from '@/lib/correction/client';
import { correctionAwait, withCorrectionDeadline } from '@/lib/correction/bytes';
import { validPrivateInput } from '@/lib/auth/schemas';
import { CorrectionRequestError } from '@/lib/correction/types';
import { CorrectionAccountContext } from './correction-account';
import styles from '@/styles/content.module.css';
import { useRouter } from 'next/navigation';
import type { ResultMetadata } from '@/lib/correction/types';
export const correctionStatusLabels = { corrected_passed: 'Corrected result · Passed', corrected_failed: 'Corrected result · Not passed', retake_required: 'Retake required', review_material: 'Review material', checked_unaffected: 'Checked · Unaffected', awaiting_review: 'Checking · Awaiting independent review' } satisfies Record<ResultMetadata['status'], string>;
export function CorrectionStatus({ status }: {
    status: ResultMetadata['status'];
}) { return <span data-correction-status={status}><UiEnum group="correction.status" value={status}/></span>; }
export type CorrectionErrorView = Pick<CorrectionRequestError, 'code' | 'message' | 'retryAt'>;
export function CorrectionState({ error, onRetry }: {
    error: CorrectionErrorView;
    onRetry?: () => void;
}) {
 const {t}=useUiI18n();
 const router = useRouter(); return <section className="content-state" role="alert"><p><UiText notice={uiError("correction",error)}/></p>{error.retryAt && <p><UiText notice={uiMessage("learning-status.try.after.value.ea24ae",{v0:uiValue(new Date(error.retryAt).toLocaleString('en'))})}/></p>}{error.code === 'AUTHENTICATION_REQUIRED' ? <Link prefetch={false} href="/login"><UiText notice={uiMessage("page.login",{})}/></Link> : ['PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN', 'REAUTHENTICATION_REQUIRED'].includes(error.code) ? <Link prefetch={false} href="/account">{error.code === 'REAUTHENTICATION_REQUIRED' ? t("admin-users.verify.your.password.0ed67a",{}) : t("auth-state.view.account.407143",{})}</Link> : <button type="button" className="button secondary" onClick={onRetry ?? (() => router.refresh())}><UiText notice={uiMessage("status.reload.page.437d0d",{})}/></button>}</section>; }
type CommandState = {
    busy: boolean;
    error: CorrectionRequestError | null;
    pending: unknown;
    confirmed: boolean;
    retry: () => Promise<void>;
    clear: () => void;
};
export function CorrectionCommandStatus({ command }: {
    command: CommandState;
}) {
 const {t}=useUiI18n();

 const [verify,setVerify]=useState(false),[verified,setVerified]=useState(false),trigger=useRef<HTMLButtonElement>(null);
 return <div aria-live="polite">{verify && <Reauthenticate close={() => { setVerify(false); trigger.current?.focus(); }} verified={() => { setVerified(true); setVerify(false); trigger.current?.focus(); }}/>}{verified && command.pending !== null && <p><UiText notice={uiMessage("status.password.verified.retry.the.preserved.request.15ada6",{})}/></p>}{command.busy && <p role="status"><UiText notice={uiMessage("status.checking.and.saving.ff6df1",{})}/></p>}{command.error && <div role="alert"><p>{command.confirmed ? t("status.your.change.was.saved.retry.the.same.request.to.load.the.latest.s.01d912",{}) : command.error.code === 'SERVICE_UNAVAILABLE' ? t("learning-status.no.confirmation.received.you.can.retry.the.same.request.e68eb0",{}) : <UiText notice={uiError("correction",command.error)}/>}</p>{command.error.retryAt && <p><UiText notice={uiMessage("learning-status.try.after.value.ea24ae",{v0:uiValue(new Date(command.error.retryAt).toLocaleString('en'))})}/></p>}{command.error.code === 'REAUTHENTICATION_REQUIRED' && <button ref={trigger} type="button" className="button secondary" disabled={command.busy} onClick={() => setVerify(true)}><UiText notice={uiMessage("admin-users.verify.your.password.0ed67a",{})}/></button>}{!!command.pending && <div className="button-group"><button type="button" className="button secondary" disabled={command.busy} onClick={() => void command.retry()}><UiText notice={uiMessage("learning-status.retry.same.request.16003a",{})}/></button>{!command.confirmed && <button type="button" className="button secondary" disabled={command.busy} onClick={command.clear}><UiText notice={uiMessage("status.edit.as.a.new.request.356c78",{})}/></button>}</div>}</div>}</div>; }
function Reauthenticate({ close, verified }: {
    close: () => void;
    verified: () => void;
}) {
 const {t}=useUiI18n();

    const account = useContext(CorrectionAccountContext), [password, setPassword] = useState(''), [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null), input = useRef<HTMLInputElement>(null), lock = useRef(false), live = useRef(true), controller = useRef<AbortController | null>(null), id = useId();
    useEffect(() => { input.current?.focus(); return () => { live.current = false; controller.current?.abort(); }; }, []);
    return <div className={styles.backdrop}><section className={styles.dialog} role="dialog" aria-modal="true" aria-labelledby={id + '-title'} onKeyDown={e => {
            if (e.key === 'Escape' && !lock.current) {
                setPassword('');
                close();
            }
            if (e.key === 'Tab') {
                const nodes = [...e.currentTarget.querySelectorAll<HTMLElement>('input:not(:disabled),button:not(:disabled)')], first = nodes[0], last = nodes[nodes.length - 1];
                if (e.shiftKey && document.activeElement === first) {
                    e.preventDefault();
                    last?.focus();
                }
                else if (!e.shiftKey && document.activeElement === last) {
                    e.preventDefault();
                    first?.focus();
                }
            }
        }}><h2 id={id + '-title'}><UiText notice={uiMessage("admin-users.verify.your.password.0ed67a",{})}/></h2><p><UiText notice={uiMessage("status.verification.lasts.five.minutes.retry.the.preserved.command.separ.043e8e",{})}/></p><form onSubmit={e => {
            e.preventDefault();
            if (lock.current || !account)
                return;
            const entered = password;
            setError(null);
            if (!validPrivateInput('reauth', { password: entered })) {
                setPassword('');
                setError(uiMessage("auth.input.password",{}));
                return;
            }
            lock.current = true;
            setBusy(true);
            const c = new AbortController();
            controller.current = c;
            void withCorrectionDeadline(c.signal, async (signal) => {
                await currentCorrectionActor({ actorId: account.actorId }, signal);
                const out = await correctionAwait(authRequest({ kind: 'reauth' }, { password: entered }), signal);
                if (!out.ok)
                    throw new CorrectionRequestError(out.code === 'AUTHENTICATION_REQUIRED' ? 'AUTHENTICATION_REQUIRED' : out.code === 'PASSWORD_CHANGE_REQUIRED' ? 'PASSWORD_CHANGE_REQUIRED' : 'SERVICE_UNAVAILABLE');
                await currentCorrectionActor({ actorId: account.actorId }, signal);
                if (live.current && !signal.aborted)
                    verified();
            }).catch(e => {
                if (live.current) {
                    const closed = e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
                    if (['AUTHENTICATION_REQUIRED', 'PASSWORD_CHANGE_REQUIRED'].includes(closed.code))
                        account.invalidate();
                    else
                        setError(uiError("correction",closed));
                }
            }).finally(() => {
                if (live.current) {
                    setPassword('');
                    setBusy(false);
                    lock.current = false;
                }
            });
        }}><label htmlFor={id + '-password'}><UiText notice={uiMessage("admin-users.your.password.bbda70",{})}/></label><input ref={input} id={id + '-password'} type="password" autoComplete="current-password" required value={password} disabled={busy} onChange={e => setPassword(e.target.value)}/><LanguageSwitch/>{error && <p role="alert"><UiText notice={error}/></p>}<div className={styles.actions}><button className="button" disabled={busy}><UiText notice={uiMessage("admin-users.verify.password.f226eb",{})}/></button><button type="button" className="button secondary" disabled={busy} onClick={() => { setPassword(''); close(); }}><UiText notice={uiMessage("admin-users.cancel.19766e",{})}/></button></div></form></section></div>;
}
