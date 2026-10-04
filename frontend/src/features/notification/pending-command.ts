"use client";
import { notificationClient } from '@/lib/notification/client';
import { uuidSchema } from '@/lib/correction/schemas';
import { CorrectionRequestError } from '@/lib/correction/types';
import type { ReadReceipt } from '@/lib/notification/types';
import { freezePrivate, usePrivatePending } from '../correction/pending-command';
export function createPendingNotificationCommand(actorId: string, notificationId: string) {
    try {
        uuidSchema.parse(actorId);
        uuidSchema.parse(notificationId);
        return freezePrivate({ actorId, notificationId, key: crypto.randomUUID(), input: {} });
    }
    catch {
        throw new CorrectionRequestError('INVALID_REQUEST');
    }
}
export function useNotificationCommand(actorId: string, onSuccess: (receipt: ReadReceipt, signal: AbortSignal) => Promise<void>) { return usePrivatePending(actorId, (id: string) => createPendingNotificationCommand(actorId, id), (pending, signal) => notificationClient.markRead(pending.notificationId, { actorId: pending.actorId, key: pending.key, signal }), onSuccess); }
