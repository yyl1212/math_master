"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { contentRequest } from "@/lib/content/client";
import type { PublicationPage, PublicationView, SubmissionPage } from "@/lib/content/types";
import { validContentInput } from "@/lib/content/schemas";
import { TextField } from "./field-controls";
import { DiffPanel } from "./diff-panel";
import { useContentCommand } from "./command-controls";
import styles from "@/styles/content.module.css";
export function PublicationPanel({ initial, selectedID }: {
    initial: PublicationPage;
    selectedID?: string;
}) {
    const router = useRouter(), command = useContentCommand(), [page, setPage] = useState(initial), [selected, setSelected] = useState(selectedID ?? initial.items[0]?.id ?? ""), [approved, setApproved] = useState<SubmissionPage | null>(null), [submissionIDs, setIDs] = useState<string[]>([]), [reason, setReason] = useState(""), [stale, setStale] = useState(false);
    const candidate = page.items.find(v => v.id === selected);
    useEffect(() => { let live = true; void contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 100 } }).then(result => { if (!live)
        return; if (result.ok)
        setApproved(result.data);
    else
        command.error(result.message); }); return () => { live = false; }; }, []);
    useEffect(() => { if (command.failure?.code === "PUBLICATION_STALE")
        setStale(true); }, [command.failure]);
    async function refresh(offset = page.offset) { const result = await contentRequest<PublicationPage>({ kind: "listPublications", query: { limit: page.limit, offset } }); if (result.ok) {
        setPage(result.data);
        router.refresh();
    }
    else
        command.error(result.message); }
    return <section className={styles.workbench}><p className="eyebrow">PUBLICATION</p><h1>Reviewed publication snapshots</h1><p>{page.head ? "Current published head: " + page.head : "No content has been published yet."}</p><Link prefetch={false} href="/admin/withdrawals">Withdraw a fixed version</Link><div className={styles.actions}><button className="button secondary" disabled={command.busy} onClick={() => void refresh()}>Refresh current head</button></div>{command.controls}<div className={styles.grid}><section className={styles.card}><h2>Approved submissions</h2>{approved === null ? <p>Loading approved batches…</p> : approved.items.length === 0 ? <p>No approved submissions.</p> : approved.items.map(s => <label className={styles.check} key={s.id}><input type="checkbox" checked={submissionIDs.includes(s.id)} disabled={command.busy || submissionIDs.length >= 20 && !submissionIDs.includes(s.id)} onChange={e => setIDs(ids => e.target.checked ? [...ids, s.id] : ids.filter(v => v !== s.id))}/><span>{s.packageId} v{s.packageVersion} <Link prefetch={false} href={"/review/" + s.id}>Review record</Link></span></label>)}{approved && <div className={styles.actions}><button className="button secondary" disabled={command.busy || approved.offset === 0} onClick={async () => { const result = await contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 100, offset: Math.max(0, approved.offset - 100) } }); if (result.ok)
        setApproved(result.data);
    else
        command.error(result.message); }}>Previous approved batches</button><button className="button secondary" disabled={command.busy || approved.offset + approved.limit >= approved.total} onClick={async () => { const result = await contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 100, offset: approved.offset + 100 } }); if (result.ok)
        setApproved(result.data);
    else
        command.error(result.message); }}>Next approved batches</button></div>}<TextField label="Publication reason" value={reason} onChange={setReason} multiline/><button className="button" disabled={command.busy || !validContentInput("prepareRelease", { submissionIds: submissionIDs, expectedHead: page.head, reason })} onClick={() => void command.run({ kind: "prepareRelease" }, { submissionIds: submissionIDs, expectedHead: page.head, reason }, async (data) => { const next = data as PublicationView; await refresh(); setPage(v => ({ ...v, items: [next, ...v.items.filter(p => p.id !== next.id)], total: v.items.some(p => p.id === next.id) ? v.total : v.total + 1 })); setSelected(next.id); setStale(false); command.message("Snapshot prepared. Inspect the fixed difference before activation."); })}>Prepare snapshot</button></section><section className={styles.card}><h2>Snapshot history</h2>{page.items.length > 0 ? <label className={styles.field}>Selected snapshot<select value={selected} disabled={command.busy} onChange={e => { setSelected(e.target.value); setStale(false); command.clearPending(); }}>{page.items.map(p => <option key={p.id} value={p.id}>{p.status} · {p.id}</option>)}</select></label> : <p>No snapshots yet.</p>}<div className={styles.actions}><button className="button secondary" disabled={command.busy || page.offset === 0} onClick={() => void refresh(Math.max(0, page.offset - page.limit))}>Previous snapshots</button><button className="button secondary" disabled={command.busy || page.offset + page.limit >= page.total} onClick={() => void refresh(page.offset + page.limit)}>Next snapshots</button></div>{candidate && <><p>{candidate.status} · Catalogue version {candidate.manifest.catalogueVersion}</p><p className={styles.metadata}>Manifest SHA: {candidate.manifestSha}</p><DiffPanel diff={candidate.diff}/><details><summary>Fixed manifest and review evidence</summary><pre>{JSON.stringify(candidate.manifest, null, 2)}</pre></details>{stale && <p>Published content changed. Prepare a new snapshot.</p>}<button className="button" disabled={command.busy || candidate.status !== "draft" || stale || candidate.manifest.baseHead !== page.head || !validContentInput("activateRelease", { expectedHead: page.head, expectedManifestSha: candidate.manifestSha, reason })} onClick={() => void command.run({ kind: "activateRelease", id: candidate.id }, { expectedHead: page.head, expectedManifestSha: candidate.manifestSha, reason }, async () => { await refresh(); command.message("Snapshot activated. Current head refreshed."); })}>Activate snapshot</button></>}</section></div></section>;
}
