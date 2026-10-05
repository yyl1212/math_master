"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import { getAuthContext } from '@/lib/auth/client';
import { correctionAwait, withCorrectionDeadline } from '@/lib/correction/bytes';
import { CorrectionRequestError, type ReadAccess, type Envelope } from '@/lib/correction/types';
import type { Role } from '@/lib/auth/types';
import { CorrectionState } from './status';
type Account = {
    actorId: string;
    roles: Role[];
    checking: boolean;
    invalidate: () => void;
};
export const CorrectionAccountContext = createContext<Account | null>(null);
type Props = {
    actorId: string;
    management?: boolean;
    children: ReactNode;
};
export function CorrectionAccountProvider(props: Props) { return <AccountBoundary key={props.actorId} {...props}/>; }
function AccountBoundary({ actorId, management = false, children }: Props) {
    const [everVerified, setEverVerified] = useState(false), [checking, setChecking] = useState(true), [roles, setRoles] = useState<Role[]>([]), [error, setError] = useState<CorrectionRequestError | null>(null), [attempt, setAttempt] = useState(0);
    const serverChildren=useRef(children);
    const invalid = useRef(false), [discarded, setDiscarded] = useState(false);
    const invalidate = useCallback(() => { invalid.current = true; setDiscarded(true); setChecking(false); setError(new CorrectionRequestError('AUTHENTICATION_REQUIRED')); }, []);
    useEffect(() => {
        // A new SSR snapshot may recover after invalidation; focus alone cannot.
        if(serverChildren.current!==children&&invalid.current){invalid.current=false;setDiscarded(false);setEverVerified(false);setError(null)}
        serverChildren.current=children;
        let live = true, revision = 0, controller: AbortController | undefined;
        const verify = () => {
            if (invalid.current)
                return;
            const n = ++revision;
            controller?.abort();
            controller = new AbortController();
            setChecking(true);
            void withCorrectionDeadline(controller.signal, async (signal) => {
                const proof = await correctionAwait(getAuthContext(true), signal);
                if (!live || n !== revision || invalid.current)
                    return;
                if (!proof.ok)
                    throw new CorrectionRequestError(proof.code === 'AUTHENTICATION_REQUIRED' ? 'AUTHENTICATION_REQUIRED' : proof.code === 'PASSWORD_CHANGE_REQUIRED' ? 'PASSWORD_CHANGE_REQUIRED' : 'SERVICE_UNAVAILABLE');
                const user = proof.data.user;
                if (!user || user.id !== actorId) {
                    invalidate();
                    return;
                }
                if (user.mustChangePassword)
                    throw new CorrectionRequestError('PASSWORD_CHANGE_REQUIRED');
                if (management && !user.roles.some(r => ['editor', 'reviewer', 'admin'].includes(r)))
                    throw new CorrectionRequestError('FORBIDDEN');
                setRoles(user.roles);
                setError(null);
                setEverVerified(true);
                setChecking(false);
            }).catch(e => {
                if (!live || n !== revision || invalid.current)
                    return;
                const closed = e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
                if (['AUTHENTICATION_REQUIRED', 'PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN'].includes(closed.code)) {
                    invalid.current = true;
                    setDiscarded(true);
                }
                setChecking(false);
                setError(closed);
            });
        };
        const changed = () => { revision++; controller?.abort(); invalidate(); }, visible = () => {
            if (document.visibilityState === 'visible')
                verify();
        };
        verify();
        window.addEventListener('math-master:auth-change', changed);
        window.addEventListener('focus', verify);
        document.addEventListener('visibilitychange', visible);
        let channel: BroadcastChannel | undefined;
        try {
            if (typeof BroadcastChannel !== 'undefined') {
                channel = new BroadcastChannel('math-master-auth');
                channel.onmessage = e => {
                    if (e.data === 'changed')
                        changed();
                };
            }
        }
        catch { /* Every command also checks current identity. */ }
        return () => { live = false; revision++; controller?.abort(); window.removeEventListener('math-master:auth-change', changed); window.removeEventListener('focus', verify); document.removeEventListener('visibilitychange', visible); channel?.close(); };
    }, [actorId, management, children, attempt, invalidate]);
    const blocked = checking || error !== null;
    return <>{checking && <p role="status"><UiText notice={uiMessage("correction-account.checking.your.correction.account.da4282",{})}/></p>}{error && <CorrectionState error={error} onRetry={() => setAttempt(n => n + 1)}/>}{everVerified && !discarded && <div hidden={blocked} inert={blocked}><CorrectionAccountContext.Provider value={{ actorId, roles, checking: blocked, invalidate }}>{children}</CorrectionAccountContext.Provider></div>}</>;
}
// Keep same-account drafts mounted during verification; actor changes replace the boundary.
export function useCorrectionRead() {
    const account = useContext(CorrectionAccountContext), live = useRef(true), generation = useRef(0), active = useRef<AbortController | null>(null);
    useEffect(() => { live.current = true; return () => { live.current = false; generation.current++; active.current?.abort(); }; }, []);
    return async <T,>(read: (access: ReadAccess) => Promise<Envelope<T>>, signal?: AbortSignal): Promise<Envelope<T> | null> => {
        if (!account || account.checking)
            return null;
        active.current?.abort();
        const c = new AbortController(), n = ++generation.current;
        active.current = c;
        const external = signal ? AbortSignal.any([signal, c.signal]) : c.signal;
        try {
            const out = await withCorrectionDeadline(external, s => correctionAwait(read({ actorId: account.actorId, signal: s }), s));
            if (!live.current || n !== generation.current || c.signal.aborted || signal?.aborted)
                return null;
            if (out.actorId !== account.actorId) {
                account.invalidate();
                return null;
            }
            return out;
        }
        catch (e) {
            if (!live.current || n !== generation.current || c.signal.aborted || signal?.aborted)
                return null;
            const closed = e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
            if (['AUTHENTICATION_REQUIRED', 'PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN'].includes(closed.code)) {
                account.invalidate();
                return null;
            }
            throw closed;
        }
    };
}
