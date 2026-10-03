"use client";
import { useContext, useEffect, useRef, useState } from 'react';
import { z } from 'zod';
import { correctionClient, currentCorrectionActor } from '@/lib/correction/client';
import { correctionAwait, validateCorrectionBytes, withCorrectionDeadline } from '@/lib/correction/bytes';
import { CorrectionRequestError, type Envelope, type Receipt } from '@/lib/correction/types';
import { uuidSchema, planRefSchema, caseInputSchema, planInputSchema, submitInputSchema, decisionInputSchema } from '@/lib/correction/schemas';
import { CorrectionAccountContext } from './correction-account';
const commandSchema = z.discriminatedUnion('kind', [
    z.object({ kind: z.literal('createCase'), input: caseInputSchema }).strict(),
    z.object({ kind: z.literal('createPlan'), caseId: uuidSchema, input: planInputSchema }).strict(),
    z.object({ kind: z.literal('updatePlan'), ref: planRefSchema, input: planInputSchema }).strict(),
    z.object({ kind: z.literal('submitPlan'), ref: planRefSchema, input: submitInputSchema }).strict(),
    z.object({ kind: z.literal('decidePlan'), ref: planRefSchema, input: decisionInputSchema }).strict(),
    z.object({ kind: z.literal('retryJob'), jobId: uuidSchema, input: submitInputSchema }).strict()
]);
export type CorrectionCommandInput = z.infer<typeof commandSchema>;
export type PendingCorrectionCommand = CorrectionCommandInput & {
    actorId: string;
    key: string;
};
export function freezePrivate<T>(value: T): T {
    if (value !== null && typeof value === 'object') {
        for (const v of Object.values(value))
            freezePrivate(v);
        Object.freeze(value);
    }
    return value;
}
export function createPendingCorrectionCommand(actorId: string, command: CorrectionCommandInput): PendingCorrectionCommand {
    try {
        uuidSchema.parse(actorId);
        const parsed = commandSchema.parse(command);
        validateCorrectionBytes(new TextEncoder().encode(JSON.stringify(parsed.input)), parsed.kind);
        return freezePrivate({ ...parsed, actorId, key: crypto.randomUUID() });
    }
    catch {
        throw new CorrectionRequestError('INVALID_REQUEST');
    }
}
function send(command: PendingCorrectionCommand, signal: AbortSignal) {
    const access = { actorId: command.actorId, key: command.key, signal };
    switch (command.kind) {
        case 'createCase': return correctionClient.createCase(command.input, access);
        case 'createPlan': return correctionClient.createPlan(command.caseId, command.input, access);
        case 'updatePlan': return correctionClient.updatePlan(command.ref, command.input, access);
        case 'submitPlan': return correctionClient.submitPlan(command.ref, command.input, access);
        case 'decidePlan': return correctionClient.decidePlan(command.ref, command.input, access);
        case 'retryJob': return correctionClient.retryJob(command.jobId, command.input, access);
    }
}
export function usePrivatePending<P extends {
    actorId: string;
    key: string;
}, R, I>(actorId: string, make: (input: I) => P, send: (pending: P, signal: AbortSignal) => Promise<Envelope<R>>, onSuccess: (receipt: R, signal: AbortSignal) => Promise<void>) {
    const account = useContext(CorrectionAccountContext), [pending, setPending] = useState<P | null>(null), [error, setError] = useState<CorrectionRequestError | null>(null), [busy, setBusy] = useState(false), [confirmed, setConfirmed] = useState(false);
    const live = useRef(true), generation = useRef(0), inFlight = useRef(false), snapshot = useRef<P | null>(null), controller = useRef<AbortController | null>(null), success = useRef(onSuccess);
    success.current = onSuccess;
    const clear = () => { generation.current++; controller.current?.abort(); inFlight.current = false; snapshot.current = null; setPending(null); setError(null); setBusy(false); setConfirmed(false); };
    useEffect(() => { live.current = true; const changed = () => clear(); window.addEventListener('math-master:auth-change', changed); return () => { live.current = false; generation.current++; controller.current?.abort(); snapshot.current = null; window.removeEventListener('math-master:auth-change', changed); }; }, [actorId]);
    const perform = async (command: P) => {
        if (inFlight.current || account?.checking)
            return;
        if (!account || account.actorId !== actorId || command.actorId !== actorId) {
            clear();
            account?.invalidate();
            return;
        }
        inFlight.current = true;
        snapshot.current = command;
        setPending(command);
        setBusy(true);
        setError(null);
        const c = new AbortController(), n = ++generation.current;
        controller.current = c;
        const valid = () => live.current && n === generation.current;
        try {
            await withCorrectionDeadline(c.signal, async (signal) => {
                await currentCorrectionActor({ actorId }, signal);
                if (!valid())
                    return;
                const out = await correctionAwait(send(command, signal), signal);
                if (!valid())
                    return;
                if (out.actorId !== actorId) {
                    clear();
                    account.invalidate();
                    return;
                }
                setConfirmed(true);
                await correctionAwait(success.current(out.data, signal), signal);
                if (valid() && !signal.aborted) {
                    snapshot.current = null;
                    setPending(null);
                    setConfirmed(false);
                }
            });
        }
        catch (e) {
            if (valid()) {
                const closed = e instanceof CorrectionRequestError ? e : new CorrectionRequestError();
                if (['AUTHENTICATION_REQUIRED', 'PASSWORD_CHANGE_REQUIRED', 'FORBIDDEN'].includes(closed.code)) {
                    clear();
                    account.invalidate();
                }
                else
                    setError(closed);
            }
        }
        finally {
            if (controller.current === c)
                controller.current = null;
            if (valid()) {
                inFlight.current = false;
                setBusy(false);
            }
        }
    };
    const run = async (input: I) => {
        if (inFlight.current || snapshot.current || account?.checking)
            return;
        try {
            await perform(make(input));
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError('INVALID_REQUEST'));
        }
    };
    const retry = async () => {
        if (snapshot.current)
            await perform(snapshot.current);
    };
    return { run, retry, clear, busy: busy || !!account?.checking, pending, confirmed, error };
}
export function useCorrectionCommand(actorId: string, onSuccess: (receipt: Receipt, signal: AbortSignal) => Promise<void>) { return usePrivatePending(actorId, (input: CorrectionCommandInput) => createPendingCorrectionCommand(actorId, input), send, onSuccess); }
