"use client";
import Link from "next/link";
import { useState } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { SubmissionPage } from "@/lib/question/types";
import { FormMessage } from "@/features/auth/auth-state";
import styles from "@/styles/question.module.css";
export function SubmissionList({ initial, scope }: {
    initial: SubmissionPage;
    scope: "mine" | "review" | "all";
}) { const [page, setPage] = useState(initial), [status, setStatus] = useState<"pending" | "approved" | "returned">("pending"), [error, setError] = useState<string | null>(null), [busy, setBusy] = useState(false); async function load(offset: number, next = status) { if (busy)
    return; setBusy(true); try {
    const r = await requestQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: next, limit: page.limit, offset } });
    if (r.ok) {
        setPage(r.data);
        setStatus(next);
        setError(null);
    }
    else
        setError(r.message);
}
finally {
    setBusy(false);
} } return <section className={styles.workbench}><h2>{scope === "review" ? "Independent question review queue" : "Frozen question submissions"}</h2><p>Review uses immutable submitted questions and every bound instance. Authors are excluded from the independent queue.</p><FormMessage message={error} error/><label>Status <select value={status} disabled={busy} onChange={e => void load(0, e.target.value as typeof status)}><option value="pending">Pending</option><option value="approved">Approved</option><option value="returned">Returned</option></select></label><ul className={styles.list}>{page.items.map(s => <li key={s.id}><Link prefetch={false} href={"/review/questions/" + s.id}>{s.packageId} v{s.packageVersion}</Link><p>{s.status} · Frozen revision {s.revision}</p><p className={styles.metadata}>Digest: {s.frozenDigest}</p></li>)}</ul>{!page.items.length && <p>No submissions in this view.</p>}<div className={styles.actions}><button className="button secondary" disabled={busy || page.offset === 0} onClick={() => void load(Math.max(0, page.offset - page.limit))}>Previous submissions</button><button className="button secondary" disabled={busy || page.offset + page.limit >= page.total} onClick={() => void load(page.offset + page.limit)}>Next submissions</button></div></section>; }
