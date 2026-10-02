"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { requestQuestion } from "@/lib/question/client";
import { contentRequest } from "@/lib/content/client";
import type { PublicationPage as ContentPublicationPage } from "@/lib/content/types";
import type { PublicationPage, PublicationSummary, SubmissionPage } from "@/lib/question/types";
import { questionInputError } from "@/lib/question/schemas";
import { TextField } from "@/features/content/field-controls";
import { useQuestionCommand } from "./command-controls";
import { DiffPanel } from "./diff-panel";
import styles from "@/styles/question.module.css";
export function PublicationPanel({ initial, knowledgeHead, selectedPublication }: {
    initial: PublicationPage;
    knowledgeHead: string | null;
    selectedPublication?: PublicationSummary;
}) {
    const [page, setPage] = useState(initial), [kHead, setKHead] = useState(knowledgeHead), [head, setHead] = useState<PublicationSummary | null>(null), [candidate, setCandidate] = useState<PublicationSummary | null>(selectedPublication ?? initial.items.find(p => p.status === "prepared") ?? initial.items[0] ?? null), [approved, setApproved] = useState<SubmissionPage | null>(null), [ids, setIDs] = useState<string[]>([]), [reason, setReason] = useState(""), [stale, setStale] = useState(false), [busy, setBusy] = useState(false), command = useQuestionCommand(), router = useRouter();
    useEffect(() => { const controller = new AbortController(); setHead(null); if (page.head)
        void requestQuestion<PublicationSummary>({ kind: "readPublication", id: page.head }, undefined, undefined, controller.signal).then(r => { if (controller.signal.aborted)
            return; if (r.ok && r.data.id === page.head)
            setHead(r.data);
        else
            command.error(r.ok ? "Current snapshot identity does not match." : r.message); }); return () => controller.abort(); }, [page.head]);
    useEffect(() => { const controller = new AbortController(); void requestQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 20 } }, undefined, undefined, controller.signal).then(r => { if (controller.signal.aborted)
        return; if (r.ok)
        setApproved(r.data);
    else
        command.error(r.message); }); return () => controller.abort(); }, []);
    useEffect(() => { if (command.failure?.code === "QUESTION_PUBLICATION_STALE")
        setStale(true); }, [command.failure]);
    async function refresh(offset = page.offset) { if (busy)
        return; setBusy(true); try {
        const [q, k] = await Promise.all([requestQuestion<PublicationPage>({ kind: "listPublications", query: { limit: page.limit, offset } }), contentRequest<ContentPublicationPage>({ kind: "listPublications", query: { limit: 1 } })]);
        if (!q.ok || !k.ok) {
            setStale(true);
            command.error(!q.ok ? q.message : !k.ok ? k.message : "");
            return;
        }
        setPage(q.data);
        setKHead(k.data.head);
        if (candidate && (candidate.baseKnowledgeHead !== k.data.head || candidate.baseQuestionHead !== q.data.head))
            setStale(true);
        router.refresh();
    }
    finally {
        setBusy(false);
    } }
    async function loadApproved(offset: number) { if (busy || !approved)
        return; setBusy(true); try {
        const r = await requestQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: approved.limit, offset } });
        if (r.ok)
            setApproved(r.data);
        else
            command.error(r.message);
    }
    finally {
        setBusy(false);
    } }
    const prepare = { submissionIds: ids, expectedKnowledgeHead: kHead, expectedQuestionHead: page.head, reason }, activate = { expectedKnowledgeHead: kHead, expectedQuestionHead: page.head, expectedManifestSha: candidate?.manifestSha ?? "", reason };
    return <section className={styles.workbench}><p className="eyebrow">QUESTION PUBLICATION</p><h1>Trusted question snapshots</h1><p className={styles.metadata}>Current knowledge head: {kHead ?? "None"}<br />Current question head: {page.head ?? "None"}</p>{head && <p>Current published bank: {head.templateCount} templates · {head.instanceCount} trusted instances · {head.blueprintCount} blueprints.</p>}<p><Link prefetch={false} href="/admin/question-withdrawals">Withdraw a question version permanently</Link></p><button className="button secondary" disabled={busy || command.blocked} onClick={() => void refresh()}>Refresh both current heads</button>{command.controls}<fieldset disabled={command.blocked}><legend>Publication reason</legend><TextField label="Publication reason" value={reason} onChange={setReason} multiline/></fieldset><div className={styles.grid}><section className={styles.card}><h2>Approved submissions</h2><p>Select 1–20 independently approved submissions. Inherited members retain their fixed evidence.</p>{!approved ? <p>Loading approved submissions…</p> : approved.items.length === 0 ? <p>No approved submissions.</p> : approved.items.map(s => <label className={styles.check} key={s.id}><input type="checkbox" checked={ids.includes(s.id)} disabled={command.blocked || ids.length >= 20 && !ids.includes(s.id)} onChange={e => setIDs(v => e.target.checked ? [...v, s.id] : v.filter(id => id !== s.id))}/><span>{s.packageId} v{s.packageVersion} <Link prefetch={false} href={"/review/questions/" + s.id}>Frozen review</Link></span></label>)}{approved && <div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || approved.offset === 0} onClick={() => void loadApproved(Math.max(0, approved.offset - approved.limit))}>Previous approved submissions</button><button className="button secondary" disabled={busy || command.blocked || approved.offset + approved.limit >= approved.total} onClick={() => void loadApproved(approved.offset + approved.limit)}>Next approved submissions</button></div>}<p>{ids.length} submissions selected across pages.</p><button className="button" disabled={busy || command.blocked || questionInputError("prepareRelease", prepare) !== null} onClick={() => void command.run({ kind: "prepareRelease" }, prepare, data => { const next = data as PublicationSummary; setCandidate(next); setStale(false); command.message("Snapshot prepared. Inspect its complete fixed difference and evidence before activation."); })}>Prepare snapshot</button></section><section className={styles.card}><h2>Snapshot history</h2>{page.items.length === 0 ? <p>No snapshots on this page.</p> : <ul className={styles.list}>{page.items.map(p => <li key={p.id}><button className="button secondary" disabled={command.blocked} onClick={() => { setCandidate(p); setStale(false); }}>{p.status} · {p.id}</button><p><Link prefetch={false} href={"/admin/question-publications/" + p.id}>Open fixed snapshot</Link></p></li>)}</ul>}<div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || page.offset === 0} onClick={() => void refresh(Math.max(0, page.offset - page.limit))}>Previous snapshots</button><button className="button secondary" disabled={busy || command.blocked || page.offset + page.limit >= page.total} onClick={() => void refresh(page.offset + page.limit)}>Next snapshots</button></div></section></div>{candidate && <section className={styles.card}><h2>Selected {candidate.status} snapshot</h2><p className={styles.metadata}>{candidate.id}<br />Manifest SHA: {candidate.manifestSha}<br />Base knowledge head: {candidate.baseKnowledgeHead ?? "None"}<br />Base question head: {candidate.baseQuestionHead ?? "None"}</p><p>{candidate.templateCount} templates · {candidate.instanceCount} instances · {candidate.blueprintCount} blueprints</p><p>Added: {candidate.diff.added} · Replaced: {candidate.diff.replaced} · Removed: {candidate.diff.removed}</p><DiffPanel publicationId={candidate.id} manifestSha={candidate.manifestSha}/>{(candidate.status === "prepared" && (stale || candidate.baseKnowledgeHead !== kHead || candidate.baseQuestionHead !== page.head)) && <p role="status">Published snapshots changed. Refresh both heads and prepare a new snapshot.</p>}<button className="button" disabled={busy || command.blocked || stale || candidate.status !== "prepared" || candidate.baseKnowledgeHead !== kHead || candidate.baseQuestionHead !== page.head || questionInputError("activateRelease", activate) !== null} onClick={() => void command.run({ kind: "activateRelease", id: candidate.id }, activate, async (data) => { await refresh(); setCandidate(data as PublicationSummary); setStale(false); command.message("Snapshot activated. Both current heads refreshed."); })}>Activate snapshot</button></section>}</section>;
}
