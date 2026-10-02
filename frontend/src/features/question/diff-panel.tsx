"use client";
import { useEffect, useState } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { ChangePage, MemberPage } from "@/lib/question/types";
import styles from "@/styles/question.module.css";
export function ChangeItems({ items }: {
    items: ChangePage["items"];
}) { return <ul className={styles.list}>{items.map((c, i) => <li key={i}><strong>{c.kind} · {c.id}</strong><p>{c.reason}</p><p className={styles.metadata}>Before: {c.before ? `v${c.before.version} · ${c.before.sha256}` : "None"}<br />After: {c.after ? `v${c.after.version} · ${c.after.sha256}` : "None"}</p></li>)}</ul>; }
export function DiffPanel({ publicationId, manifestSha }: {
    publicationId: string;
    manifestSha: string;
}) {
    const [changes, setChanges] = useState<ChangePage | null>(null), [members, setMembers] = useState<MemberPage | null>(null), [error, setError] = useState<string | null>(null), [busy, setBusy] = useState(false);
    useEffect(() => { const controller = new AbortController(); setChanges(null); setMembers(null); setError(null); setBusy(true); void Promise.all([requestQuestion<ChangePage>({ kind: "listChanges", id: publicationId, query: { limit: 20 } }, undefined, undefined, controller.signal), requestQuestion<MemberPage>({ kind: "listMembers", id: publicationId, query: { limit: 20 } }, undefined, undefined, controller.signal)]).then(([a, b]) => { if (controller.signal.aborted)
        return; if (!a.ok || !b.ok) {
        setError(!a.ok ? a.message : !b.ok ? b.message : "");
        return;
    } if (a.data.publicationId !== publicationId || b.data.publicationId !== publicationId || a.data.manifestSha !== manifestSha || b.data.manifestSha !== manifestSha) {
        setError("Snapshot data does not match the selected manifest.");
        return;
    } setChanges(a.data); setMembers(b.data); }).finally(() => { if (!controller.signal.aborted)
        setBusy(false); }); return () => controller.abort(); }, [publicationId, manifestSha]);
    async function load(kind: "listChanges" | "listMembers", offset: number) { if (busy)
        return; setBusy(true); try {
        const page = kind === "listChanges" ? changes : members;
        const r = await requestQuestion<ChangePage | MemberPage>({ kind, id: publicationId, query: { limit: page?.limit ?? 20, offset } });
        if (!r.ok) {
            setError(r.message);
            return;
        }
        if (r.data.publicationId !== publicationId || r.data.manifestSha !== manifestSha) {
            setChanges(null);
            setMembers(null);
            setError("Snapshot data does not match the selected manifest.");
            return;
        }
        if (kind === "listChanges")
            setChanges(r.data as ChangePage);
        else
            setMembers(r.data as MemberPage);
    }
    finally {
        setBusy(false);
    } }
    return <section><h3>Fixed mathematical differences</h3>{error && <p role="alert">{error}</p>}{busy && !changes && <p>Loading fixed differences…</p>}{changes && <><p>{changes.total} changes · Offset {changes.offset}</p><ChangeItems items={changes.items}/><div className={styles.actions}><button className="button secondary" disabled={busy || changes.offset === 0} onClick={() => void load("listChanges", Math.max(0, changes.offset - changes.limit))}>Previous changes</button><button className="button secondary" disabled={busy || changes.offset + changes.limit >= changes.total} onClick={() => void load("listChanges", changes.offset + changes.limit)}>Next changes</button></div></>}<h3>Fixed members and approval evidence</h3>{members && <><p>{members.total} members · Offset {members.offset}. Approval provenance can change without a mathematical replacement.</p><ul className={styles.list}>{members.items.map((m, i) => <li key={i}><strong>{m.identity.kind} · {m.identity.id} v{m.identity.version}</strong><p className={styles.metadata}>SHA {m.identity.sha256}<br />Package {m.identity.packageId} v{m.identity.packageVersion}<br />Submission {m.evidence.submissionId}<br />Decision {m.evidence.decisionId}<br />Frozen digest {m.evidence.frozenDigest}<br />Inherited from {m.evidence.inheritedFrom ?? "Newly selected approval"}</p></li>)}</ul><div className={styles.actions}><button className="button secondary" disabled={busy || members.offset === 0} onClick={() => void load("listMembers", Math.max(0, members.offset - members.limit))}>Previous members</button><button className="button secondary" disabled={busy || members.offset + members.limit >= members.total} onClick={() => void load("listMembers", members.offset + members.limit)}>Next members</button></div></>}</section>;
}
