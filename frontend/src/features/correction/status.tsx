"use client";
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
}) { return <span data-correction-status={status}>{correctionStatusLabels[status]}</span>; }
export type CorrectionErrorView = Pick<CorrectionRequestError, 'code' | 'message' | 'retryAt'>;
export function CorrectionState({ error, onRetry }: {
    error: CorrectionErrorView;
    onRetry?: () => void;
}) { const router = useRouter(); return <section className="content-state" role="alert"><p>{error.message}</p>{error.retryAt && <p>Try after {new Date(error.retryAt).toLocaleString('en')}</p>}{error.code === 'AUTHENTICATION_REQUIRED' ? <Link prefetch={false} href="/login">Sign in</Link> : ['PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN', 'REAUTHENTICATION_REQUIRED'].includes(error.code) ? <Link prefetch={false} href="/account">{error.code === 'REAUTHENTICATION_REQUIRED' ? 'Verify your password' : 'View account'}</Link> : <button type="button" className="button secondary" onClick={onRetry ?? (() => router.refresh())}>Reload page</button>}</section>; }
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
 const [verify,setVerify]=useState(false),[verified,setVerified]=useState(false),trigger=useRef<HTMLButtonElement>(null);
 return <div aria-live="polite">{verify && <Reauthenticate close={() => { setVerify(false); trigger.current?.focus(); }} verified={() => { setVerified(true); setVerify(false); trigger.current?.focus(); }}/>}{verified && command.pending !== null && <p>Password verified. Retry the preserved request.</p>}{command.busy && <p role="status">Checking and saving…</p>}{command.error && <div role="alert"><p>{command.confirmed ? 'Your change was saved. Retry the same request to load the latest status.' : command.error.code === 'SERVICE_UNAVAILABLE' ? 'No confirmation received. You can retry the same request.' : command.error.message}</p>{command.error.retryAt && <p>Try after {new Date(command.error.retryAt).toLocaleString('en')}</p>}{command.error.code === 'REAUTHENTICATION_REQUIRED' && <button ref={trigger} type="button" className="button secondary" disabled={command.busy} onClick={() => setVerify(true)}>Verify your password</button>}{!!command.pending && <div className="button-group"><button type="button" className="button secondary" disabled={command.busy} onClick={() => void command.retry()}>Retry same request</button>{!command.confirmed && <button type="button" className="button secondary" disabled={command.busy} onClick={command.clear}>Edit as a new request</button>}</div>}</div>}</div>; }
function Reauthenticate({ close, verified }: {
    close: () => void;
    verified: () => void;
}) {
    const account = useContext(CorrectionAccountContext), [password, setPassword] = useState(''), [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null), input = useRef<HTMLInputElement>(null), lock = useRef(false), live = useRef(true), controller = useRef<AbortController | null>(null), id = useId();
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
        }}><h2 id={id + '-title'}>Verify your password</h2><p>Verification lasts five minutes. Retry the preserved command separately.</p><form onSubmit={e => {
            e.preventDefault();
            if (lock.current || !account)
                return;
            const entered = password;
            setError(null);
            if (!validPrivateInput('reauth', { password: entered })) {
                setPassword('');
                setError('Passwords must contain 15–128 characters.');
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
                        setError(closed.message);
                }
            }).finally(() => {
                if (live.current) {
                    setPassword('');
                    setBusy(false);
                    lock.current = false;
                }
            });
        }}><label htmlFor={id + '-password'}>Your password</label><input ref={input} id={id + '-password'} type="password" autoComplete="current-password" required value={password} disabled={busy} onChange={e => setPassword(e.target.value)}/>{error && <p role="alert">{error}</p>}<div className={styles.actions}><button className="button" disabled={busy}>Verify password</button><button type="button" className="button secondary" disabled={busy} onClick={() => { setPassword(''); close(); }}>Cancel</button></div></form></section></div>;
}
