"use client";
import Link from "next/link";
import { useState } from "react";
import { contentRequest } from "@/lib/content/client";
import type { SubmissionPage } from "@/lib/content/types";
import styles from "@/styles/content.module.css";
import { FormMessage } from "@/features/auth/auth-state";
export function SubmissionList({ initial, scope }: {
    initial: SubmissionPage;
    scope: "mine" | "review" | "all";
}) {
    const [page, setPage] = useState(initial), [status, setStatus] = useState("pending"), [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null);
    async function load(next: string, offset = 0) { if (busy)
        return; setBusy(true); try {
        const result = await contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: next as "pending", limit: page.limit, offset } });
        if (result.ok) {
            setPage(result.data);
            setStatus(next);
            setError(null);
        }
        else
            setError(result.message);
    }
    finally {
        setBusy(false);
    } }
    return <section className={styles.workbench}><h1>{scope === "review" ? "Independent review queue" : "Submissions"}</h1><p>Each submission fixes the entire batch, sources, authors and illustration bytes.</p><label className={styles.field}>Submission status<select value={status} disabled={busy} onChange={e => void load(e.target.value)}>{["pending", "approved", "returned"].map(v => <option key={v}>{v}</option>)}</select></label><FormMessage message={error} error/><ul className={styles.list}>{page.items.map(s => <li key={s.id}><Link prefetch={false} href={"/review/" + s.id}>{s.packageId} · Version {s.packageVersion}</Link><p>{s.status} · Revision {s.revision}</p><p className={styles.metadata}>Frozen digest: {s.frozenDigest}</p></li>)}</ul>{!page.items.length && <p>No matching submissions.</p>}<div className={styles.actions}><button className="button secondary" disabled={busy || !page.offset} onClick={() => void load(status, Math.max(0, page.offset - page.limit))}>Previous</button><button className="button secondary" disabled={busy || page.offset + page.limit >= page.total} onClick={() => void load(status, page.offset + page.limit)}>Next</button></div></section>;
}
