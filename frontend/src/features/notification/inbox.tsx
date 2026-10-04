"use client";
import Link from 'next/link';
import { useContext, useState, useEffect } from 'react';
import { notificationClient } from '@/lib/notification/client';
import { CorrectionRequestError, type Page } from '@/lib/correction/types';
import type { Metadata } from '@/lib/notification/types';
import { CorrectionAccountContext, useCorrectionRead } from '../correction/correction-account';
import { CorrectionState, CorrectionCommandStatus } from '../correction/status';
import { useNotificationCommand } from './pending-command';
import styles from '@/styles/content.module.css';
const text = { checking: 'Your learning evidence is being checked.', corrected: 'A corrected result is ready to review.', retake: 'A new assessment is required.', review_material: 'Learning material needs another review.', path_unavailable: 'A learning path is currently unavailable.' } satisfies Record<Metadata['type'], string>;
export function Inbox({ initial, count }: {
    initial: Page<Metadata>;
    count: number;
}) {
    const account = useContext(CorrectionAccountContext), read = useCorrectionRead(), [page, setPage] = useState(initial), [unread, setUnread] = useState(count), [loading, setLoading] = useState(false), [error, setError] = useState<CorrectionRequestError | null>(null);
    useEffect(() => { setPage(initial); setUnread(count); }, [initial, count]);
    const command = useNotificationCommand(account?.actorId ?? '', async (receipt, signal) => {
        const out = await read(async (a) => {
            const [count, note] = await Promise.all([notificationClient.count(a), notificationClient.read(receipt.notificationId, a)]);
            if (count.actorId !== note.actorId)
                throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
            return { actorId: count.actorId, data: { count: count.data.count, note: note.data } };
        }, signal);
        if (!out || signal.aborted)
            throw new CorrectionRequestError();
        setUnread(out.data.count);
        setPage(v => ({ ...v, items: v.items.map(i => i.id === out.data.note.id ? out.data.note : i) }));
    });
    async function reload(cursor?: string) {
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            const out = await read(async (a) => {
                const [page, count] = await Promise.all([notificationClient.list(cursor ? { cursor } : {}, a), notificationClient.count(a)]);
                if (page.actorId !== count.actorId)
                    throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
                return { actorId: page.actorId, data: { page: page.data, count: count.data.count } };
            });
            if (out) {
                setPage(out.data.page);
                setUnread(out.data.count);
            }
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
        }
        finally {
            setLoading(false);
        }
    }
    return <main className={styles.workbench}><h1>Notifications</h1><p aria-live="polite">{unread} unread notifications</p><button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void reload()}>Reload notifications</button>{error && <CorrectionState error={error} onRetry={() => void reload()}/>}<ul className={styles.list}>{page.items.map(note => <li key={note.id}><h2>{text[note.type]}</h2><p>{note.readAt ? 'Read' : 'Unread'} · <time dateTime={note.createdAt}>{new Date(note.createdAt).toLocaleString('en')}</time></p>{note.resultId && <Link prefetch={false} href={'/corrections/' + note.resultId}>View correction</Link>}{!note.readAt && <button type="button" className="button secondary" disabled={command.busy || !!command.pending} onClick={() => void command.run(note.id)}>Mark as read</button>}</li>)}</ul>{page.items.length === 0 && <p>You have no notifications yet.</p>}{page.nextCursor && <button type="button" disabled={loading || account?.checking} onClick={() => void reload(page.nextCursor!)}>Next notifications</button>}<CorrectionCommandStatus command={command}/></main>;
}
