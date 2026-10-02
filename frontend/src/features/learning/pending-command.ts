"use client";
import { useContext, useEffect, useRef, useState } from "react";
import { LearningAccountContext } from "./learning-account";
import { getAuthContext } from "@/lib/auth/client";
import { requestLearning, bindLearningInput } from "@/lib/learning/client";
import { learningAwait, validateLearningBytes, LearningInputError } from "@/lib/learning/bytes";
import { learningFailure, learningRouteRequest, learningUUID } from "@/lib/learning/schemas";
import type { LearningResult, LearningRoute } from "@/lib/learning/types";

export type PendingLearningCommand = { key: string; route: LearningRoute; input: Readonly<unknown>; actorId: string };
function freeze<T>(value: T): T {
    if (value !== null && typeof value === "object") {
        for (const item of Object.values(value)) freeze(item);
        Object.freeze(value);
    }
    return value;
}
export function createPendingLearningCommand(route: LearningRoute, input: unknown, actorId: string): PendingLearningCommand {
    if (!learningUUID.test(actorId) || learningRouteRequest(route)?.method !== "POST") throw new Error("Invalid learning command.");
    const raw = JSON.stringify(input);
    validateLearningBytes(new TextEncoder().encode(raw), route.kind);
    const value = freeze(JSON.parse(raw));
    bindLearningInput(value, actorId);
    return Object.freeze({ key: crypto.randomUUID(), route: freeze(structuredClone(route)), input: value, actorId });
}
export const pendingForActor = (command: PendingLearningCommand | null, actorId: string | null) => command?.actorId === actorId ? command : null;

export function useLearningCommand<T>(onSuccess: (value: T) => void) {
    const account = useContext(LearningAccountContext);
    const [pending, setPending] = useState<PendingLearningCommand | null>(null);
    const [error, setError] = useState<Extract<LearningResult<never>, { ok: false }> | null>(null);
    const [busy, setBusy] = useState(false);
    const inFlight = useRef(false), live = useRef(true), generation = useRef(0);
    const controller = useRef<AbortController | null>(null), success = useRef(onSuccess);
    success.current = onSuccess;
    const clear = () => {
        generation.current++;
        controller.current?.abort();
        inFlight.current = false;
        setPending(null);
        setError(null);
        setBusy(false);
    };
    useEffect(() => {
        live.current = true;
        window.addEventListener("math-master:auth-change", clear);
        return () => {
            live.current = false;
            generation.current++;
            controller.current?.abort();
            window.removeEventListener("math-master:auth-change", clear);
        };
    }, []);
    const active = (g: number) => live.current && generation.current === g;
    const perform = async (g: number, operation: (signal: AbortSignal) => Promise<void>) => {
        const current = new AbortController();
        controller.current = current;
        // This single deadline starts at the click, before identity preparation.
        const timer = setTimeout(() => current.abort(), 10000);
        try {
            await operation(current.signal);
        } catch (e) {
            if (active(g)) setError(learningFailure(e instanceof LearningInputError ? e.code : undefined));
        } finally {
            clearTimeout(timer);
            if (controller.current === current) controller.current = null;
            if (active(g)) {
                inFlight.current = false;
                setBusy(false);
            }
        }
    };
    const verifyActor = async (g: number, signal: AbortSignal) => {
        if (!account) { setError(learningFailure("FORBIDDEN")); return false; }
        const context = await learningAwait(getAuthContext(true), signal);
        if (!active(g)) return false;
        if (!context.ok) { setError(learningFailure(context.code)); return false; }
        if (context.data.user?.id !== account.actorId || context.data.user.mustChangePassword) {
            clear();
            account.invalidate();
            return false;
        }
        return true;
    };
    const send = async (command: PendingLearningCommand, g: number, signal: AbortSignal) => {
        if (!active(g)) return;
        setPending(command);
        setError(null);
        const result = await learningAwait(requestLearning<T>(command.route, command.input, command.key, signal), signal);
        if (!active(g)) return;
        if (result.ok) {
            setPending(null);
            success.current(result.data);
        } else {
            if (result.code === "FORBIDDEN" && !await verifyActor(g, signal)) return;
            if (result.code === "AUTHENTICATION_REQUIRED" || result.code === "PASSWORD_CHANGE_REQUIRED") {
                clear(); account?.invalidate(); return;
            }
            if (active(g)) setError(result);
        }
    };
    const run = async (route: LearningRoute, input: unknown) => {
        if (inFlight.current) return;
        if (!account) { setError(learningFailure("FORBIDDEN")); return; }
        inFlight.current = true;
        const g = ++generation.current;
        setBusy(true);
        setError(null);
        await perform(g, async signal => {
            if (!await verifyActor(g, signal)) return;
            await send(createPendingLearningCommand(route, input, account.actorId), g, signal);
        });
    };
    const retry = async () => {
        if (!pending || inFlight.current) return;
        const command = pending;
        if (!account || command.actorId !== account.actorId) { clear(); account?.invalidate(); return; }
        inFlight.current = true;
        setBusy(true);
        const g = ++generation.current;
        await perform(g, async signal => { if (await verifyActor(g, signal)) await send(command, g, signal); });
    };
    return { run, retry, clear, busy, pending, error };
}
