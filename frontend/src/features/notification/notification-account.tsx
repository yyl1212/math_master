"use client";
import { CorrectionAccountProvider } from '../correction/correction-account';
export function NotificationAccountProvider({ actorId, children }: {
    actorId: string;
    children: React.ReactNode;
}) { return <CorrectionAccountProvider actorId={actorId}>{children}</CorrectionAccountProvider>; }
